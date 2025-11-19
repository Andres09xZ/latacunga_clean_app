package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/clients"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ValidationHandler handles validation operations
type ValidationHandler struct {
	incidentClient *clients.IncidentClient
}

// NewValidationHandler creates a new validation handler
func NewValidationHandler(incidentServiceURL string) *ValidationHandler {
	return &ValidationHandler{
		incidentClient: clients.NewIncidentClient(incidentServiceURL),
	}
}

// MarkValidRequest for marking incident as valid
type MarkValidRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// RejectRequest for rejecting incident
type RejectRequest struct {
	Reason         string `json:"reason" binding:"required"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// MarkIncidentAsValid marks an incident as valid (MVP endpoint)
// @Summary Mark incident as valid
// @Description Manually validate an incident and change its status to "valida"
// @Tags validations
// @Accept json
// @Produce json
// @Param incident_id path string true "Incident ID"
// @Param request body MarkValidRequest true "Validation request"
// @Security Bearer
// @Success 200 {object} models.ValidationResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/validations/{incident_id}/mark-valid [post]
func (vh *ValidationHandler) MarkIncidentAsValid(c *gin.Context) {
	incidentID := c.Param("incident_id")
	userID, _ := c.Get("user_id")
	authHeader := c.GetHeader("Authorization")

	// Extract token from "Bearer <token>"
	token := strings.TrimPrefix(authHeader, "Bearer ")

	var req MarkValidRequest
	if err := c.BindJSON(&req); err != nil {
		log.Printf("❌ Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Check idempotency
	existingKey := &models.IdempotencyKey{}
	if err := database.DB.Where("key = ?", req.IdempotencyKey).First(existingKey).Error; err == nil {
		// Key exists, fetch existing validation
		validation := &models.Validation{}
		if err := database.DB.Where("incident_id = ? AND status = ?", incidentID, models.StatusValido).First(validation).Error; err == nil {
			log.Printf("✅ Idempotent call - returning existing validation for %s", incidentID)
			c.JSON(http.StatusOK, convertValidationToResponse(validation, req.IdempotencyKey))
			return
		}
	}

	// Get incident from incident-service
	incident, err := vh.incidentClient.GetIncident(incidentID, token)
	if err != nil {
		log.Printf("❌ Failed to get incident: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
		return
	}

	// Verify incident is in "emitida" status
	if incident.Status != "emitido" && incident.Status != "emitida" {
		log.Printf("❌ Incident %s is not in emitida state: %s", incidentID, incident.Status)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("incident is in %s state, cannot validate", incident.Status)})
		return
	}

	// Create validation record in database
	now := time.Now()
	validation := &models.Validation{
		ID:            uuid.New(),
		IncidentID:    incidentID,
		Status:        models.StatusValido,
		ValidatorKind: models.ValidatorManual,
		RequestedAt:   now,
		DecidedAt:     &now,
	}

	if err := database.DB.Create(validation).Error; err != nil {
		log.Printf("❌ Failed to create validation record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create validation"})
		return
	}

	// Update incident status via incident-service
	if err := vh.incidentClient.UpdateIncidentStatus(incidentID, "rechazado", token); err != nil {
		log.Printf("❌ Failed to update incident status: %v", err)
		// Rollback validation record
		database.DB.Delete(validation)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update incident status"})
		return
	}

	// Publish validation event
	if err := publishValidationEvent(validation, incident); err != nil {
		log.Printf("⚠️  Warning: Failed to publish event: %v", err)
		// Still return 200 but log warning
	}

	// Store idempotency key
	idempKey := &models.IdempotencyKey{
		Key:        req.IdempotencyKey,
		IncidentID: incidentID,
		Action:     "mark_valid",
	}
	database.DB.Create(idempKey)

	log.Printf("✅ Incident %s marked as valid by %v", incidentID, userID)
	c.JSON(http.StatusOK, convertValidationToResponse(validation, req.IdempotencyKey))
}

// RejectIncident rejects an incident with a reason
// @Summary Reject incident
// @Description Manually reject an incident and change its status to "rechazada"
// @Tags validations
// @Accept json
// @Produce json
// @Param incident_id path string true "Incident ID"
// @Param request body RejectRequest true "Rejection request"
// @Security Bearer
// @Success 200 {object} models.ValidationResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/validations/{incident_id}/reject [post]
func (vh *ValidationHandler) RejectIncident(c *gin.Context) {
	incidentID := c.Param("incident_id")
	userID, _ := c.Get("user_id")
	authHeader := c.GetHeader("Authorization")

	// Extract token from "Bearer <token>"
	token := strings.TrimPrefix(authHeader, "Bearer ")

	var req RejectRequest
	if err := c.BindJSON(&req); err != nil {
		log.Printf("❌ Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}

	// Check idempotency
	existingKey := &models.IdempotencyKey{}
	if err := database.DB.Where("key = ?", req.IdempotencyKey).First(existingKey).Error; err == nil {
		// Key exists, fetch existing validation
		validation := &models.Validation{}
		if err := database.DB.Where("incident_id = ? AND status = ?", incidentID, models.StatusRechazado).First(validation).Error; err == nil {
			log.Printf("✅ Idempotent call - returning existing rejection for %s", incidentID)
			c.JSON(http.StatusOK, convertValidationToResponse(validation, req.IdempotencyKey))
			return
		}
	}

	// Get incident from incident-service
	incident, err := vh.incidentClient.GetIncident(incidentID, token)
	if err != nil {
		log.Printf("❌ Failed to get incident: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
		return
	}

	// Verify incident is in "emitida" status
	if incident.Status != "emitido" && incident.Status != "emitida" {
		log.Printf("❌ Incident %s is not in emitida state: %s", incidentID, incident.Status)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("incident is in %s state, cannot reject", incident.Status)})
		return
	}

	// Create rejection record
	now := time.Now()
	validation := &models.Validation{
		ID:            uuid.New(),
		IncidentID:    incidentID,
		Status:        models.StatusRechazado,
		Reason:        &req.Reason,
		ValidatorKind: models.ValidatorManual,
		RequestedAt:   now,
		DecidedAt:     &now,
	}

	if err := database.DB.Create(validation).Error; err != nil {
		log.Printf("❌ Failed to create rejection record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rejection"})
		return
	}

	// Update incident status to "rechazada"
	if err := vh.incidentClient.UpdateIncidentStatus(incidentID, "rechazada", token); err != nil {
		log.Printf("❌ Failed to update incident status: %v", err)
		// Rollback rejection record
		database.DB.Delete(validation)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update incident status"})
		return
	}

	// Publish rejection event
	if err := publishRejectionEvent(validation, incident); err != nil {
		log.Printf("⚠️  Warning: Failed to publish event: %v", err)
	}

	// Store idempotency key
	idempKey := &models.IdempotencyKey{
		Key:        req.IdempotencyKey,
		IncidentID: incidentID,
		Action:     "reject",
	}
	database.DB.Create(idempKey)

	log.Printf("✅ Incident %s rejected by %v - reason: %s", incidentID, userID, req.Reason)
	c.JSON(http.StatusOK, convertValidationToResponse(validation, req.IdempotencyKey))
}

// GetValidation retrieves validation history for an incident
// @Summary Get validation history
// @Description Get all validations for a specific incident
// @Tags validations
// @Accept json
// @Produce json
// @Param incident_id path string true "Incident ID"
// @Security Bearer
// @Success 200 {array} models.ValidationResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/validations/{incident_id} [get]
func (vh *ValidationHandler) GetValidation(c *gin.Context) {
	incidentID := c.Param("incident_id")

	validations := []models.Validation{}
	if err := database.DB.Where("incident_id = ?", incidentID).Order("created_at DESC").Find(&validations).Error; err != nil && err != gorm.ErrRecordNotFound {
		log.Printf("❌ Failed to query validations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve validations"})
		return
	}

	if len(validations) == 0 {
		c.JSON(http.StatusOK, []models.ValidationResponse{})
		return
	}

	responses := make([]models.ValidationResponse, len(validations))
	for i, v := range validations {
		responses[i] = convertValidationToResponse(&v, "")
	}

	c.JSON(http.StatusOK, responses)
}

// Helper functions

func convertValidationToResponse(v *models.Validation, idempotencyKey string) models.ValidationResponse {
	return models.ValidationResponse{
		ID:             v.ID,
		IncidentID:     v.IncidentID,
		Status:         v.Status.String(),
		Reason:         v.Reason,
		ValidatorKind:  v.ValidatorKind.String(),
		RequestedAt:    v.RequestedAt,
		DecidedAt:      v.DecidedAt,
		IdempotencyKey: idempotencyKey,
	}
}

func publishValidationEvent(validation *models.Validation, incident *clients.IncidentResponse) error {
	payload := models.ValidationEventPayload{
		ValidationID:   validation.ID,
		IncidentID:     validation.IncidentID,
		Status:         validation.Status.String(),
		ValidatorKind:  validation.ValidatorKind.String(),
		EventTimestamp: time.Now().UTC(),
	}

	if err := messaging.PublishEvent("validada", payload); err != nil {
		return fmt.Errorf("failed to publish validada event: %w", err)
	}

	log.Printf("📤 Published 'incidencia_validada' event for %s", validation.IncidentID)
	return nil
}

func publishRejectionEvent(validation *models.Validation, incident *clients.IncidentResponse) error {
	payload := models.ValidationEventPayload{
		ValidationID:   validation.ID,
		IncidentID:     validation.IncidentID,
		Status:         validation.Status.String(),
		Reason:         validation.Reason,
		ValidatorKind:  validation.ValidatorKind.String(),
		EventTimestamp: time.Now().UTC(),
	}

	if err := messaging.PublishEvent("rechazada", payload); err != nil {
		return fmt.Errorf("failed to publish rechazada event: %w", err)
	}

	log.Printf("📤 Published 'incidencia_rechazada' event for %s", validation.IncidentID)
	return nil
}
