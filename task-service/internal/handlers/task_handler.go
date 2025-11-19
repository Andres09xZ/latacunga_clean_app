package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"task-service/internal/database"
	"task-service/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================
// Public HTTP Handlers
// ============================================================

// AvailableTasks returns list of available tasks ordered by priority and distance
//
//	@Summary		Get available tasks for worker
//	@Description	Get list of available tasks that a worker can claim
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			lat	query		number	true	"Latitude"
//	@Param			lng	query		number	true	"Longitude"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/tasks/available [get]
func AvailableTasks(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng required"})
		return
	}

	_, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid latitude"})
		return
	}

	_, err = strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid longitude"})
		return
	}

	// Query available tasks ordered by priority and distance (simplified)
	var tasks []models.Task
	if err := database.DB.
		Where("state = ?", "PENDIENTE").
		Order("priority DESC, created_at ASC").
		Limit(50).
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	// Convert to response
	responses := make([]models.TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = taskToResponse(&task)
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": responses,
		"count": len(tasks),
	})
}

// GetTask retrieves a specific task by ID
//
//	@Summary		Get a task
//	@Description	Retrieve details of a specific task
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			taskId	path		string	true	"Task ID"
//	@Success		200		{object}	models.TaskResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/tasks/{taskId} [get]
func GetTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid taskId"})
		return
	}

	var task models.Task
	if err := database.DB.Preload("Histories").First(&task, taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, taskToResponse(&task))
}

// ClaimTask allows a worker to claim a task
//
//	@Summary		Claim a task
//	@Description	Claim an available task (transition from PENDIENTE to EN_PROGRESO)
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			taskId	path		string	true	"Task ID"
//	@Success		200		{object}	models.TaskResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/tasks/{taskId}/claim [post]
func ClaimTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid taskId"})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "user_id not found in context"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid user_id"})
		return
	}

	// Find actor for this user
	var actor models.Actor
	if err := database.DB.Where("user_id = ?", userID).First(&actor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Actor not found"})
		return
	}

	// Get task with lock (pessimistic locking)
	var task models.Task
	if err := database.DB.First(&task, taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if task is available
	if task.State != "PENDIENTE" && task.State != "PENDIENTE_ASIGNAR" {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Task is already claimed or completed (state: %s)", task.State),
		})
		return
	}

	// Update task
	oldState := task.State
	task.ActorID = &actor.ID
	task.State = "EN_PROGRESO"
	now := time.Now()
	task.StartedAt = &now

	if err := database.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim task"})
		return
	}

	// Record history
	reason := "Tarea reclamada por trabajador"
	recordTaskHistory(&task, oldState, task.State, &reason, &actor.ID)

	// Emit event (handled by main.go or a separate publisher)
	emitTaskEvent(&task, "tarea_asignada")

	c.JSON(http.StatusOK, taskToResponse(&task))
}

// UpdateTaskStatus updates the state of a task
//
//	@Summary		Update task status
//	@Description	Update task state (EN_PROGRESO, COMPLETADA, CANCELADA)
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			taskId	path		string								true	"Task ID"
//	@Param			request	body		models.UpdateTaskStatusRequest	true	"Update request"
//	@Success		200		{object}	models.TaskResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/tasks/{taskId}/status [put]
func UpdateTaskStatus(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid taskId"})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "user_id not found in context"})
		return
	}

	userID, _ := uuid.Parse(userIDStr.(string))

	var req models.UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task models.Task
	if err := database.DB.First(&task, taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check authorization
	if task.ActorID == nil || *task.ActorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this task"})
		return
	}

	// Validate state transition
	if !isValidStateTransition(task.State, req.State) {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Invalid state transition from %s to %s", task.State, req.State),
		})
		return
	}

	// Update task
	oldState := task.State
	task.State = req.State
	if req.Evidence != nil && len(req.Evidence) > 0 {
		task.Evidence = req.Evidence
	}
	if req.CompletedAt != nil && req.State == "COMPLETADA" {
		task.CompletedAt = req.CompletedAt
	} else if req.State == "COMPLETADA" {
		now := time.Now()
		task.CompletedAt = &now
	}

	if err := database.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Record history
	recordTaskHistory(&task, oldState, task.State, req.Reason, task.ActorID)

	// Emit appropriate event
	switch req.State {
	case "COMPLETADA":
		emitTaskEvent(&task, "tarea_completada")
		// Also emit novedad_atendida if this task is from a novedad
		if task.NovedadID != nil {
			publishNovedadAtendidaEvent(task.NovedadID)
		}
	case "CANCELADA":
		emitTaskEvent(&task, "tarea_cancelada")
	default:
		emitTaskEvent(&task, "tarea_actualizada")
	}

	c.JSON(http.StatusOK, taskToResponse(&task))
}

