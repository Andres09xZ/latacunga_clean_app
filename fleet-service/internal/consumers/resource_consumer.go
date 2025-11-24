package consumers

import (
	"encoding/json"
	"log"
	"time"

	"github.com/fleet-service/internal/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ResourceConsumer maneja las solicitudes de recursos desde Scheduler
type ResourceConsumer struct {
	channel           *amqp.Channel
	queueName         string
	allocationService *services.AllocationService
	publisher         *services.EventPublisher
}

// ResourceRequestedEvent representa una solicitud de recurso
type ResourceRequestedEvent struct {
	RequestID string `json:"request_id"`
	ZoneID    int    `json:"zone_id"`
	TruckType string `json:"truck_type"` // "CARGA_LATERAL" o "CARGA_POSTERIOR"
	RouteID   string `json:"route_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

// NewResourceConsumer crea un nuevo consumidor de recursos
func NewResourceConsumer(
	conn *amqp.Connection,
	queueName, exchange string,
	allocationService *services.AllocationService,
	publisher *services.EventPublisher,
) (*ResourceConsumer, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declarar la cola
	_, err = channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, err
	}

	// Bind a routing key
	err = channel.QueueBind(
		queueName,                        // queue name
		"planning.resource.requested.v1", // routing key
		exchange,                         // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("✓ ResourceConsumer configurado - Queue: %s", queueName)

	return &ResourceConsumer{
		channel:           channel,
		queueName:         queueName,
		allocationService: allocationService,
		publisher:         publisher,
	}, nil
}

// Start inicia el consumo de mensajes
func (c *ResourceConsumer) Start() error {
	msgs, err := c.channel.Consume(
		c.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	log.Printf("🟢 ResourceConsumer escuchando en cola: %s", c.queueName)

	go func() {
		for msg := range msgs {
			c.handleMessage(msg)
		}
	}()

	return nil
}

// handleMessage procesa cada solicitud de recurso
func (c *ResourceConsumer) handleMessage(msg amqp.Delivery) {
	log.Printf("📥 [Resource] Mensaje recibido: %s", msg.RoutingKey)

	var event ResourceRequestedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Error al parsear evento: %v", err)
		msg.Nack(false, false) // No requeue
		return
	}

	log.Printf("🔍 Buscando conductor para Zona: %d, Tipo: %s, RequestID: %s",
		event.ZoneID, event.TruckType, event.RequestID)

	// Llamar al motor de asignación
	allocation, err := c.allocationService.FindBestDriver(event.ZoneID, event.TruckType)
	if err != nil {
		log.Printf("❌ No se pudo asignar conductor: %v", err)
		// TODO: Publicar evento de fallo si es necesario
		msg.Ack(false) // Ack para no reenviar (ya no hay conductores)
		return
	}

	// Publicar evento de asignación exitosa
	allocatedEvent := services.DriverAllocatedEvent{
		DriverID:   allocation.DriverID.String(),
		DriverName: allocation.DriverName,
		TruckID:    allocation.TruckID,
		TruckPlate: allocation.TruckPlate,
		TruckType:  allocation.TruckType,
		ShiftID:    allocation.ShiftID.String(),
		ZoneID:     event.ZoneID,
		RequestID:  event.RequestID,
		Timestamp:  time.Now(),
	}

	if err := c.publisher.PublishDriverAllocated(allocatedEvent); err != nil {
		log.Printf("❌ Error al publicar evento de asignación: %v", err)
		msg.Nack(false, true) // Requeue
		return
	}

	log.Printf("✅ Conductor asignado exitosamente: %s -> Zona %d (Request: %s)",
		allocation.DriverName, event.ZoneID, event.RequestID)
	msg.Ack(false)
}

// Close cierra el canal
func (c *ResourceConsumer) Close() error {
	return c.channel.Close()
}
