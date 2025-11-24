package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/fleet-service/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

// EventPublisher maneja la publicación de eventos a RabbitMQ
type EventPublisher struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	exchanges config.ExchangeConfig
}

// NewEventPublisher crea una nueva instancia del publicador
func NewEventPublisher(rabbitURL string, exchanges config.ExchangeConfig) (*EventPublisher, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("error al conectar con RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error al crear canal de RabbitMQ: %w", err)
	}

	// Declarar exchanges
	exchanges_to_declare := []string{
		exchanges.CityCleaningResources,
		exchanges.IdentityManagement,
		exchanges.PlanningScheduler,
		exchanges.OperationsWorkorders,
	}

	for _, exchange := range exchanges_to_declare {
		err = channel.ExchangeDeclare(
			exchange, // name
			"topic",  // type
			true,     // durable
			false,    // auto-deleted
			false,    // internal
			false,    // no-wait
			nil,      // arguments
		)
		if err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("error al declarar exchange %s: %w", exchange, err)
		}
	}

	log.Println("✓ EventPublisher inicializado correctamente")

	return &EventPublisher{
		conn:      conn,
		channel:   channel,
		exchanges: exchanges,
	}, nil
}

// ResourceAvailableEvent representa el evento cuando un recurso está disponible
type ResourceAvailableEvent struct {
	DriverID   string    `json:"driver_id"`
	DriverName string    `json:"driver_name"`
	TruckID    uint      `json:"truck_id"`
	TruckPlate string    `json:"truck_plate"`
	TruckType  string    `json:"truck_type"`
	ShiftID    string    `json:"shift_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// DriverAllocatedEvent representa el evento cuando un conductor es asignado
type DriverAllocatedEvent struct {
	DriverID   string    `json:"driver_id"`
	DriverName string    `json:"driver_name"`
	TruckID    uint      `json:"truck_id"`
	TruckPlate string    `json:"truck_plate"`
	TruckType  string    `json:"truck_type"`
	ShiftID    string    `json:"shift_id"`
	ZoneID     int       `json:"zone_id"`
	RequestID  string    `json:"request_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// PublishResourceAvailable publica el evento resource.available.v1
func (p *EventPublisher) PublishResourceAvailable(event ResourceAvailableEvent) error {
	return p.publish(
		p.exchanges.CityCleaningResources,
		"resource.available.v1",
		event,
	)
}

// PublishDriverAllocated publica el evento resources.driver.allocated.v1
func (p *EventPublisher) PublishDriverAllocated(event DriverAllocatedEvent) error {
	return p.publish(
		p.exchanges.CityCleaningResources,
		"resources.driver.allocated.v1",
		event,
	)
}

// publish es el método interno para publicar cualquier evento
func (p *EventPublisher) publish(exchange, routingKey string, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error al serializar evento: %w", err)
	}

	err = p.channel.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Mensaje persistente
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("error al publicar evento: %w", err)
	}

	log.Printf("📤 Evento publicado: %s -> %s", exchange, routingKey)
	return nil
}

// Close cierra las conexiones del publicador
func (p *EventPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}
