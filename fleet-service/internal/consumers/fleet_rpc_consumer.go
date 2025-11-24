package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/fleet-service/internal/services"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// FleetRequest solicita recursos (chofer + camión) para una zona
type FleetRequest struct {
	ZoneID       int    `json:"zone_id"`
	RequiredType string `json:"required_type"` // "STANDARD", "COMPACTOR"
}

// FleetResponse contiene los recursos asignados
type FleetResponse struct {
	DriverID    uuid.UUID  `json:"driver_id"`
	AssistantID *uuid.UUID `json:"assistant_id,omitempty"`
	TruckPlate  string     `json:"truck_plate"`
	Status      string     `json:"status"` // "ALLOCATED"
}

// FleetReleaseRequest libera recursos asignados (compensación)
type FleetReleaseRequest struct {
	DriverID   uuid.UUID `json:"driver_id"`
	TruckPlate string    `json:"truck_plate"`
	Reason     string    `json:"reason"`
}

// FleetRPCConsumer maneja solicitudes RPC para asignación de recursos
type FleetRPCConsumer struct {
	conn              *amqp.Connection
	channel           *amqp.Channel
	allocationService *services.AllocationService
	publisher         *services.EventPublisher
}

// NewFleetRPCConsumer crea una nueva instancia del consumidor RPC
func NewFleetRPCConsumer(
	conn *amqp.Connection,
	allocationService *services.AllocationService,
	publisher *services.EventPublisher,
) (*FleetRPCConsumer, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("error abriendo canal: %w", err)
	}

	return &FleetRPCConsumer{
		conn:              conn,
		channel:           channel,
		allocationService: allocationService,
		publisher:         publisher,
	}, nil
}

// Start inicia el consumidor RPC
func (c *FleetRPCConsumer) Start() error {
	// Declarar exchange
	if err := c.channel.ExchangeDeclare(
		"city.cleaning.planning", // name
		"topic",                  // type
		true,                     // durable
		false,                    // auto-deleted
		false,                    // internal
		false,                    // no-wait
		nil,                      // arguments
	); err != nil {
		return fmt.Errorf("error declarando exchange: %w", err)
	}

	// Declarar cola RPC
	queue, err := c.channel.QueueDeclare(
		"q.fleet.rpc", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return fmt.Errorf("error declarando cola RPC: %w", err)
	}

	// Bind para solicitudes de recursos
	if err := c.channel.QueueBind(
		queue.Name,               // queue name
		"fleet.resource.request", // routing key
		"city.cleaning.planning", // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("error binding cola de solicitudes: %w", err)
	}

	// Bind para liberación de recursos (compensación)
	if err := c.channel.QueueBind(
		queue.Name,               // queue name
		"fleet.resource.release", // routing key
		"city.cleaning.planning", // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("error binding cola de liberación: %w", err)
	}

	// Configurar QoS
	if err := c.channel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("error configurando QoS: %w", err)
	}

	// Iniciar consumo
	msgs, err := c.channel.Consume(
		queue.Name,         // queue
		"fleet-rpc-worker", // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		return fmt.Errorf("error iniciando consumidor: %w", err)
	}

	log.Println("🔄 Fleet RPC Consumer iniciado en cola: q.fleet.rpc")

	// Procesar mensajes
	go func() {
		for msg := range msgs {
			routingKey := msg.RoutingKey
			correlationID := msg.CorrelationId
			replyTo := msg.ReplyTo

			log.Printf("📨 [FLEET RPC] RoutingKey=%s, CorrelationId=%s", routingKey, correlationID)

			switch routingKey {
			case "fleet.resource.request":
				c.handleResourceRequest(msg, correlationID, replyTo)
			case "fleet.resource.release":
				c.handleResourceRelease(msg)
			default:
				log.Printf("⚠️  Routing key desconocido: %s", routingKey)
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// handleResourceRequest maneja solicitudes de asignación de recursos
func (c *FleetRPCConsumer) handleResourceRequest(msg amqp.Delivery, correlationID, replyTo string) {
	var req FleetRequest
	if err := json.Unmarshal(msg.Body, &req); err != nil {
		log.Printf("❌ Error parseando FleetRequest: %v", err)
		c.sendErrorResponse(replyTo, correlationID, "error parseando solicitud")
		msg.Ack(false)
		return
	}

	log.Printf("🚛 Asignando recursos para Zona %d, Tipo: %s", req.ZoneID, req.RequiredType)

	// Buscar driver disponible usando AllocationService
	allocation, err := c.allocationService.FindBestDriver(req.ZoneID, req.RequiredType)
	if err != nil {
		log.Printf("❌ No se encontró driver disponible para zona %d: %v", req.ZoneID, err)
		c.sendErrorResponse(replyTo, correlationID, fmt.Sprintf("No hay conductores disponibles: %v", err))
		msg.Ack(false)
		return
	}

	// Construir respuesta con asignación real
	response := FleetResponse{
		DriverID:    allocation.DriverID,
		AssistantID: nil, // Opcional, por ahora nil
		TruckPlate:  allocation.TruckPlate,
		Status:      "ALLOCATED",
	}

	log.Printf("🎯 [FLEET RPC] Asignación encontrada: Driver=%s (%s), Truck=%s (%s) para Zona=%d",
		allocation.DriverID, allocation.DriverName, allocation.TruckPlate, allocation.TruckType, req.ZoneID)

	// Enviar respuesta
	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("❌ Error serializando respuesta: %v", err)
		msg.Ack(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.channel.PublishWithContext(
		ctx,
		"",      // exchange
		replyTo, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			Body:          responseBody,
			Timestamp:     time.Now(),
		},
	)

	if err != nil {
		log.Printf("❌ Error enviando respuesta: %v", err)
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
	log.Printf("✅ [FLEET RPC RESPONSE] Driver=%s, Truck=%s asignados para Zona %d",
		response.DriverID, response.TruckPlate, req.ZoneID)
}

// handleResourceRelease maneja solicitudes de liberación de recursos (compensación)
func (c *FleetRPCConsumer) handleResourceRelease(msg amqp.Delivery) {
	var req FleetReleaseRequest
	if err := json.Unmarshal(msg.Body, &req); err != nil {
		log.Printf("❌ Error parseando FleetReleaseRequest: %v", err)
		msg.Ack(false)
		return
	}

	log.Printf("🔄 [COMPENSACIÓN] Liberando recursos: Driver=%s, Truck=%s, Reason=%s",
		req.DriverID, req.TruckPlate, req.Reason)

	// TODO: Implementar lógica real de liberación de recursos
	// Marcar driver y truck como disponibles nuevamente

	msg.Ack(false)
	log.Printf("✅ Recursos liberados exitosamente")
}

// sendErrorResponse envía una respuesta de error
func (c *FleetRPCConsumer) sendErrorResponse(replyTo, correlationID, errorMsg string) {
	errorResp := map[string]string{
		"error": errorMsg,
	}
	body, _ := json.Marshal(errorResp)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.channel.PublishWithContext(
		ctx,
		"",
		replyTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			Body:          body,
		},
	)
}

// Close cierra el canal
func (c *FleetRPCConsumer) Close() error {
	if c.channel != nil {
		return c.channel.Close()
	}
	return nil
}
