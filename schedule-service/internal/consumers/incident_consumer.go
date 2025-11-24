package consumers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ValidationResult representa un incidente validado desde validation-service
type ValidationResult struct {
	IncidentID  string  `json:"incident_id"` // Corregido: validation-service usa incident_id
	Tipo        string  `json:"tipo"`
	Latitud     float64 `json:"latitud"`
	Longitud    float64 `json:"longitud"`
	Status      string  `json:"status"`    // Corregido: es "status" no "estado"
	Validator   string  `json:"validator"` // Corregido: es "validator" no "operador"
	ValidatedAt string  `json:"validated_at"`
	Notes       string  `json:"notes"`
}

// IncidentConsumer consume eventos incidents.validated.v1 y llama ProcessIncident
type IncidentConsumer struct {
	conn            *amqp.Connection
	planningService *services.PlanningService
}

func NewIncidentConsumer(conn *amqp.Connection, planningService *services.PlanningService) (*IncidentConsumer, error) {
	return &IncidentConsumer{
		conn:            conn,
		planningService: planningService,
	}, nil
}

// Start inicia el consumer que escucha incidents.validated.v1
func (ic *IncidentConsumer) Start() error {
	ch, err := ic.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declarar exchange (debe coincidir con incident-service)
	exchangeName := "city.cleaning.incidents"
	if err := ch.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declarar cola para scheduler
	queueName := "q.scheduler.incidents"
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue a routing key incidents.validated.v1
	routingKey := "incidents.validated.v1"
	if err := ch.QueueBind(
		q.Name,       // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Iniciar consumo
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack (false para ack manual)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("✅ IncidentConsumer listening on '%s' for routing key '%s'", queueName, routingKey)

	// Procesar mensajes en goroutine
	go func() {
		for msg := range msgs {
			shouldRequeue, err := ic.handleMessage(msg)
			if err != nil {
				log.Printf("❌ Error processing incident: %v", err)
				if shouldRequeue {
					msg.Nack(false, true) // Requeue for transient errors
				} else {
					msg.Ack(false) // ACK to discard permanently (irrecoverable error)
				}
			} else {
				msg.Ack(false) // Acknowledge success
			}
		}
	}()

	return nil
}

// handleMessage procesa un mensaje de incident validado
// Retorna (shouldRequeue, error) - shouldRequeue=false para errores permanentes
func (ic *IncidentConsumer) handleMessage(msg amqp.Delivery) (bool, error) {
	var result ValidationResult
	if err := json.Unmarshal(msg.Body, &result); err != nil {
		// Error de parseo: no reencolar (mensaje corrupto)
		return false, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	log.Printf("📥 Received validated incident: ID=%s Type=%s Status=%s (Lat: %.6f, Lon: %.6f)",
		result.IncidentID, result.Tipo, result.Status, result.Latitud, result.Longitud)

	// Solo procesar incidentes con status incidente_valido
	if result.Status != "incidente_valido" {
		log.Printf("⏭️  Skipping incident %s (Status: %s)", result.IncidentID, result.Status)
		return false, nil // No error, simplemente descartamos
	}

	// Llamar a ProcessIncident del planning service
	planningResult, err := ic.planningService.ProcessIncident(
		result.Latitud,
		result.Longitud,
		result.Tipo,
	)
	if err != nil {
		// Verificar si es un error de "zona no encontrada" (irrecuperable)
		if strings.Contains(err.Error(), "zona no encontrada") {
			log.Printf("⚠️  Discarding incident %s: coordinates outside all zones (%.6f, %.6f)",
				result.IncidentID, result.Latitud, result.Longitud)
			return false, fmt.Errorf("zona no encontrada para coordenadas (%.6f, %.6f)", result.Latitud, result.Longitud)
		}
		// Otros errores: reencolar (ej: BD temporalmente caída)
		return true, fmt.Errorf("failed to process incident: %w", err)
	}

	log.Printf("✅ Incident processed: Zone=%s (%d) Score=%d/%d Triggered=%v",
		planningResult.ZoneName,
		planningResult.ZoneID,
		planningResult.NewScore,
		planningResult.Threshold,
		planningResult.Triggered,
	)

	return false, nil
}
