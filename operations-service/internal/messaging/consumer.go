package messaging

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/models"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// WorkOrderCreatedEvent representa el evento de creación de orden de trabajo
type WorkOrderCreatedEvent struct {
	RequestID  string `json:"request_id"`
	ZoneID     int    `json:"zone_id"`
	DriverID   string `json:"driver_id"`
	TruckPlate string `json:"truck_plate"`
	Geometry   string `json:"geometry"` // RoutePolyline
	Stops      []struct {
		ID        string  `json:"id"` // incident_ref_id
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lon"`
		Address   string  `json:"address,omitempty"`
	} `json:"stops"`
}

// StartConsumer inicia el consumidor de mensajes de RabbitMQ
func StartConsumer() error {
	if RabbitConn == nil || rabbitCh == nil {
		log.Println("RabbitMQ not initialized, consumer disabled")
		return nil
	}

	msgs, err := rabbitCh.Consume(
		QueueName,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	log.Printf("Started consuming from queue: %s", QueueName)

	go func() {
		for msg := range msgs {
			if err := processWorkOrderCreated(msg); err != nil {
				log.Printf("Error processing message: %v", err)
				msg.Nack(false, true) // Requeue message
			} else {
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// processWorkOrderCreated procesa el evento de creación de orden de trabajo
func processWorkOrderCreated(msg amqp.Delivery) error {
	var event WorkOrderCreatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("Failed to unmarshal event: %v", err)
		return err
	}

	log.Printf("Received workorder.created event: request_id=%s, zone=%d, driver=%s",
		event.RequestID, event.ZoneID, event.DriverID)

	// Parsear driver_id
	driverUUID, err := uuid.Parse(event.DriverID)
	if err != nil {
		log.Printf("Invalid driver_id: %v", err)
		return err
	}

	// Crear la Work Order
	workOrder := models.WorkOrder{
		RequestID:     event.RequestID,
		ZoneID:        event.ZoneID,
		DriverID:      driverUUID,
		TruckPlate:    event.TruckPlate,
		Status:        models.WorkOrderStatusAssigned,
		RoutePolyline: event.Geometry,
		TotalStops:    len(event.Stops),
		AssignedAt:    time.Now(),
	}

	// Iniciar transacción
	tx := database.DB.Begin()

	if err := tx.Create(&workOrder).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to create work order: %v", err)
		return err
	}

	// Crear las paradas
	for i, stop := range event.Stops {
		incidentID, err := uuid.Parse(stop.ID)
		if err != nil {
			tx.Rollback()
			log.Printf("Invalid incident_ref_id: %v", err)
			return err
		}

		workOrderStop := models.WorkOrderStop{
			WorkOrderID:   workOrder.ID,
			IncidentRefID: incidentID,
			Latitude:      stop.Latitude,
			Longitude:     stop.Longitude,
			Address:       stop.Address,
			SequenceOrder: i + 1,
			Status:        models.StopStatusPending,
		}

		if err := tx.Create(&workOrderStop).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to create work order stop: %v", err)
			return err
		}
	}

	// Commit transacción
	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return err
	}

	log.Printf("Work order created successfully: id=%s, stops=%d", workOrder.ID, len(event.Stops))
	return nil
}
