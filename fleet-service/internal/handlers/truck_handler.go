package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/fleet-service/internal/models"
	"github.com/fleet-service/internal/services"
	"github.com/gin-gonic/gin"
)

// TruckHandler maneja las operaciones CRUD de camiones
type TruckHandler struct {
	service *services.TruckService
}

// NewTruckHandler crea una nueva instancia de TruckHandler
func NewTruckHandler(service *services.TruckService) *TruckHandler {
	return &TruckHandler{
		service: service,
	}
}

// CreateTruckRequest representa la solicitud de creación de camión
type CreateTruckRequest struct {
	Plate  string              `json:"plate" binding:"required" example:"ABC-1234"`
	Type   models.TruckType    `json:"type" binding:"required" example:"CARGA_LATERAL"`
	Status models.TruckStatus  `json:"status" example:"DISPONIBLE"`
}

// UpdateTruckRequest representa la solicitud de actualización de camión
type UpdateTruckRequest struct {
	Plate  *string             `json:"plate,omitempty" example:"XYZ-5678"`
	Type   *models.TruckType   `json:"type,omitempty" example:"CARGA_POSTERIOR"`
	Status *models.TruckStatus `json:"status,omitempty" example:"MANTENIMIENTO"`
}

// TruckResponse representa la respuesta con información del camión
type TruckResponse struct {
	ID        uint               `json:"id" example:"1"`
	Plate     string             `json:"plate" example:"ABC-1234"`
	Type      models.TruckType   `json:"type" example:"CARGA_LATERAL"`
	Status    models.TruckStatus `json:"status" example:"DISPONIBLE"`
	CreatedAt string             `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt string             `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// ListTrucksResponse representa la respuesta con lista de camiones
type ListTrucksResponse struct {
	Trucks []TruckResponse `json:"trucks"`
	Count  int             `json:"count" example:"5"`
}

// CreateTruck crea un nuevo camión
// @Summary Crear camión
// @Description Registra un nuevo camión en la flota
// @Tags Camiones
// @Accept json
// @Produce json
// @Param request body CreateTruckRequest true "Datos del camión"
// @Success 201 {object} TruckResponse "Camión creado exitosamente"
// @Failure 400 {object} ErrorResponse "Datos inválidos o placa duplicada"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /trucks [post]
func (h *TruckHandler) CreateTruck(c *gin.Context) {
	var req CreateTruckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Si no se proporciona status, usar DISPONIBLE por defecto
	status := req.Status
	if status == "" {
		status = models.TruckStatusDisponible
	}

	truck := &models.Truck{
		Plate:  req.Plate,
		Type:   req.Type,
		Status: status,
	}

	if err := h.service.CreateTruck(truck); err != nil {
		log.Printf("❌ Error al crear camión: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toTruckResponse(truck))
}

// GetTruck obtiene un camión por ID
// @Summary Obtener camión
// @Description Obtiene la información de un camión específico por ID
// @Tags Camiones
// @Produce json
// @Param id path int true "ID del camión"
// @Success 200 {object} TruckResponse "Información del camión"
// @Failure 400 {object} ErrorResponse "ID inválido"
// @Failure 404 {object} ErrorResponse "Camión no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /trucks/{id} [get]
func (h *TruckHandler) GetTruck(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	truck, err := h.service.GetTruckByID(uint(id))
	if err != nil {
		log.Printf("❌ Error al obtener camión: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toTruckResponse(truck))
}

// ListTrucks lista todos los camiones con filtros opcionales
// @Summary Listar camiones
// @Description Obtiene la lista de todos los camiones con filtros opcionales
// @Tags Camiones
// @Produce json
// @Param status query string false "Filtrar por estado" Enums(DISPONIBLE, EN_USO, MANTENIMIENTO)
// @Param type query string false "Filtrar por tipo" Enums(CARGA_LATERAL, CARGA_POSTERIOR)
// @Success 200 {object} ListTrucksResponse "Lista de camiones"
// @Failure 400 {object} ErrorResponse "Parámetros inválidos"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /trucks [get]
func (h *TruckHandler) ListTrucks(c *gin.Context) {
	// Leer filtros opcionales
	var statusFilter *models.TruckStatus
	if statusParam := c.Query("status"); statusParam != "" {
		status := models.TruckStatus(statusParam)
		statusFilter = &status
	}

	var typeFilter *models.TruckType
	if typeParam := c.Query("type"); typeParam != "" {
		truckType := models.TruckType(typeParam)
		typeFilter = &truckType
	}

	trucks, err := h.service.ListTrucks(statusFilter, typeFilter)
	if err != nil {
		log.Printf("❌ Error al listar camiones: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convertir a respuesta
	truckResponses := make([]TruckResponse, 0, len(trucks))
	for _, truck := range trucks {
		truckResponses = append(truckResponses, toTruckResponse(&truck))
	}

	c.JSON(http.StatusOK, ListTrucksResponse{
		Trucks: truckResponses,
		Count:  len(truckResponses),
	})
}

// UpdateTruck actualiza un camión existente
// @Summary Actualizar camión
// @Description Actualiza la información de un camión existente
// @Tags Camiones
// @Accept json
// @Produce json
// @Param id path int true "ID del camión"
// @Param request body UpdateTruckRequest true "Datos a actualizar"
// @Success 200 {object} TruckResponse "Camión actualizado exitosamente"
// @Failure 400 {object} ErrorResponse "Datos inválidos o ID inválido"
// @Failure 404 {object} ErrorResponse "Camión no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /trucks/{id} [put]
func (h *TruckHandler) UpdateTruck(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var req UpdateTruckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Construir mapa de actualizaciones
	updates := make(map[string]interface{})
	if req.Plate != nil {
		updates["plate"] = *req.Plate
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se proporcionaron campos para actualizar"})
		return
	}

	truck, err := h.service.UpdateTruck(uint(id), updates)
	if err != nil {
		log.Printf("❌ Error al actualizar camión: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toTruckResponse(truck))
}

// DeleteTruck elimina un camión
// @Summary Eliminar camión
// @Description Elimina un camión de la flota (solo si no tiene turnos activos)
// @Tags Camiones
// @Produce json
// @Param id path int true "ID del camión"
// @Success 200 {object} map[string]string "Camión eliminado exitosamente"
// @Failure 400 {object} ErrorResponse "ID inválido o camión con turnos activos"
// @Failure 404 {object} ErrorResponse "Camión no encontrado"
// @Failure 500 {object} ErrorResponse "Error interno del servidor"
// @Router /trucks/{id} [delete]
func (h *TruckHandler) DeleteTruck(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := h.service.DeleteTruck(uint(id)); err != nil {
		log.Printf("❌ Error al eliminar camión: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Camión eliminado exitosamente"})
}

// toTruckResponse convierte un modelo Truck a TruckResponse
func toTruckResponse(truck *models.Truck) TruckResponse {
	return TruckResponse{
		ID:        truck.ID,
		Plate:     truck.Plate,
		Type:      truck.Type,
		Status:    truck.Status,
		CreatedAt: truck.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: truck.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
