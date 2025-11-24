package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fleet-service/internal/models"
	"github.com/fleet-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DriverHandler maneja las operaciones CRUD de conductores
type DriverHandler struct {
	service *services.DriverService
}

// NewDriverHandler crea una nueva instancia de DriverHandler
func NewDriverHandler(service *services.DriverService) *DriverHandler {
	return &DriverHandler{
		service: service,
	}
}

// UpdateDriverRequest representa la solicitud de actualización de conductor
type UpdateDriverRequest struct {
	FullName *string              `json:"full_name,omitempty" example:"Juan Pérez"`
	Status   *models.DriverStatus `json:"status,omitempty" example:"DISPONIBLE"`
}

// UpdateOperatorProfileRequest representa la solicitud de actualización de perfil
type UpdateOperatorProfileRequest struct {
	PreferredZoneID   *int  `json:"preferred_zone_id,omitempty" example:"3"`
	CanDriveLateral   *bool `json:"can_drive_lateral,omitempty" example:"true"`
	CanDrivePosterior *bool `json:"can_drive_posterior,omitempty" example:"false"`
}

// DriverResponse representa la respuesta con información del conductor
type DriverResponse struct {
	ID              string                     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FullName        string                     `json:"full_name" example:"Juan Pérez"`
	Status          models.DriverStatus        `json:"status" example:"DISPONIBLE"`
	CreatedAt       string                     `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt       string                     `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	OperatorProfile *OperatorProfileResponse   `json:"operator_profile,omitempty"`
	ActiveShifts    []ActiveShiftResponse      `json:"active_shifts,omitempty"`
}

// OperatorProfileResponse representa el perfil de operador
type OperatorProfileResponse struct {
	ID                uint   `json:"id" example:"1"`
	PreferredZoneID   int    `json:"preferred_zone_id" example:"3"`
	CanDriveLateral   bool   `json:"can_drive_lateral" example:"true"`
	CanDrivePosterior bool   `json:"can_drive_posterior" example:"false"`
}

// ActiveShiftResponse representa un turno activo
type ActiveShiftResponse struct {
	ID        string `json:"id" example:"650e8400-e29b-41d4-a716-446655440099"`
	TruckID   uint   `json:"truck_id" example:"1"`
	StartTime string `json:"start_time" example:"2024-01-15T08:00:00Z"`
	IsActive  bool   `json:"is_active" example:"true"`
}

// ListDriversResponse representa la respuesta con lista de conductores
type ListDriversResponse struct {
	Drivers []DriverResponse `json:"drivers"`
	Count   int              `json:"count" example:"5"`
}

