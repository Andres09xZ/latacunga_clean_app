package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/fleet-service/internal/database"
	"github.com/fleet-service/internal/models"
	"github.com/fleet-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShiftHandler maneja las operaciones de turnos
type ShiftHandler struct {
	publisher *services.EventPublisher
}

// NewShiftHandler crea una nueva instancia de ShiftHandler
func NewShiftHandler(publisher *services.EventPublisher) *ShiftHandler {
	return &ShiftHandler{
		publisher: publisher,
	}
}

// ClockInRequest representa la solicitud de clock-in
type ClockInRequest struct {
	DriverID   string `json:"driver_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	TruckPlate string `json:"truck_plate" binding:"required" example:"ABC-1234"`
}

// ClockInResponse representa la respuesta de clock-in
type ClockInResponse struct {
	ShiftID uuid.UUID `json:"shift_id" example:"550e8400-e29b-41d4-a716-446655440099"`
	Message string    `json:"message" example:"Turno iniciado exitosamente"`
}

// ClockOutRequest representa la solicitud de clock-out
type ClockOutRequest struct {
	DriverID string `json:"driver_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ClockOutResponse representa la respuesta de clock-out
type ClockOutResponse struct {
	Message string `json:"message" example:"Turno finalizado exitosamente"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error string `json:"error" example:"Mensaje de error descriptivo"`
}

// ClockIn registra el inicio de un turno
// @Summary Iniciar turno (Clock-in)
// @Description Registra el inicio de turno de un conductor con un camión asignado
// @Tags Turnos
// @Accept json
// @Produce json
// @Param request body ClockInRequest true "Datos del clock-in"
// @Success 200 {object} ClockInResponse "Turno iniciado exitosamente"
// @Failure 400 {object} ErrorResponse "Datos inválidos o ID de conductor inválido"
// @Failure 404 {object} ErrorResponse "Conductor no encontrado o Camión no encontrado"
// @Failure 409 {object} ErrorResponse "El conductor ya tiene un turno activo o El camión no está disponible"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /shifts/clock-in [post]
func (h *ShiftHandler) ClockIn(c *gin.Context) {
	var req ClockInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Parsear UUID del driver
	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de conductor inválido"})
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

	// 1. Verificar que el conductor existe
	var driver models.Driver
	if err := tx.First(&driver, "id = ?", driverID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conductor no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al buscar conductor"})
		return
	}

	// 2. Verificar que el conductor no tenga turno activo
	var existingShift models.ActiveShift
	err = tx.Where("driver_id = ? AND is_active = ?", driverID, true).First(&existingShift).Error
	if err == nil {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "El conductor ya tiene un turno activo"})
		return
	}

	// 3. Buscar el camión por placa
	var truck models.Truck
	if err := tx.Where("plate = ?", req.TruckPlate).First(&truck).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Camión no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al buscar camión"})
		return
	}

	// 4. Verificar que el camión esté disponible
	if truck.Status != models.TruckStatusDisponible {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "El camión no está disponible (Estado: " + string(truck.Status) + ")"})
		return
	}

	// 5. Crear el turno activo
	shift := models.ActiveShift{
		DriverID:  driverID,
		TruckID:   truck.ID,
		StartTime: time.Now(),
		IsActive:  true,
	}

	if err := tx.Create(&shift).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear turno"})
		log.Printf("Error al crear turno: %v", err)
		return
	}

	// 6. Actualizar estado del camión a EN_USO
	if err := tx.Model(&truck).Update("status", models.TruckStatusEnUso).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado del camión"})
		log.Printf("Error al actualizar camión: %v", err)
		return
	}

	// 7. Actualizar estado del conductor a DISPONIBLE
	if err := tx.Model(&driver).Update("status", models.DriverStatusDisponible).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado del conductor"})
		log.Printf("Error al actualizar conductor: %v", err)
		return
	}

	// Commit de la transacción
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar la transacción"})
		log.Printf("Error en commit: %v", err)
		return
	}

	// 8. Publicar evento resource.available.v1
	event := services.ResourceAvailableEvent{
		DriverID:   driverID.String(),
		DriverName: driver.FullName,
		TruckID:    truck.ID,
		TruckPlate: truck.Plate,
		TruckType:  string(truck.Type),
		ShiftID:    shift.ID.String(),
		Timestamp:  time.Now(),
	}

	if err := h.publisher.PublishResourceAvailable(event); err != nil {
		log.Printf("ADVERTENCIA: Error al publicar evento resource.available: %v", err)
		// No fallar la operación si el evento no se publica
	}

	log.Printf("✓ Clock-in exitoso - Driver: %s, Truck: %s, Shift: %s", driverID, truck.Plate, shift.ID)

	c.JSON(http.StatusOK, ClockInResponse{
		ShiftID: shift.ID,
		Message: "Turno iniciado exitosamente",
	})
}

// ClockOut registra el fin de un turno
// @Summary Finalizar turno (Clock-out)
// @Description Registra el fin de turno de un conductor, liberando el camión
// @Tags Turnos
// @Accept json
// @Produce json
// @Param request body ClockOutRequest true "Datos del clock-out"
// @Success 200 {object} ClockOutResponse "Turno finalizado exitosamente"
// @Failure 400 {object} ErrorResponse "Datos inválidos o ID de conductor inválido"
// @Failure 404 {object} ErrorResponse "No se encontró un turno activo para este conductor"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /shifts/clock-out [post]
func (h *ShiftHandler) ClockOut(c *gin.Context) {
	var req ClockOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Parsear UUID del driver
	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de conductor inválido"})
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

	// 1. Buscar el turno activo del conductor
	var shift models.ActiveShift
	if err := tx.Where("driver_id = ? AND is_active = ?", driverID, true).
		Preload("Truck").
		First(&shift).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró un turno activo para este conductor"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al buscar turno activo"})
		return
	}

	// 2. Cerrar el turno
	now := time.Now()
	shift.EndTime = &now
	shift.IsActive = false

	if err := tx.Save(&shift).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cerrar turno"})
		log.Printf("Error al cerrar turno: %v", err)
		return
	}

	// 3. Actualizar estado del conductor a OFFLINE
	if err := tx.Model(&models.Driver{}).Where("id = ?", driverID).
		Update("status", models.DriverStatusOffline).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado del conductor"})
		log.Printf("Error al actualizar conductor: %v", err)
		return
	}

	// 4. Actualizar estado del camión a DISPONIBLE
	if err := tx.Model(&models.Truck{}).Where("id = ?", shift.TruckID).
		Update("status", models.TruckStatusDisponible).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado del camión"})
		log.Printf("Error al actualizar camión: %v", err)
		return
	}

	// Commit de la transacción
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar la transacción"})
		log.Printf("Error en commit: %v", err)
		return
	}

	log.Printf("✓ Clock-out exitoso - Driver: %s, Truck: %s, Shift: %s", driverID, shift.Truck.Plate, shift.ID)

	c.JSON(http.StatusOK, ClockOutResponse{
		Message: "Turno finalizado exitosamente",
	})
}

// RegisterRoutes registra las rutas del handler
func (h *ShiftHandler) RegisterRoutes(router *gin.RouterGroup) {
	shifts := router.Group("/shifts")
	{
		shifts.POST("/clock-in", h.ClockIn)
		shifts.POST("/clock-out", h.ClockOut)
	}
}
