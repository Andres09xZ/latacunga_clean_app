package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	rabbitMQConn *amqp.Connection
	connMutex    sync.Mutex
)

// InitRabbitMQ initializes RabbitMQ connection and declares exchange/queues
func InitRabbitMQ(rabbitMQURL string) error {
	connMutex.Lock()
	defer connMutex.Unlock()

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	rabbitMQConn = conn
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchange
	if err := ch.ExchangeDeclare(
		"incidentes", // name (unified with incident-service)
		"topic",      // kind
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue for consuming incident pending events
	q, err := ch.QueueDeclare(
		"validation.q", // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange with routing key for incident.incidente_pendiente
	if err := ch.QueueBind(
		q.Name,                          // queue name
		"incidente.incidente_pendiente", // routing key for pending incidents
		"incidentes",                    // exchange name
		false,                           // no-wait
		nil,                             // arguments
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Println("✅ RabbitMQ initialized - Exchange 'incidentes' and queue 'validation.q' declared")
	return nil
}

// PublishEvent publishes a validation event to RabbitMQ
func PublishEvent(eventType string, payload interface{}) error {
	connMutex.Lock()
	defer connMutex.Unlock()

	if rabbitMQConn == nil {
		log.Println("⚠️  RabbitMQ not connected, skipping event publication")
		return fmt.Errorf("rabbitmq not connected")
	}

	ch, err := rabbitMQConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Create payload with timestamp
	eventPayload := map[string]interface{}{
		"type":      eventType,
		"timestamp": time.Now().UTC(),
		"data":      payload,
	}

	body, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Construct routing key: validacion.{eventType}
	routingKey := fmt.Sprintf("validacion.%s", eventType)

	err = ch.Publish(
		"incidentes", // exchange (unified with incident-service)
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("📤 Event published: %s -> %s", routingKey, string(body))
	return nil
}

// IncidentCreatedEvent structure from incident-service
type IncidentCreatedEvent struct {
	ID             string    `json:"id"`
	ReporterKind   string    `json:"reporter_kind"`
	ReporterID     string    `json:"reporter_id"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	IncidentDay    string    `json:"incident_day"`
	PhotosCount    int       `json:"photos_count"`
	Latitude       *float64  `json:"latitude,omitempty"`
	Longitude      *float64  `json:"longitude,omitempty"`
	CreatedAt      string    `json:"created_at"`
	EventTimestamp time.Time `json:"timestamp"`
}

// ConsumeIncidentCreatedEvents starts consuming incident creation events
func ConsumeIncidentCreatedEvents(handler func(*IncidentCreatedEvent) error) error {
	connMutex.Lock()
	defer connMutex.Unlock()

	if rabbitMQConn == nil {
		return fmt.Errorf("rabbitmq not connected")
	}

	ch, err := rabbitMQConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	// Don't defer close - keep channel open for consumption

	msgs, err := ch.Consume(
		"validation.q", // queue
		"",             // consumer
		false,          // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Println("👂 Listening for incident creation events...")

	go func() {
		for delivery := range msgs {
			// Parse event payload
			var eventData map[string]interface{}
			if err := json.Unmarshal(delivery.Body, &eventData); err != nil {
				log.Printf("❌ Failed to unmarshal event: %v", err)
				delivery.Nack(false, true) // Requeue on error
				continue
			}

			// Extract data field
			dataRaw, ok := eventData["data"]
			if !ok {
				log.Printf("❌ Missing 'data' field in event")
				delivery.Nack(false, true)
				continue
			}

			// Convert to IncidentCreatedEvent
			dataJSON, err := json.Marshal(dataRaw)
			if err != nil {
				log.Printf("❌ Failed to marshal data field: %v", err)
				delivery.Nack(false, true)
				continue
			}

			var incident IncidentCreatedEvent
			if err := json.Unmarshal(dataJSON, &incident); err != nil {
				log.Printf("❌ Failed to unmarshal incident event: %v", err)
				delivery.Nack(false, true)
				continue
			}

			// Call handler
			if err := handler(&incident); err != nil {
				log.Printf("❌ Handler error for incident %s: %v", incident.ID, err)
				delivery.Nack(false, true) // Requeue on error
				continue
			}

			// Acknowledge delivery
			delivery.Ack(false)
			log.Printf("✅ Processed incident event: %s", incident.ID)
		}
	}()

	return nil
}

// CloseRabbitMQ closes RabbitMQ connection
func CloseRabbitMQ() {
	connMutex.Lock()
	defer connMutex.Unlock()

	if rabbitMQConn != nil && !rabbitMQConn.IsClosed() {
		rabbitMQConn.Close()
		log.Println("✅ RabbitMQ connection closed")
	}
}
