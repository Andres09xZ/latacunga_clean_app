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

var (
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	mu         sync.Mutex
)

// InitRabbitMQ inicializa la conexión a RabbitMQ
func InitRabbitMQ() error {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RABBITMQ_URL not set, RabbitMQ messaging disabled")
		return nil
	}

	var err error
	rabbitConn, err = amqp.Dial(rabbitURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
		return err
	}

	rabbitCh, err = rabbitConn.Channel()
	if err != nil {
		log.Printf("Warning: Failed to open RabbitMQ channel: %v", err)
		return err
	}

	// Declarar exchange "incidentes"
	err = rabbitCh.ExchangeDeclare(
		"incidentes",
		amqp.ExchangeTopic,
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Printf("Warning: Failed to declare exchange: %v", err)
		return err
	}

	log.Println("Connected to RabbitMQ successfully")
	return nil
}

// PublishEvent publica un evento a RabbitMQ
func PublishEvent(eventType string, payload map[string]interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	if rabbitConn == nil || rabbitCh == nil {
		log.Println("RabbitMQ not connected, skipping event publication")
		return nil
	}

	// Agregar timestamp al payload
	payload["timestamp"] = time.Now().Format(time.RFC3339)
	payload["event_type"] = eventType

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return err
	}

	// Routing key: incidente.{eventType}
	routingKey := fmt.Sprintf("incidente.%s", eventType)

	err = rabbitCh.Publish(
		"incidentes",
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		log.Printf("Failed to publish event to RabbitMQ: %v", err)
		return err
	}

	log.Printf("Event published to RabbitMQ - Type: %s, RoutingKey: %s", eventType, routingKey)
	return nil
}

// CloseRabbitMQ cierra la conexión a RabbitMQ
func CloseRabbitMQ() error {
	mu.Lock()
	defer mu.Unlock()

	if rabbitCh != nil {
		rabbitCh.Close()
	}
	if rabbitConn != nil {
		return rabbitConn.Close()
	}
	return nil
}
