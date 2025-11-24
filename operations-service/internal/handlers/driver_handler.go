package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetActiveOrder obtiene la orden activa para un conductor
//
//	@Summary		Get active work order for driver
//	@Description	Retrieves the active (ASIGNADA or EN_PROGRESO) work order for a specific driver
//	@Tags			driver
//	@Accept			json
//	@Produce		json
//	@Param			driver_id	query		string	true	"Driver UUID"
//	@Success		200			{object}	models.WorkOrderResponse
//	@Success		204			{object}	nil	"No active orders"
//	@Failure		400			{object}	models.ErrorResponse
//	@Failure		500			{object}	models.ErrorResponse
//	@Router			/api/v1/driver/orders/active [get]
func GetActiveOrder(c *gin.Context) {
	driverIDStr := c.Query("driver_id")
	if driverIDStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "driver_id is required",
			Message: "Please provide driver_id as query parameter",
		})
		return
	}

	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid driver_id",
			Message: "Driver ID must be a valid UUID",
		})
		return
	}

	var workOrder models.WorkOrder
	err = database.DB.
		Preload("Stops").
		Where("driver_id = ? AND status IN ?", driverID, []string{
			string(models.WorkOrderStatusAssigned),
			string(models.WorkOrderStatusInProgress),
		}).
		Order("assigned_at DESC").
		First(&workOrder).Error

	if err != nil {
		// No hay orden activa
		c.Status(http.StatusNoContent)
		return
	}

	response := models.WorkOrderResponse{
		WorkOrder: workOrder,
		Stops:     workOrder.Stops,
	}

	c.JSON(http.StatusOK, response)
}

// StartOrder inicia una orden de trabajo
//
//	@Summary		Start work order
//	@Description	Marks a work order as IN_PROGRESS and records start time
//	@Tags			driver
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Work Order UUID"
//	@Param			request	body		models.StartOrderRequest	true	"Start location"
//	@Success		200		{object}	models.SuccessResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/api/v1/driver/orders/{id}/start [post]
func StartOrder(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid order_id",
			Message: "Order ID must be a valid UUID",
		})
		return
	}

	var req models.StartOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
		})
		return
	}

	var workOrder models.WorkOrder
	if err := database.DB.First(&workOrder, "id = ?", orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "work order not found",
			Message: "The specified work order does not exist",
		})
		return
	}

	// Validar que esté en estado ASIGNADA
	if workOrder.Status != models.WorkOrderStatusAssigned {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid status",
			Message: "Work order must be in ASIGNADA status to start",
		})
		return
	}

	// Actualizar estado
	now := time.Now()
	workOrder.Status = models.WorkOrderStatusInProgress
	workOrder.StartedAt = &now

	if err := database.DB.Save(&workOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to update work order",
			Message: err.Error(),
		})
		return
	}

	log.Printf("Work order started: id=%s, driver=%s", workOrder.ID, workOrder.DriverID)

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Work order started successfully",
		Data:    workOrder,
	})
}

