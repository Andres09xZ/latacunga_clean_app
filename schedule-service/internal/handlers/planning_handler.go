package handlers

import (
	"net/http"
	"strconv"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/services"
	"github.com/gin-gonic/gin"
)

type PlanningHandler struct {
	repo        *repository.ZoneRepository
	pendingRepo *repository.PendingItemRepository
	service     *services.PlanningService
}

func NewPlanningHandler(repo *repository.ZoneRepository, pendingRepo *repository.PendingItemRepository) *PlanningHandler {
	return &PlanningHandler{
		repo:        repo,
		pendingRepo: pendingRepo,
		service:     services.NewPlanningService(repo, pendingRepo),
	}
}

// GetService retorna el servicio de planificación para inyección de dependencias
func (h *PlanningHandler) GetService() *services.PlanningService {
	return h.service
}

// ListZones godoc
// @Summary Lista todas las zonas de recolección
// @Description Obtiene todas las zonas macro de planificación con su geometría y configuración de horario
// @Tags Planning
// @Produce json
// @Success 200 {array} object "Lista de zonas"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/planning/zones [get]
func (h *PlanningHandler) ListZones(c *gin.Context) {
	zones, err := h.repo.ListZones()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, zones)
}

// ZoneMetrics godoc
// @Summary Obtiene métricas de acumulación de una zona
// @Description Retorna el puntaje actual y umbral configurado para una zona específica
// @Tags Planning
// @Produce json
// @Param id path int true "ID de la zona"
// @Success 200 {object} map[string]interface{} "Métricas de la zona"
// @Failure 404 {object} map[string]string "Zona o métricas no encontradas"
// @Router /api/v1/planning/zones/{id}/metrics [get]
func (h *PlanningHandler) ZoneMetrics(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	m, err := h.repo.GetMetrics(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zona o métricas no encontradas"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"zone_id": m.ZoneID, "score": m.CurrentScore, "threshold": m.Threshold})
}

// ForceTrigger godoc
// @Summary Dispara manualmente la recolección para una zona
// @Description Fuerza un trigger de recolección ignorando el puntaje acumulado, útil para emergencias
// @Tags Planning
// @Accept json
// @Produce json
// @Param id path int true "ID de la zona"
// @Param body body object true "Razón del trigger manual (ej: {\"reason\": \"emergencia\"})"
// @Success 201 {object} object "Resultado del trigger"
// @Failure 400 {object} map[string]string "JSON inválido"
// @Failure 500 {object} map[string]string "Error interno"
// @Router /api/v1/planning/zones/{id}/trigger [post]
func (h *PlanningHandler) ForceTrigger(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}
	res, err := h.service.ForceTrigger(uint(id), body.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

// UpdateThresholds godoc
// @Summary Actualiza umbrales de disparo por zona
// @Description Permite configurar dinámicamente los umbrales que activan la recolección automática
// @Tags Planning
// @Accept json
// @Produce json
// @Param body body map[string]int true "Mapa de zona a umbral (ej: {\"URBANO_CENTRAL\": 60})"
// @Success 200 {object} map[string]int "Cantidad de umbrales actualizados"
// @Failure 400 {object} map[string]string "JSON inválido o umbral negativo"
// @Router /api/v1/planning/config/thresholds [put]
func (h *PlanningHandler) UpdateThresholds(c *gin.Context) {
	var payload map[string]int
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}
	if err := h.repo.UpdateThresholds(payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": len(payload)})
}

// SimulateIncident godoc
// @Summary Simula un incidente para debugging
// @Description Inyecta un incidente falso con coordenadas para verificar detección de zona y acumulación de puntaje
// @Tags Planning
// @Accept json
// @Produce json
// @Param body body models.SimulateIncidentRequest true "Datos del incidente simulado"
// @Success 202 {object} models.PlanningResult "Incidente procesado, zona detectada, puntaje acumulado (umbral no superado)"
// @Success 201 {object} models.PlanningResult "Umbral superado, recolección activada automáticamente"
// @Failure 400 {object} map[string]string "Coordenadas inválidas o zona no encontrada"
// @Router /api/v1/planning/simulate [post]
func (h *PlanningHandler) SimulateIncident(c *gin.Context) {
	var body models.SimulateIncidentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}
	res, err := h.service.ProcessIncident(body.Lat, body.Lon, body.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := http.StatusAccepted
	if res.Triggered {
		status = http.StatusCreated
	}
	c.JSON(status, res)
}
