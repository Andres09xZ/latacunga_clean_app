package messaging

import (
	"encoding/json"
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/models"
)

// ValidationResult representa el resultado de validación recibido
type ValidationResult struct {
	IncidentID  string `json:"incident_id"`
	Status      string `json:"status"`
	Validator   string `json:"validator"`
	ValidatedAt string `json:"validated_at"`
	Notes       string `json:"notes,omitempty"`
}

// StartValidationConsumer inicia el consumidor que escucha resultados de validación
func StartValidationConsumer() {
	if RabbitConn == nil {
		log.Println("⚠️ RabbitMQ not connected, validation consumer not started")
		return
	}

	ch, err := RabbitConn.Channel()
	if err != nil {
		log.Printf("❌ Failed to open channel for validation consumer: %v", err)
		return
	}
	// No cerramos el canal aquí porque lo necesitamos para consumir continuamente

	// Declarar exchange
	exchangeName := "city.cleaning.incidents"
	if err := ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Printf("❌ Failed to declare exchange: %v", err)
		return
	}

	// Crear cola duradera para este servicio
	q, err := ch.QueueDeclare(
		"incident-service.validation-results.queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Printf("❌ Failed to declare validation results queue: %v", err)
		return
	}

	// Bind a los eventos de validación
	routingKey := "incidents.validated.v1"
	if err := ch.QueueBind(q.Name, routingKey, exchangeName, false, nil); err != nil {
		log.Printf("❌ Failed to bind validation results queue: %v", err)
		return
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("❌ Failed to register validation consumer: %v", err)
		return
	}

	log.Println("🎧 Validation results consumer started, listening for incidents.validated.v1...")

	// Procesar mensajes en goroutine
	go func() {
		for d := range msgs {
			log.Printf("\n📨 Validation result received: routing_key=%s", d.RoutingKey)

			var result ValidationResult
			if err := json.Unmarshal(d.Body, &result); err != nil {
				log.Printf("❌ Failed to unmarshal validation result: %v", err)
				d.Nack(false, false) // No reencolar si es malformado
				continue
			}

			log.Printf("📋 Validation: incident_id=%s status=%s validator=%s notes=%s",
				result.IncidentID, result.Status, result.Validator, result.Notes)

			// Actualizar el incidente en la base de datos
			if err := updateIncidentStatus(result.IncidentID, result.Status); err != nil {
				log.Printf("❌ Failed to update incident status: %v", err)
				d.Nack(false, true) // Reencolar para reintentar
				continue
			}

			log.Printf("✅ Incident %s updated to status: %s", result.IncidentID, result.Status)
			d.Ack(false)
		}
	}()
}

// updateIncidentStatus actualiza el estado del incidente en la base de datos
func updateIncidentStatus(incidentID string, newStatus string) error {
	db := database.DB

	var incident models.Incident
	if err := db.Where("id = ?", incidentID).First(&incident).Error; err != nil {
		return err
	}

	// Mapear estados de validación a estados de incidente
	var status models.IncidentStatus
	switch newStatus {
	case "incidente_valido":
		status = models.IncidentStatusValido
	case "incidente_rechazado":
		status = models.IncidentStatusRechazado
	default:
		log.Printf("⚠️ Unknown validation status: %s, keeping current status", newStatus)
		return nil
	}

	// Actualizar estado
	incident.Status = status
	if err := db.Save(&incident).Error; err != nil {
		return err
	}

	// Publicar evento de actualización de estado
	payload := map[string]interface{}{
		"incident_id": incidentID,
		"old_status":  incident.Status, // Esto ya tiene el nuevo valor, pero está bien
		"new_status":  status,
		"updated_by":  "validation-service",
	}

	if err := PublishEvent("incidents.status-updated.v1", payload); err != nil {
		log.Printf("⚠️ Failed to publish status update event: %v", err)
	}

	return nil
}
