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
	ExchangeName = "city.cleaning.operations"
	ExchangeType = "topic"
	QueueName    = "q.operations.workorders"
	RoutingKey   = "workorders.created.v1"
)

var (
	RabbitConn *amqp.Connection
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
	RabbitConn, err = amqp.Dial(rabbitURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	rabbitCh, err = RabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	// Declarar el exchange
	err = rabbitCh.ExchangeDeclare(
		ExchangeName,
		ExchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Declarar la cola
	_, err = rabbitCh.QueueDeclare(
		QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Bind la cola al exchange
	err = rabbitCh.QueueBind(
		QueueName,
		RoutingKey,
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %v", err)
	}

	log.Printf("Connected to RabbitMQ. Exchange '%s' and Queue '%s' are ready.", ExchangeName, QueueName)
	return nil
}

// PublishEvent publica un evento a RabbitMQ
func PublishEvent(routingKey string, payload map[string]interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	if RabbitConn == nil || rabbitCh == nil || RabbitConn.IsClosed() {
		log.Println("RabbitMQ connection lost or not initialized")
		return fmt.Errorf("connection closed")
	}

	// Agregar timestamp al evento
	payload["event_timestamp"] = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	err = rabbitCh.Publish(
		ExchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
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

// CloseRabbitMQ cierra la conexión a RabbitMQ
func CloseRabbitMQ() {
	if rabbitCh != nil {
		rabbitCh.Close()
	}
	if RabbitConn != nil {
		RabbitConn.Close()
	}
}
