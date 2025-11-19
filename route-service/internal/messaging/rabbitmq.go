package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQClient wraps RabbitMQ connection and channels
type RabbitMQClient struct {
	conn            *amqp.Connection
	publishChannel  *amqp.Channel
	consumerChannel *amqp.Channel
	exchangeName    string
	eventHandlers   map[string]EventHandler
}

// EventHandler is a function that handles an event
type EventHandler func(ctx context.Context, eventData []byte) error

// NewRabbitMQClient creates a new RabbitMQ client
func NewRabbitMQClient(rabbitURL string, exchangeName string) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	publishChannel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create publish channel: %w", err)
	}

	consumerChannel, err := conn.Channel()
	if err != nil {
		publishChannel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to create consumer channel: %w", err)
	}

	// Declare exchange
	err = publishChannel.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		publishChannel.Close()
		consumerChannel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	client := &RabbitMQClient{
		conn:            conn,
		publishChannel:  publishChannel,
		consumerChannel: consumerChannel,
		exchangeName:    exchangeName,
		eventHandlers:   make(map[string]EventHandler),
	}

	log.Printf("✅ Connected to RabbitMQ, exchange: %s", exchangeName)
	return client, nil
}

// RegisterEventHandler registers a handler for a specific event type
func (rc *RabbitMQClient) RegisterEventHandler(eventType string, handler EventHandler) {
	rc.eventHandlers[eventType] = handler
	log.Printf("📝 Registered handler for event type: %s", eventType)
}

// PublishEvent publishes an event to the exchange
func (rc *RabbitMQClient) PublishEvent(ctx context.Context, routingKey string, eventData interface{}) error {
	body, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = rc.publishChannel.PublishWithContext(
		ctx,
		rc.exchangeName, // exchange
		routingKey,      // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("📤 Published event: %s (routing key: %s)", eventData, routingKey)
	return nil
}

// ConsumeEvent consumes events from a specific routing key
func (rc *RabbitMQClient) ConsumeEvent(ctx context.Context, queueName string, routingKey string, eventType string) error {
	// Declare queue
	queue, err := rc.consumerChannel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // auto-delete
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = rc.consumerChannel.QueueBind(
		queue.Name,      // queue name
		routingKey,      // routing key
		rc.exchangeName, // exchange name
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Set QoS to process one message at a time
	err = rc.consumerChannel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	msgs, err := rc.consumerChannel.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack (manual ack)
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Printf("🔔 Listening for events: %s (queue: %s, routing key: %s)", eventType, queueName, routingKey)

	// Process messages
	go func() {
		for d := range msgs {
			log.Printf("📨 Received message from %s: %s", queueName, string(d.Body))

			handler, exists := rc.eventHandlers[eventType]
			if !exists {
				log.Printf("⚠️  No handler for event type: %s", eventType)
				d.Nack(false, true) // Requeue
				continue
			}

			err := handler(ctx, d.Body)
			if err != nil {
				log.Printf("❌ Error processing event: %v", err)
				d.Nack(false, true) // Requeue on error
			} else {
				d.Ack(false) // Acknowledge success
			}
		}
	}()

	return nil
}

// Close closes the RabbitMQ connection
func (rc *RabbitMQClient) Close() error {
	if rc.publishChannel != nil {
		rc.publishChannel.Close()
	}
	if rc.consumerChannel != nil {
		rc.consumerChannel.Close()
	}
	if rc.conn != nil {
		return rc.conn.Close()
	}
	return nil
}

// EventPayload represents a generic event payload
type EventPayload struct {
	EventID     string      `json:"event_id"`
	EventType   string      `json:"event_type"`
	AggregateID string      `json:"aggregate_id"`
	Timestamp   string      `json:"timestamp"`
	Data        interface{} `json:"data"`
}

// TareaEvent represents a task event
type TareaEvent struct {
	TareaID     string  `json:"tarea_id"`
	OperadorID  string  `json:"operador_id"`
	Latitud     float64 `json:"latitud"`
	Longitud    float64 `json:"longitud"`
	Duracion    int64   `json:"duracion"` // seconds
	Prioridad   string  `json:"prioridad"`
	Descripcion string  `json:"descripcion"`
	WindowStart *string `json:"window_start"`
	WindowEnd   *string `json:"window_end"`
	Timestamp   string  `json:"timestamp"`
}

// ScheduleEvent represents a schedule event
type ScheduleEvent struct {
	ScheduleID   string `json:"schedule_id"`
	OperadorID   string `json:"operador_id"`
	FechaLaboral string `json:"fecha_laboral"`
	HoraInicio   string `json:"hora_inicio"`
	HoraFin      string `json:"hora_fin"`
	Timestamp    string `json:"timestamp"`
}

// RutaCalculadaEvent represents a calculated route event
type RutaCalculadaEvent struct {
	RutaID         string          `json:"ruta_id"`
	OperadorID     string          `json:"operador_id"`
	FechaLaboral   string          `json:"fecha_laboral"`
	DistanciaTotal int64           `json:"distancia_total"` // meters
	TiempoTotal    int64           `json:"tiempo_total"`    // seconds
	EsHeuristica   bool            `json:"es_heuristica"`
	Pasos          []PasoCalculado `json:"pasos"`
	Timestamp      string          `json:"timestamp"`
}

// PasoCalculado represents a step in the calculated route
type PasoCalculado struct {
	TareaID        string `json:"tarea_id"`
	Secuencia      int    `json:"secuencia"`
	HoraLlegada    string `json:"hora_llegada"`
	HoraSalida     string `json:"hora_salida"`
	DistanciaTramo int64  `json:"distancia_tramo"` // meters
	TiempoTramo    int64  `json:"tiempo_tramo"`    // seconds
}

// TareaNoEnrutadaEvent represents tasks that couldn't be routed
type TareaNoEnrutadaEvent struct {
	RutaID       string   `json:"ruta_id"`
	OperadorID   string   `json:"operador_id"`
	TareasID     []string `json:"tareas_id"`
	FechaLaboral string   `json:"fecha_laboral"`
	Razon        string   `json:"razon"`
	Timestamp    string   `json:"timestamp"`
}
