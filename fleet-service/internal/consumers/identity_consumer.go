package consumers

import (
	"encoding/json"
	"log"

	"github.com/fleet-service/internal/database"
	"github.com/fleet-service/internal/models"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// IdentityConsumer maneja la sincronización de operadores desde Identity Service
type IdentityConsumer struct {
	channel   *amqp.Channel
	queueName string
}

// OperatorCreatedEvent representa el evento de creación de operador
// Soporta múltiples nombres para el ID del operador
type OperatorCreatedEvent struct {
	DriverID          string `json:"driver_id"`   // Intenta con driver_id
	OperatorID        string `json:"operator_id"` // Intenta con operator_id
	ID                string `json:"id"`          // Intenta con id
	UserID            string `json:"user_id"`     // Intenta con user_id (auth service)
	FullName          string `json:"full_name"`
	Email             string `json:"email"`
	PreferredZoneID   int    `json:"preferred_zone_id"`
	CanDriveLateral   bool   `json:"can_drive_lateral"`
	CanDrivePosterior bool   `json:"can_drive_posterior"`
}

// GetID retorna el primer ID válido encontrado
func (e *OperatorCreatedEvent) GetID() string {
	if e.DriverID != "" {
		return e.DriverID
	}
	if e.OperatorID != "" {
		return e.OperatorID
	}
	if e.UserID != "" {
		return e.UserID
	}
	return e.ID
}

// NewIdentityConsumer crea un nuevo consumidor de identidad
func NewIdentityConsumer(conn *amqp.Connection, queueName, exchange string) (*IdentityConsumer, error) {
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
		queueName,                      // queue name
		"identity.operator.created.v1", // routing key
		exchange,                       // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("✓ IdentityConsumer configurado - Queue: %s", queueName)

	return &IdentityConsumer{
		channel:   channel,
		queueName: queueName,
	}, nil
}

// Start inicia el consumo de mensajes
func (c *IdentityConsumer) Start() error {
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

	log.Printf("🔵 IdentityConsumer escuchando en cola: %s", c.queueName)

	go func() {
		for msg := range msgs {
			c.handleMessage(msg)
		}
	}()

	return nil
}

// handleMessage procesa cada mensaje recibido
func (c *IdentityConsumer) handleMessage(msg amqp.Delivery) {
	log.Printf("📥 [Identity] Mensaje recibido: %s", msg.RoutingKey)
	log.Printf("📦 [Identity] Body completo: %s", string(msg.Body))

	var event OperatorCreatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Error al parsear evento: %v", err)
		msg.Nack(false, false) // No requeue
		return
	}

	log.Printf("🔍 [Identity] Event parseado - DriverID: '%s', FullName: '%s'", event.DriverID, event.FullName)

	// Obtener el ID del operador (puede venir como driver_id, operator_id o id)
	operatorIDStr := event.GetID()
	if operatorIDStr == "" {
		log.Printf("❌ No se encontró ningún ID válido en el evento (driver_id, operator_id o id)")
		msg.Nack(false, false)
		return
	}

	// Parsear UUID
	driverID, err := uuid.Parse(operatorIDStr)
	if err != nil {
		log.Printf("❌ UUID inválido ('%s'): %v", operatorIDStr, err)
		msg.Nack(false, false)
		return
	}

	db := database.GetDB()

	// Iniciar transacción
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Crear el Driver
	driver := models.Driver{
		ID:       driverID,
		FullName: event.FullName,
		Status:   models.DriverStatusOffline,
	}

	if err := tx.Create(&driver).Error; err != nil {
		tx.Rollback()
		log.Printf("❌ Error al crear driver: %v", err)
		msg.Nack(false, true) // Requeue
		return
	}

	// 2. Crear el OperatorProfile
	profile := models.OperatorProfile{
		DriverID:          driverID,
		UserID:            &driverID, // Usar el mismo UUID como user_id
		PreferredZoneID:   event.PreferredZoneID,
		CanDriveLateral:   event.CanDriveLateral,
		CanDrivePosterior: event.CanDrivePosterior,
	}

	if err := tx.Create(&profile).Error; err != nil {
		tx.Rollback()
		log.Printf("❌ Error al crear operator_profile: %v", err)
		msg.Nack(false, true) // Requeue
		return
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		log.Printf("❌ Error al hacer commit: %v", err)
		msg.Nack(false, true) // Requeue
		return
	}

	log.Printf("✅ Operador sincronizado: %s (%s) - Zona: %d", event.FullName, driverID, event.PreferredZoneID)
	msg.Ack(false)
}

// Close cierra el canal
func (c *IdentityConsumer) Close() error {
	return c.channel.Close()
}
