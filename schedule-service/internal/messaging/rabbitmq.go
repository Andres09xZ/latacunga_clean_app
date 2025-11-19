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
	rabbitConn *amqp.Connection
	connMutex  sync.Mutex
)

// InitRabbitMQ initializes RabbitMQ connection and declares exchange/queues
func InitRabbitMQ(rabbitMQURL string) error {
	connMutex.Lock()
	defer connMutex.Unlock()

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	rabbitConn = conn
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchange (shared with other services)
	if err := ch.ExchangeDeclare(
		"incidentes", // name
		"topic",      // kind
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue for consuming task creation events
	q, err := ch.QueueDeclare(
		"schedule.q", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange for tarea_creada events
	if err := ch.QueueBind(
		q.Name,               // queue name
		"tarea.tarea_creada", // routing key
		"incidentes",         // exchange name
		false,                // no-wait
		nil,                  // arguments
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Println("✅ RabbitMQ initialized - Exchange 'incidentes' and queue 'schedule.q' declared")
	return nil
}

// PublishEvent publishes a schedule event to RabbitMQ
func PublishEvent(eventType string, payload interface{}) error {
	connMutex.Lock()
	defer connMutex.Unlock()

	if rabbitConn == nil {
		log.Println("⚠️  RabbitMQ not connected, skipping event publication")
		return fmt.Errorf("rabbitmq not connected")
	}

	ch, err := rabbitConn.Channel()
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

	// Construct routing key: tarea.{eventType}
	routingKey := fmt.Sprintf("tarea.%s", eventType)

	err = ch.Publish(
		"incidentes", // exchange
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

// TaskCreatedEvent structure from task-service
type TaskCreatedEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	TaskType  string    `json:"task_type"`
	Priority  string    `json:"priority"`
	Location  string    `json:"location"`
	Latitude  *float64  `json:"latitude"`
	Longitude *float64  `json:"longitude"`
	ReportDay time.Time `json:"report_day"`
	CreatedAt time.Time `json:"created_at"`
	Timestamp time.Time `json:"timestamp"`
}

// ConsumeTaskCreatedEvents starts consuming task creation events
func ConsumeTaskCreatedEvents(handler func(*TaskCreatedEvent) error) error {
	connMutex.Lock()
	defer connMutex.Unlock()

	if rabbitConn == nil {
		return fmt.Errorf("rabbitmq not connected")
	}

	ch, err := rabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	// Don't defer close - keep channel open for consumption

	msgs, err := ch.Consume(
		"schedule.q", // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Println("👂 Listening for task creation events...")

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

			// Convert to TaskCreatedEvent
			dataJSON, err := json.Marshal(dataRaw)
			if err != nil {
				log.Printf("❌ Failed to marshal data field: %v", err)
				delivery.Nack(false, true)
				continue
			}

			var task TaskCreatedEvent
			if err := json.Unmarshal(dataJSON, &task); err != nil {
				log.Printf("❌ Failed to unmarshal task event: %v", err)
				delivery.Nack(false, true)
				continue
			}

			// Call handler
			if err := handler(&task); err != nil {
				log.Printf("❌ Handler error for task %s: %v", task.ID, err)
				delivery.Nack(false, true) // Requeue on error
				continue
			}

			// Acknowledge delivery
			delivery.Ack(false)
			log.Printf("✅ Processed task event: %s", task.ID)
		}
	}()

	return nil
}

// CloseRabbitMQ closes RabbitMQ connection
func CloseRabbitMQ() {
	connMutex.Lock()
	defer connMutex.Unlock()

	if rabbitConn != nil && !rabbitConn.IsClosed() {
		rabbitConn.Close()
		log.Println("✅ RabbitMQ connection closed")
	}
}
