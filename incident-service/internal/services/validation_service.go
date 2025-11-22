package services

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/models"
)

// RequestValidation envia el incidente a la cola de validacion en RabbitMQ
func RequestValidation(incident *models.Incident) error {

	// Extraemos lat/long del string POINT(x y) para facilitar la vida al validador
	var lat, lon float64
	fmt.Sscanf(incident.Location, "POINT(%f %f)", &lon, &lat)

	payload := map[string]interface{}{
		"incident_id": incident.ID,
		"reporter_id": incident.ReporterID,
		"type":        incident.Type,
		"description": incident.Description,
		"location": map[string]float64{
			"latitude":  lat,
			"longitude": lon,
		},
		"address":      incident.Address,
		"photos_count": incident.PhotosCount,
		"incident_day": incident.IncidentDay,
		"status":       incident.Status,
	}

	// CLAVE: Usamos la Routing Key definida en tus Bindings del JSON inicial
	// Source: city.cleaning.incidents -> Key: incidents.submitted.v1 -> Queue: q.validation.incidents-submitted
	routingKey := "incidents.submitted.v1"

	if err := messaging.PublishEvent(routingKey, payload); err != nil {
		// NOTA: Aquí podrías guardar en la tabla 'outbox_events' si falla RabbitMQ
		// para reintentar luego (Patrón Transactional Outbox), pero por ahora solo logueamos.
		log.Printf("Error: Failed to publish to RabbitMQ: %v", err)
		return err
	}

	log.Printf("Validation requested for incident %s (Key: %s)", incident.ID, routingKey)
	return nil
}

func HandleValidationResponse(message []byte) error {
	var response struct {
		IncidentID string  `json:"incident_id"`
		IsValid    bool    `json:"is_valid"`
		Reason     *string `json:"reason"`
	}

	if err := json.Unmarshal(message, &response); err != nil {
		log.Printf("Error parsign validation response")
		return err
	}

	db := database.DB

	//Obtener incidente
	var incident models.Incident
	if err := db.Where("id = ?", response.IncidentID).First(&incident).Error; err != nil {
		log.Printf("Error: Incident not found: %s", response.IncidentID)
		return err
	}

	//Actualizar estado del incidente segun la validacion
	if response.IsValid {
		incident.Status = models.IncidentStatusValido
		log.Printf("Incidente validado: %s")
	} else {
		incident.Status = models.IncidentStatusRechazado
		log.Printf("Incident rejected: %s - Reason: %s", incident.ID, *response.Reason)
	}

	// Guardar cambios
	if err := db.Save(&incident).Error; err != nil {
		log.Printf("Error updating incident status: %v", err)
		return err
	}

	// Publicar evento del resultado
	eventType := "incident.validation.completed"
	payload := map[string]interface{}{
		"incident_id": incident.ID,
		"is_valid":    response.IsValid,
		"reason":      response.Reason,
		"status":      incident.Status,
	}

	if err := messaging.PublishEvent(eventType, payload); err != nil {
		log.Printf("Warning: Failed to publish validation result: %v", err)
	}

	return nil

}