// CompleteStop marca una parada como completada
//
//	@Summary		Complete a stop
//	@Description	Marks a work order stop as RECOGIDO or NO_RECOGIDO
//	@Tags			driver
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Stop ID"
//	@Param			request	body		models.CompleteStopRequest	true	"Stop completion data"
//	@Success		200		{object}	models.SuccessResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/api/v1/driver/stops/{id}/complete [post]
func CompleteStop(c *gin.Context) {
	stopIDStr := c.Param("id")
	stopID, err := parseUint(stopIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid stop_id",
			Message: "Stop ID must be a valid number",
		})
		return
	}

	var req models.CompleteStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
		})
		return
	}

	// Buscar la parada
	var stop models.WorkOrderStop
	if err := database.DB.First(&stop, stopID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "stop not found",
			Message: "The specified stop does not exist",
		})
		return
	}

	// Validar que esté pendiente
	if stop.Status != models.StopStatusPending {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid status",
			Message: "Stop must be in PENDIENTE status",
		})
		return
	}

	// Iniciar transacción
	tx := database.DB.Begin()

	// Actualizar la parada
	now := time.Now()
	stop.Status = req.Status
	stop.ServicedAt = &now

	if err := tx.Save(&stop).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to update stop",
			Message: err.Error(),
		})
		return
	}

	// Si fue recogido, incrementar contador
	if req.Status == models.StopStatusCollected {
		if err := tx.Model(&models.WorkOrder{}).
			Where("id = ?", stop.WorkOrderID).
			UpdateColumn("completed_stops", database.DB.Raw("completed_stops + 1")).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "failed to update work order",
				Message: err.Error(),
			})
			return
		}
	}

	tx.Commit()

	log.Printf("Stop completed: id=%d, status=%s", stop.ID, stop.Status)

	// Publicar evento (opcional - no bloqueante)
	go func() {
		payload := map[string]interface{}{
			"event_type":      "stop.serviced",
			"stop_id":         stop.ID,
			"work_order_id":   stop.WorkOrderID.String(),
			"incident_ref_id": stop.IncidentRefID.String(),
			"status":          string(stop.Status),
		}
		if err := messaging.PublishEvent("stop.serviced.v1", payload); err != nil {
			log.Printf("Failed to publish stop.serviced event: %v", err)
		}
	}()

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Stop completed successfully",
		Data:    stop,
	})
}

// FinishOrder finaliza una orden de trabajo
//
//	@Summary		Finish work order
//	@Description	Marks a work order as COMPLETADA, validates all stops are processed, and publishes completion event
//	@Tags			driver
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Work Order UUID"
//	@Param			request	body		models.FinishOrderRequest	true	"End location"
//	@Success		200		{object}	models.SuccessResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/api/v1/driver/orders/{id}/finish [post]
func FinishOrder(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid order_id",
			Message: "Order ID must be a valid UUID",
		})
		return
	}

	var req models.FinishOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
		})
		return
	}

	// Buscar la orden con sus paradas
	var workOrder models.WorkOrder
	if err := database.DB.Preload("Stops").First(&workOrder, "id = ?", orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "work order not found",
			Message: "The specified work order does not exist",
		})
		return
	}

	// Validar que esté en progreso
	if workOrder.Status != models.WorkOrderStatusInProgress {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid status",
			Message: "Work order must be in EN_PROGRESO status to finish",
		})
		return
	}

	// Validar que todas las paradas estén procesadas
	for _, stop := range workOrder.Stops {
		if stop.Status == models.StopStatusPending {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "incomplete stops",
				Message: "All stops must be processed before finishing the work order",
			})
			return
		}
	}

	// Actualizar estado
	now := time.Now()
	workOrder.Status = models.WorkOrderStatusCompleted
	workOrder.CompletedAt = &now

	if err := database.DB.Save(&workOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to update work order",
			Message: err.Error(),
		})
		return
	}

	log.Printf("Work order completed: id=%s, driver=%s, zone=%d", workOrder.ID, workOrder.DriverID, workOrder.ZoneID)

	// CRÍTICO: Publicar evento de finalización
	go func() {
		payload := map[string]interface{}{
			"event_type":      "workorder.completed",
			"work_order_id":   workOrder.ID.String(),
			"driver_id":       workOrder.DriverID.String(),
			"zone_id":         workOrder.ZoneID,
			"status":          "COMPLETED",
			"total_stops":     workOrder.TotalStops,
			"completed_stops": workOrder.CompletedStops,
		}
		if err := messaging.PublishEvent("workorder.completed.v1", payload); err != nil {
			log.Printf("ERROR: Failed to publish workorder.completed event: %v", err)
		} else {
			log.Printf("Published workorder.completed.v1 event for zone %d, driver %s", workOrder.ZoneID, workOrder.DriverID)
		}
	}()

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Work order finished successfully",
		Data:    workOrder,
	})
}

// Helper function
func parseUint(s string) (uint, error) {
	var id uint
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}
