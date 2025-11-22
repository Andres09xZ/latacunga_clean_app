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
	ExchangeName = "city.cleaning.incidents" 
	ExchangeType = "topic"
)

var (
	RabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	mu 	   		sync.Mutex
)

// InitRabbitMQ incializa la conexion y asegura que el Exchange exista 
func InitRabbitMQ() error {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RABBITMQ_URL not set, RabbitMQ messaging disabled")
		return nil
	}

	var err error
	RabbitConn, err = amqp.Dial(rabbitURL)
	if err != nil {
		return fmt.Errorf("Failed to connect to RabbitMQ: %v", err)
	}

	rabbitCh, err = RabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("Failed to open a channel: %v", err)
	}

	// Declarar el Exchange
	err = rabbitCh.ExchangeDeclare(
		ExchangeName, // cityt.cleaning.incidents
		ExchangeType, // topic
		true, 		  // durable
		false, 		  // auto-deleted
		false, 		  // internal
		false, 		  // no-wait
		nil,		  // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to delcare exchange: %v", err)
	}

	log.Printf("Connected to RabbitMQ. Exchange '%s' is ready.", ExchangeName)
	return nil

}

func PublishEvent(routingKey string, payload map[string]interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	if RabbitConn == nil || rabbitCh == nil || RabbitConn.IsClosed() {
		// Intento simple de reconexion o fallo rapido 
		log.Println("RabbitMQ connection lost or not initialized")
		return fmt.Errorf("connection closed")
	}

	// Metodos utiles 

	payload["event_timestamp"] = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	err = rabbitCh.Publish(
		ExchangeName, // city.cleaning.incidents
		routingKey,   // Ej: incidents.submitted.v1
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // Importante para no perder mensajes si Rabbit se reinicia
			MessageId:    fmt.Sprintf("%v", time.Now().UnixNano()),
			Timestamp:    time.Now(),
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish: %v", err)
	}

	log.Printf("Event published -> Exchange: %s | Key: %s", ExchangeName, routingKey)
	return nil
}

func CloseRabbitMQ() {
	mu.Lock()
	defer mu.Unlock()
	if rabbitCh != nil {
		rabbitCh.Close()
	}
	if RabbitConn != nil {
		RabbitConn.Close()
	}
}

