package services

import (
	"encoding/json"
    "log"

    "github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/database"
    "github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/messaging"
    "github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/models"

)


func RequestValidation(incident *models.Incident) error {
	payload := map[string]interface{}{
		"incident_id": incident.ID,
		"type": 	  incident.Type,
		"description": incident.Description,
		"location":    incident.Location,
		"address":     incident.Address,
		"reporter_kind": incident.ReporterKind,
		"photos_count": incident.PhotosCount,
		"incident_day": incident.IncidentDay,
	}

	//Publicar evento para que validation-service lo consuma
	if err := messaging.PublishEvent("incident.validation.requested", payload); err != nil {
		log.Printf("Error: Failed to request validation: %v", err)
		return err
	}

	log.Printf("Validation requested for incident: %s", incident.ID)
	return nil
}

func HandleValidationResponse(message []byte) error {
	var response struct {
		IncidentID string `json:"incident_id"`
		IsValid	bool   `json:"is_valid"`
		Reason   *string `json:"reason"`
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
	if response.IsValid{
		incident.Status = models.IncidentStatusValido
		log.Printf("Incidente validado: %s")
	}else{
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