// DriverStatsResponse representa estadísticas de un conductor
type DriverStatsResponse struct {
	DriverID        string              `json:"driver_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FullName        string              `json:"full_name" example:"Juan Pérez"`
	Status          models.DriverStatus `json:"status" example:"DISPONIBLE"`
	TotalShifts     int64               `json:"total_shifts" example:"25"`
	ActiveShifts    int64               `json:"active_shifts" example:"1"`
	CompletedShifts int64               `json:"completed_shifts" example:"24"`
}

// GetDriver obtiene un conductor por ID
// @Summary Obtener conductor
// @Description Obtiene la información de un conductor específico con su perfil y turnos
// @Tags Conductores
// @Produce json
// @Param id path string true "ID del conductor (UUID)"
// @Success 200 {object} DriverResponse "Información del conductor"
// @Failure 400 {object} ErrorResponse "ID inválido"
// @Failure 404 {object} ErrorResponse "Conductor no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /drivers/{id} [get]
func (h *DriverHandler) GetDriver(c *gin.Context) {
	idParam := c.Param("id")
	driverID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de conductor inválido"})
		return
	}

	driver, err := h.service.GetDriverByID(driverID)
	if err != nil {
		log.Printf("❌ Error al obtener conductor: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toDriverResponse(driver))
}

// ListDrivers lista todos los conductores con filtros opcionales
// @Summary Listar conductores
// @Description Obtiene la lista de todos los conductores con filtros opcionales
// @Tags Conductores
// @Produce json
// @Param status query string false "Filtrar por estado" Enums(OFFLINE, DISPONIBLE, OCUPADO)
// @Param zone_id query int false "Filtrar por zona preferida"
// @Success 200 {object} ListDriversResponse "Lista de conductores"
// @Failure 400 {object} ErrorResponse "Parámetros inválidos"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /drivers [get]
func (h *DriverHandler) ListDrivers(c *gin.Context) {
	// Leer filtros opcionales
	var statusFilter *models.DriverStatus
	if statusParam := c.Query("status"); statusParam != "" {
		status := models.DriverStatus(statusParam)
		statusFilter = &status
	}

	var zoneIDFilter *int
	if zoneIDParam := c.Query("zone_id"); zoneIDParam != "" {
		var zoneID int
		if _, err := fmt.Sscanf(zoneIDParam, "%d", &zoneID); err == nil {
			zoneIDFilter = &zoneID
		}
	}

	drivers, err := h.service.ListDrivers(statusFilter, zoneIDFilter)
	if err != nil {
		log.Printf("❌ Error al listar conductores: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convertir a respuesta
	driverResponses := make([]DriverResponse, 0, len(drivers))
	for _, driver := range drivers {
		driverResponses = append(driverResponses, toDriverResponse(&driver))
	}

	c.JSON(http.StatusOK, ListDriversResponse{
		Drivers: driverResponses,
		Count:   len(driverResponses),
	})
}

// UpdateDriver actualiza un conductor existente
// @Summary Actualizar conductor
// @Description Actualiza la información básica de un conductor
// @Tags Conductores
// @Accept json
// @Produce json
// @Param id path string true "ID del conductor (UUID)"
// @Param request body UpdateDriverRequest true "Datos a actualizar"
// @Success 200 {object} DriverResponse "Conductor actualizado exitosamente"
// @Failure 400 {object} ErrorResponse "Datos inválidos o ID inválido"
// @Failure 404 {object} ErrorResponse "Conductor no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /drivers/{id} [put]
func (h *DriverHandler) UpdateDriver(c *gin.Context) {
	idParam := c.Param("id")
	driverID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de conductor inválido"})
		return
	}

	var req UpdateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Construir mapa de actualizaciones
	updates := make(map[string]interface{})
	if req.FullName != nil {
		updates["full_name"] = *req.FullName
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se proporcionaron campos para actualizar"})
		return
	}

	driver, err := h.service.UpdateDriver(driverID, updates)
	if err != nil {
		log.Printf("❌ Error al actualizar conductor: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toDriverResponse(driver))
}

// UpdateOperatorProfile actualiza el perfil de operador de un conductor
// @Summary Actualizar perfil de operador
// @Description Actualiza las preferencias y permisos del perfil de operador
// @Tags Conductores
// @Accept json
// @Produce json
// @Param id path string true "ID del conductor (UUID)"
// @Param request body UpdateOperatorProfileRequest true "Datos del perfil a actualizar"
// @Success 200 {object} OperatorProfileResponse "Perfil actualizado exitosamente"
// @Failure 400 {object} ErrorResponse "Datos inválidos o ID inválido"
// @Failure 404 {object} ErrorResponse "Conductor o perfil no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /drivers/{id}/profile [put]
func (h *DriverHandler) UpdateOperatorProfile(c *gin.Context) {
	idParam := c.Param("id")
	driverID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de conductor inválido"})
		return
	}

	var req UpdateOperatorProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Construir mapa de actualizaciones
	updates := make(map[string]interface{})
	if req.PreferredZoneID != nil {
		updates["preferred_zone_id"] = *req.PreferredZoneID
	}
	if req.CanDriveLateral != nil {
		updates["can_drive_lateral"] = *req.CanDriveLateral
	}
	if req.CanDrivePosterior != nil {
		updates["can_drive_posterior"] = *req.CanDrivePosterior
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se proporcionaron campos para actualizar"})
		return
	}

	profile, err := h.service.UpdateOperatorProfile(driverID, updates)
	if err != nil {
		log.Printf("❌ Error al actualizar perfil: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toOperatorProfileResponse(profile))
}

// DeleteDriver elimina un conductor
// @Summary Eliminar conductor
// @Description Elimina un conductor (solo si no tiene turnos activos)
// @Tags Conductores
// @Produce json
// @Param id path string true "ID del conductor (UUID)"
// @Success 200 {object} map[string]string "Conductor eliminado exitosamente"
// @Failure 400 {object} ErrorResponse "ID inválido o conductor con turnos activos"
// @Failure 404 {object} ErrorResponse "Conductor no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /drivers/{id} [delete]
func (h *DriverHandler) DeleteDriver(c *gin.Context) {
	idParam := c.Param("id")
	driverID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de conductor inválido"})
		return
	}

	if err := h.service.DeleteDriver(driverID); err != nil {
		log.Printf("❌ Error al eliminar conductor: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conductor eliminado exitosamente"})
}

// GetAvailableDrivers obtiene conductores disponibles con turno activo
// @Summary Obtener conductores disponibles
// @Description Lista conductores con turno activo y estado disponible
// @Tags Conductores
// @Produce json
// @Param zone_id query int false "Filtrar por zona preferida"
// @Success 200 {object} ListDriversResponse "Lista de conductores disponibles"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /drivers/available [get]
func (h *DriverHandler) GetAvailableDrivers(c *gin.Context) {
	var zoneIDFilter *int
	if zoneIDParam := c.Query("zone_id"); zoneIDParam != "" {
		var zoneID int
		if _, err := fmt.Sscanf(zoneIDParam, "%d", &zoneID); err == nil {
			zoneIDFilter = &zoneID
		}
	}

	drivers, err := h.service.GetAvailableDrivers(zoneIDFilter)
	if err != nil {
		log.Printf("❌ Error al obtener conductores disponibles: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convertir a respuesta
	driverResponses := make([]DriverResponse, 0, len(drivers))
	for _, driver := range drivers {
		driverResponses = append(driverResponses, toDriverResponse(&driver))
	}

	c.JSON(http.StatusOK, ListDriversResponse{
		Drivers: driverResponses,
		Count:   len(driverResponses),
	})
}

// GetDriverStats obtiene estadísticas de un conductor
// @Summary Obtener estadísticas de conductor
// @Description Obtiene estadísticas de turnos y actividad de un conductor
// @Tags Conductores
// @Produce json
// @Param id path string true "ID del conductor (UUID)"
// @Success 200 {object} DriverStatsResponse "Estadísticas del conductor"
// @Failure 400 {object} ErrorResponse "ID inválido"
// @Failure 404 {object} ErrorResponse "Conductor no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /drivers/{id}/stats [get]
func (h *DriverHandler) GetDriverStats(c *gin.Context) {
	idParam := c.Param("id")
	driverID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de conductor inválido"})
		return
	}

	stats, err := h.service.GetDriverStats(driverID)
	if err != nil {
		log.Printf("❌ Error al obtener estadísticas: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// toDriverResponse convierte un modelo Driver a DriverResponse
func toDriverResponse(driver *models.Driver) DriverResponse {
	response := DriverResponse{
		ID:        driver.ID.String(),
		FullName:  driver.FullName,
		Status:    driver.Status,
		CreatedAt: driver.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: driver.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Agregar perfil de operador si existe
	if driver.OperatorProfile != nil {
		response.OperatorProfile = toOperatorProfileResponse(driver.OperatorProfile)
	}

	// Agregar turnos activos
	if len(driver.ActiveShifts) > 0 {
		shifts := make([]ActiveShiftResponse, 0, len(driver.ActiveShifts))
		for _, shift := range driver.ActiveShifts {
			shifts = append(shifts, ActiveShiftResponse{
				ID:        shift.ID.String(),
				TruckID:   shift.TruckID,
				StartTime: shift.StartTime.Format("2006-01-02T15:04:05Z07:00"),
				IsActive:  shift.IsActive,
			})
		}
		response.ActiveShifts = shifts
	}

	return response
}

// toOperatorProfileResponse convierte un modelo OperatorProfile a OperatorProfileResponse
func toOperatorProfileResponse(profile *models.OperatorProfile) *OperatorProfileResponse {
	return &OperatorProfileResponse{
		ID:                profile.ID,
		PreferredZoneID:   profile.PreferredZoneID,
		CanDriveLateral:   profile.CanDriveLateral,
		CanDrivePosterior: profile.CanDrivePosterior,
	}
}
