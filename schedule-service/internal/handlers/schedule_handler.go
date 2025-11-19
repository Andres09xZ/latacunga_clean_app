package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ScheduleHandler struct {
	db *gorm.DB
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(db *gorm.DB) *ScheduleHandler {
	return &ScheduleHandler{db: db}
}

// CreateShift creates a new shift
// @Summary Create a new shift
// @Description Create a new shift in the schedule
// @Tags shifts
// @Accept json
// @Produce json
// @Param request body models.CreateShiftRequest true "Shift details"
// @Success 201 {object} models.ShiftResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/shifts [post]
func (h *ScheduleHandler) CreateShift(c *gin.Context) {
	var req models.CreateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate times
	if req.StartTime.After(req.EndTime) || req.StartTime.Equal(req.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time must be before end_time"})
		return
	}

	shift := models.Shift{
		ID:        uuid.New(),
		Date:      req.Date,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		MaxTasks:  req.MaxTasks,
		Status:    models.ShiftStatusActive,
	}

	if err := h.db.Create(&shift).Error; err != nil {
		log.Printf("❌ Error creating shift: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shift"})
		return
	}

	log.Printf("✅ Shift created: %s (%s)", shift.ID, shift.Date.Format("2006-01-02"))
	c.JSON(http.StatusCreated, shift.ToResponse())
}

// ListShifts lists all shifts with optional date filter
// @Summary List all shifts
// @Description Get all shifts, optionally filtered by date
// @Tags shifts
// @Accept json
// @Produce json
// @Param date query string false "Filter by date (YYYY-MM-DD)"
// @Success 200 {array} models.ShiftResponse
// @Failure 500 {object} map[string]string
// @Router /api/v1/shifts [get]
func (h *ScheduleHandler) ListShifts(c *gin.Context) {
	dateFilter := c.Query("date")

	var shifts []models.Shift
	query := h.db

	if dateFilter != "" {
		date, err := time.Parse("2006-01-02", dateFilter)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		nextDay := date.Add(24 * time.Hour)
		query = query.Where("date >= ? AND date < ?", date, nextDay)
	}

	if err := query.Order("date DESC").Find(&shifts).Error; err != nil {
		log.Printf("❌ Error listing shifts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shifts"})
		return
	}

	responses := make([]models.ShiftResponse, len(shifts))
	for i, shift := range shifts {
		responses[i] = shift.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetShift gets a specific shift by ID
// @Summary Get a shift by ID
// @Description Get details of a specific shift
// @Tags shifts
// @Accept json
// @Produce json
// @Param id path string true "Shift ID"
// @Success 200 {object} models.ShiftResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/shifts/{id} [get]
func (h *ScheduleHandler) GetShift(c *gin.Context) {
	id := c.Param("id")

	var shift models.Shift
	if err := h.db.First(&shift, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "shift not found"})
			return
		}
		log.Printf("❌ Error getting shift: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get shift"})
		return
	}

	c.JSON(http.StatusOK, shift.ToResponse())
}

// UpdateShift updates a shift
// @Summary Update a shift
// @Description Update shift details
// @Tags shifts
// @Accept json
// @Produce json
// @Param id path string true "Shift ID"
// @Param request body models.CreateShiftRequest true "Updated shift details"
// @Success 200 {object} models.ShiftResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/shifts/{id} [put]
func (h *ScheduleHandler) UpdateShift(c *gin.Context) {
	id := c.Param("id")

	var req models.CreateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var shift models.Shift
	if err := h.db.First(&shift, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "shift not found"})
			return
		}
		log.Printf("❌ Error getting shift: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get shift"})
		return
	}

	shift.Date = req.Date
	shift.StartTime = req.StartTime
	shift.EndTime = req.EndTime
	shift.MaxTasks = req.MaxTasks

	if err := h.db.Save(&shift).Error; err != nil {
		log.Printf("❌ Error updating shift: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update shift"})
		return
	}

	log.Printf("✅ Shift updated: %s", shift.ID)
	c.JSON(http.StatusOK, shift.ToResponse())
}

// DeleteShift deletes a shift
// @Summary Delete a shift
// @Description Delete a shift by ID
// @Tags shifts
// @Accept json
// @Produce json
// @Param id path string true "Shift ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /api/v1/shifts/{id} [delete]
func (h *ScheduleHandler) DeleteShift(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&models.Shift{}, "id = ?", id).Error; err != nil {
		log.Printf("❌ Error deleting shift: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete shift"})
		return
	}

	log.Printf("✅ Shift deleted: %s", id)
	c.Status(http.StatusNoContent)
}

// GetDailySchedule gets all tasks for a specific date
// @Summary Get daily schedule
// @Description Get all tasks scheduled for a specific date
// @Tags schedule
// @Accept json
// @Produce json
// @Param date path string true "Date (YYYY-MM-DD)"
// @Success 200 {object} models.DailyScheduleResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/schedule/{date} [get]
func (h *ScheduleHandler) GetDailySchedule(c *gin.Context) {
	dateStr := c.Param("date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	// Get shifts for the date
	var shifts []models.Shift
	nextDay := date.Add(24 * time.Hour)
	if err := h.db.Where("date >= ? AND date < ?", date, nextDay).Order("start_time ASC").Find(&shifts).Error; err != nil {
		log.Printf("❌ Error getting shifts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get daily schedule"})
		return
	}

	// Get tasks for the date
	var tasks []models.ScheduleTask
	if err := h.db.Where("date >= ? AND date < ?", date, nextDay).Order("priority DESC").Find(&tasks).Error; err != nil {
		log.Printf("❌ Error getting tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get daily schedule"})
		return
	}

	// Build response
	shiftResponses := make([]models.ShiftResponse, len(shifts))
	for i, shift := range shifts {
		shiftResponses[i] = shift.ToResponse()
	}

	taskResponses := make([]models.ScheduleTaskResponse, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = task.ToResponse()
	}

	response := models.DailyScheduleResponse{
		Date:   date,
		Shifts: shiftResponses,
		Tasks:  taskResponses,
	}

	c.JSON(http.StatusOK, response)
}

