package consumers

import (
	"encoding/json"
	"log"

	"github.com/fleet-service/internal/services"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// WorkorderConsumer maneja actualizaciones de órdenes de trabajo
type WorkorderConsumer struct {
	channel           *amqp.Channel
	queueName         string
	allocationService *services.AllocationService
}

// WorkorderCompletedEvent representa una orden de trabajo completada
type WorkorderCompletedEvent struct {
	WorkorderID string `json:"workorder_id"`
	DriverID    string `json:"driver_id"`
	RouteID     string `json:"route_id,omitempty"`
	Status      string `json:"status"` // "COMPLETED"
	CompletedAt string `json:"completed_at"`
}

// NewWorkorderConsumer crea un nuevo consumidor de workorders
func NewWorkorderConsumer(
	conn *amqp.Connection,
	queueName, exchange string,
	allocationService *services.AllocationService,
) (*WorkorderConsumer, error) {
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
		queueName,                // queue name
		"workorder.completed.v1", // routing key
		exchange,                 // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("✓ WorkorderConsumer configurado - Queue: %s", queueName)

	return &WorkorderConsumer{
		channel:           channel,
		queueName:         queueName,
		allocationService: allocationService,
	}, nil
}

// Start inicia el consumo de mensajes
func (c *WorkorderConsumer) Start() error {
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

	log.Printf("🟡 WorkorderConsumer escuchando en cola: %s", c.queueName)

	go func() {
		for msg := range msgs {
			c.handleMessage(msg)
		}
	}()

	return nil
}

// handleMessage procesa eventos de workorder completada
func (c *WorkorderConsumer) handleMessage(msg amqp.Delivery) {
	log.Printf("📥 [Workorder] Mensaje recibido: %s", msg.RoutingKey)

	var event WorkorderCompletedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Error al parsear evento: %v", err)
		msg.Nack(false, false) // No requeue
		return
	}

	// Validar que el status sea COMPLETED
	if event.Status != "COMPLETED" {
		log.Printf("⚠ Status no es COMPLETED: %s (ignorado)", event.Status)
		msg.Ack(false)
		return
	}

	// Parsear UUID del driver
	driverID, err := uuid.Parse(event.DriverID)
	if err != nil {
		log.Printf("❌ UUID inválido: %v", err)
		msg.Nack(false, false)
		return
	}

	// Liberar al conductor (cambiar status a DISPONIBLE)
	if err := c.allocationService.ReleaseDriver(driverID); err != nil {
		log.Printf("❌ Error al liberar conductor: %v", err)
		msg.Nack(false, true) // Requeue
		return
	}

	log.Printf("✅ Conductor liberado por completar workorder: %s (Workorder: %s)",
		event.DriverID, event.WorkorderID)
	msg.Ack(false)
}

// Close cierra el canal
func (c *WorkorderConsumer) Close() error {
	return c.channel.Close()
}
