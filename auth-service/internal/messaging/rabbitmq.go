package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "city.cleaning.identity"
	ExchangeType = "topic"
)

var (
	RabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	mu         sync.Mutex
)

// InitRabbitMQ initializes the RabbitMQ connection and ensures the exchange exists
func InitRabbitMQ() error {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RABBITMQ_URL not set, RabbitMQ messaging disabled")
		return nil
	}

	var err error
	RabbitConn, err = amqp.Dial(rabbitURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	rabbitCh, err = RabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	// Declare the exchange
	err = rabbitCh.ExchangeDeclare(
		ExchangeName, // city.cleaning.identity
		ExchangeType, // topic
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	log.Printf("Connected to RabbitMQ. Exchange '%s' is ready.", ExchangeName)
	return nil
}

// PublishEvent publishes an event to RabbitMQ
func PublishEvent(routingKey string, payload map[string]interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	if RabbitConn == nil || rabbitCh == nil || RabbitConn.IsClosed() {
		log.Println("RabbitMQ connection lost or not initialized")
		return fmt.Errorf("connection closed")
	}

	// Add event timestamp
	payload["event_timestamp"] = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	err = rabbitCh.Publish(
		ExchangeName, // city.cleaning.identity
		routingKey,   // e.g., identity.operator.created.v1
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    fmt.Sprintf("%v", time.Now().UnixNano()),
			Timestamp:    time.Now(),
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish: %v", err)
	}

	log.Printf("Published event to RabbitMQ: routing_key=%s", routingKey)
	return nil
}

// CloseRabbitMQ closes the RabbitMQ connection
func CloseRabbitMQ() {
	if rabbitCh != nil {
		rabbitCh.Close()
	}
	if RabbitConn != nil {
		RabbitConn.Close()
	}
}