// AssignTaskToShift manually assigns a task to a shift
// @Summary Assign task to shift
// @Description Manually assign a task to a specific shift
// @Tags schedule
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param shift_id query string true "Shift ID"
// @Success 200 {object} models.ScheduleTaskResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/tasks/{id}/assign [post]
func (h *ScheduleHandler) AssignTaskToShift(c *gin.Context) {
	taskID := c.Param("id")
	shiftID := c.Query("shift_id")

	if shiftID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shift_id is required"})
		return
	}

	var task models.ScheduleTask
	if err := h.db.First(&task, "id = ?", taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		log.Printf("❌ Error getting task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get task"})
		return
	}

	shiftUUID, err := uuid.Parse(shiftID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift_id format"})
		return
	}

	var shift models.Shift
	if err := h.db.First(&shift, "id = ?", shiftUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "shift not found"})
			return
		}
		log.Printf("❌ Error getting shift: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get shift"})
		return
	}

	// Update task assignment
	task.ShiftID = &shiftUUID
	task.Status = models.TaskStatusScheduled

	if err := h.db.Save(&task).Error; err != nil {
		log.Printf("❌ Error assigning task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign task"})
		return
	}

	// Publish event
	if err := messaging.PublishEvent("tarea_planificada", task.ToResponse()); err != nil {
		log.Printf("⚠️  Warning: Failed to publish event: %v", err)
	}

	log.Printf("✅ Task assigned to shift: %s -> %s", taskID, shiftID)
	c.JSON(http.StatusOK, task.ToResponse())
}

// HandleTaskCreatedEvent handles incoming task creation events from RabbitMQ
func (h *ScheduleHandler) HandleTaskCreatedEvent(event *messaging.TaskCreatedEvent) error {
	// Parse report day
	reportDay := event.ReportDay

	// Try to find an existing shift for this date
	var existingShift models.Shift
	nextDay := reportDay.Add(24 * time.Hour)
	err := h.db.Where("date >= ? AND date < ? AND status = ?", reportDay, nextDay, models.ShiftStatusActive).
		Order("end_time DESC").
		First(&existingShift).Error

	taskID := event.ID
	var taskStatus models.TaskStatus
	var shiftID *uuid.UUID

	if err == gorm.ErrRecordNotFound {
		// No shift exists for this date, create default shift if needed
		log.Printf("⚠️  No shift found for date %s, creating default shift", reportDay.Format("2006-01-02"))

		// Create a default shift (e.g., 08:00 - 17:00)
		startTime := time.Date(reportDay.Year(), reportDay.Month(), reportDay.Day(), 8, 0, 0, 0, reportDay.Location())
		endTime := time.Date(reportDay.Year(), reportDay.Month(), reportDay.Day(), 17, 0, 0, 0, reportDay.Location())

		defaultShift := models.Shift{
			ID:        uuid.New(),
			Date:      reportDay,
			StartTime: startTime,
			EndTime:   endTime,
			MaxTasks:  10,
			Status:    models.ShiftStatusActive,
		}

		if err := h.db.Create(&defaultShift).Error; err != nil {
			log.Printf("❌ Error creating default shift: %v", err)
			taskStatus = models.TaskStatusUnscheduled
		} else {
			taskStatus = models.TaskStatusScheduled
			shiftID = &defaultShift.ID
		}
	} else if err != nil {
		log.Printf("❌ Error querying shifts: %v", err)
		taskStatus = models.TaskStatusUnscheduled
	} else {
		// Shift exists, check if it has capacity
		var taskCount int64
		h.db.Model(&models.ScheduleTask{}).Where("shift_id = ?", existingShift.ID).Count(&taskCount)

		if taskCount < int64(existingShift.MaxTasks) {
			taskStatus = models.TaskStatusScheduled
			shiftID = &existingShift.ID
			log.Printf("✅ Task assigned to existing shift: %s", existingShift.ID)
		} else {
			log.Printf("⚠️  Shift at capacity, marking task as not scheduled")
			taskStatus = models.TaskStatusUnscheduled
		}
	}

	// Create schedule task record
	scheduleTask := models.ScheduleTask{
		ID:        uuid.New(),
		TaskID:    taskID,
		ShiftID:   shiftID,
		Date:      reportDay,
		Status:    taskStatus,
		TaskType:  event.TaskType,
		Priority:  event.Priority,
		Location:  event.Location,
		Latitude:  event.Latitude,
		Longitude: event.Longitude,
	}

	if err := h.db.Create(&scheduleTask).Error; err != nil {
		log.Printf("❌ Error creating schedule task: %v", err)
		return fmt.Errorf("failed to create schedule task: %w", err)
	}

	log.Printf("✅ Schedule task created: %s (Status: %s)", scheduleTask.ID, taskStatus)

	// Publish response event
	eventType := "tarea_planificada"
	if taskStatus == models.TaskStatusUnscheduled {
		eventType = "tarea_no_planificada"
	}

	if err := messaging.PublishEvent(eventType, map[string]interface{}{
		"task_id":   taskID,
		"status":    taskStatus,
		"shift_id":  shiftID,
		"date":      reportDay,
		"timestamp": time.Now().UTC(),
	}); err != nil {
		log.Printf("⚠️  Warning: Failed to publish event: %v", err)
	}

	return nil
}