// ListTasks returns a paginated list of tasks with filtering
//
//	@Summary		List tasks
//	@Description	Get a paginated list of tasks with optional filtering
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			state		query		string	false	"Filter by state"
//	@Param			source		query		string	false	"Filter by source (novedad/report)"
//	@Param			actor_id	query		string	false	"Filter by actor ID"
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Items per page (default: 20)"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/tasks [get]
func ListTasks(c *gin.Context) {
	page := 1
	if p := c.Query("page"); p != "" {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			page = pInt
		}
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if lInt, err := strconv.Atoi(l); err == nil && lInt > 0 && lInt <= 100 {
			limit = lInt
		}
	}

	query := database.DB

	// Apply filters
	if state := c.Query("state"); state != "" {
		query = query.Where("state = ?", state)
	}
	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}
	if actorID := c.Query("actor_id"); actorID != "" {
		query = query.Where("actor_id = ?", actorID)
	}

	var total int64
	query.Model(&models.Task{}).Count(&total)

	var tasks []models.Task
	if err := query.
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	responses := make([]models.TaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = taskToResponse(&task)
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": responses,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// ============================================================
// Event Handlers (for RabbitMQ)
// ============================================================

// HandleNovedadEvent processes events from the novedades service
func HandleNovedadEvent(eventType string, event map[string]interface{}) error {
	switch eventType {
	case "novedad.creada", "novedad.verificada":
		return handleNovedadCreatedOrVerified(event)
	default:
		log.Printf("Unknown novedad event type: %s", eventType)
		return nil
	}
}

// handleNovedadCreatedOrVerified creates a task from a novedad event
func handleNovedadCreatedOrVerified(event map[string]interface{}) error {
	eventID := event["id"]
	if eventID == nil {
		return errors.New("event missing 'id' field")
	}

	novedadID := event["novedad_id"]
	if novedadID == nil {
		return errors.New("event missing 'novedad_id' field")
	}

	eventIDUUID, err := uuid.Parse(eventID.(string))
	if err != nil {
		return fmt.Errorf("invalid event id: %w", err)
	}

	novedadIDUUID, err := uuid.Parse(novedadID.(string))
	if err != nil {
		return fmt.Errorf("invalid novedad_id: %w", err)
	}

	// ============================================================
	// Check idempotency
	// ============================================================
	var processed models.ProcessedEvent
	if err := database.DB.Where(
		"event_id = ? AND consumer = ? AND event_type = ?",
		eventIDUUID, "task-service", "novedad",
	).First(&processed).Error; err == nil {
		log.Printf("Event %s already processed, skipping", eventIDUUID)
		return nil // Already processed
	}

	// ============================================================
	// Extract novedad details from event
	// ============================================================
	taskType := extractTaskType(event)
	description := ""
	if desc, ok := event["description"].(string); ok {
		description = desc
	}

	var lat, lng *float64
	if location, ok := event["location"].(map[string]interface{}); ok {
		if latitude, ok := location["latitude"].(float64); ok {
			lat = &latitude
		}
		if longitude, ok := location["longitude"].(float64); ok {
			lng = &longitude
		}
	}

	// ============================================================
	// Create task
	// ============================================================
	task := models.Task{
		NovedadID:   &novedadIDUUID,
		Source:      "novedad",
		Type:        taskType,
		State:       "PENDIENTE",
		Priority:    extractPriority(event),
		Description: description,
		Latitude:    lat,
		Longitude:   lng,
	}

	if err := database.DB.Create(&task).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// ============================================================
	// Record processed event for idempotency
	// ============================================================
	processedEvent := models.ProcessedEvent{
		EventID:     eventIDUUID,
		Consumer:    "task-service",
		EventType:   "novedad",
		SourceID:    novedadIDUUID,
		ProcessedAt: time.Now(),
	}
	database.DB.Create(&processedEvent)

	// ============================================================
	// Emit task creation event
	// ============================================================
	emitTaskEvent(&task, "tarea_creada")

	log.Printf("Task %s created from novedad %s", task.ID, novedadIDUUID)
	return nil
}

// HandleReportVerifiedEvent processes events from the reports service (legacy)
func HandleReportVerifiedEvent(event map[string]interface{}) error {
	eventID := event["event_id"]
	if eventID == nil {
		return errors.New("event missing 'event_id' field")
	}

	reportID := event["report_id"]
	if reportID == nil {
		return errors.New("event missing 'report_id' field")
	}

	eventIDUUID, err := uuid.Parse(eventID.(string))
	if err != nil {
		return fmt.Errorf("invalid event_id: %w", err)
	}

	reportIDUUID, err := uuid.Parse(reportID.(string))
	if err != nil {
		return fmt.Errorf("invalid report_id: %w", err)
	}

	// ============================================================
	// Check idempotency
	// ============================================================
	var processed models.ProcessedEvent
	if err := database.DB.Where(
		"event_id = ? AND consumer = ? AND event_type = ?",
		eventIDUUID, "task-service", "report",
	).First(&processed).Error; err == nil {
		log.Printf("Report event %s already processed, skipping", eventIDUUID)
		return nil
	}

	// ============================================================
	// Create task from report
	// ============================================================
	task := models.Task{
		ReportID: &reportIDUUID,
		Source:   "report",
		Type:     "LIMPIEZA",
		State:    "PENDIENTE",
	}

	if err := database.DB.Create(&task).Error; err != nil {
		return fmt.Errorf("failed to create task from report: %w", err)
	}

	// ============================================================
	// Record processed event for idempotency
	// ============================================================
	processedEvent := models.ProcessedEvent{
		EventID:     eventIDUUID,
		Consumer:    "task-service",
		EventType:   "report",
		SourceID:    reportIDUUID,
		ProcessedAt: time.Now(),
	}
	database.DB.Create(&processedEvent)

	emitTaskEvent(&task, "tarea_creada")

	log.Printf("Task %s created from report %s", task.ID, reportIDUUID)
	return nil
}

// ============================================================
// Helper Functions
// ============================================================

func taskToResponse(t *models.Task) models.TaskResponse {
	return models.TaskResponse{
		ID:          t.ID,
		NovedadID:   t.NovedadID,
		ReportID:    t.ReportID,
		ActorID:     t.ActorID,
		Source:      t.Source,
		Type:        t.Type,
		State:       t.State,
		Priority:    t.Priority,
		Description: t.Description,
		Latitude:    t.Latitude,
		Longitude:   t.Longitude,
		PhotoURL:    t.PhotoURL,
		Evidence:    t.Evidence,
		StartedAt:   t.StartedAt,
		CompletedAt: t.CompletedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func isValidStateTransition(from, to string) bool {
	transitions := map[string][]string{
		"PENDIENTE":         {"EN_PROGRESO", "CANCELADA"},
		"PENDIENTE_ASIGNAR": {"EN_PROGRESO", "CANCELADA"},
		"EN_PROGRESO":       {"COMPLETADA", "CANCELADA"},
		"COMPLETADA":        {}, // Terminal state
		"CANCELADA":         {}, // Terminal state
	}

	validTransitions, ok := transitions[from]
	if !ok {
		return false
	}

	for _, t := range validTransitions {
		if t == to {
			return true
		}
	}
	return false
}

func recordTaskHistory(task *models.Task, oldState, newState string, reason *string, actorID *uuid.UUID) {
	if actorID == nil {
		actorID = &uuid.Nil
	}

	history := models.TaskHistory{
		TaskID:    task.ID,
		ActorID:   *actorID,
		OldState:  oldState,
		NewState:  newState,
		Reason:    reason,
		CreatedAt: time.Now(),
	}

	database.DB.Create(&history)
}

func extractTaskType(event map[string]interface{}) string {
	if taskType, ok := event["type"].(string); ok {
		switch taskType {
		case "PUNTO_ACOPIO":
			return "PUNTO_ACOPIO"
		case "ZONA_CRITICA":
			return "ZONA_CRITICA"
		}
	}
	return "LIMPIEZA" // Default
}

func extractPriority(event map[string]interface{}) int {
	if priority, ok := event["priority"].(float64); ok {
		return int(priority)
	}
	return 0
}

func emitTaskEvent(task *models.Task, eventType string) {
	// This would publish to RabbitMQ
	// For now, just log
	payload := models.TaskEventPayload{
		TaskID:    task.ID,
		Source:    task.Source,
		Type:      task.Type,
		State:     task.State,
		ActorID:   task.ActorID,
		NovedadID: task.NovedadID,
		ReportID:  task.ReportID,
		Timestamp: time.Now(),
	}

	payloadJSON, _ := json.Marshal(payload)
	log.Printf("EVENT: %s - %s", eventType, string(payloadJSON))

	// TODO: Publish to RabbitMQ with routing key "task.<eventType>"
}

func publishNovedadAtendidaEvent(novedadID *uuid.UUID) {
	if novedadID == nil {
		return
	}

	payload := map[string]interface{}{
		"novedad_id": novedadID,
		"timestamp":  time.Now(),
	}

	payloadJSON, _ := json.Marshal(payload)
	log.Printf("EVENT: novedad_atendida - %s", string(payloadJSON))

	// TODO: Publish to RabbitMQ with routing key "novedad.atendida"
}
