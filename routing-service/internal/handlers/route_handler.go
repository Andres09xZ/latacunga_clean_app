package handlers

import (
	"net/http"
	"strconv"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/service"
	"github.com/gin-gonic/gin"
)

type RouteHandler struct {
	routeService *service.RouteService
}

func NewRouteHandler(routeService *service.RouteService) *RouteHandler {
	return &RouteHandler{
		routeService: routeService,
	}
}

// GetOptimizedRouteByZone godoc
// @Summary Obtiene la ruta optimizada por zona
// @Description Retorna la última ruta optimizada para una zona específica con las coordenadas de los puntos
// @Tags Routes
// @Produce json
// @Param zone_id path int true "ID de la zona"
// @Success 200 {object} map[string]interface{} "Ruta optimizada con coordenadas"
// @Failure 400 {object} map[string]string "ID de zona inválido"
// @Failure 404 {object} map[string]string "No se encontró ruta para esta zona"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/routes/zone/{zone_id} [get]
func (h *RouteHandler) GetOptimizedRouteByZone(c *gin.Context) {
	zoneIDStr := c.Param("zone_id")
	zoneID, err := strconv.ParseUint(zoneIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de zona inválido"})
		return
	}

	route, err := h.routeService.GetLatestRouteByZone(uint(zoneID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró ruta optimizada para esta zona"})
		return
	}

	// Decodificar la geometría polyline a coordenadas
	coordinates, err := h.routeService.DecodePolyline(route.Geometry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decodificando geometría"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"zone_id":     zoneID,
		"request_id":  route.RequestID,
		"distance_m":  route.Distance,
		"duration_s":  route.Duration,
		"coordinates": coordinates,
		"created_at":  route.CreatedAt,
	})
}

// GetAllOptimizedRoutes godoc
// @Summary Lista todas las rutas optimizadas
// @Description Retorna todas las rutas optimizadas agrupadas por zona
// @Tags Routes
// @Produce json
// @Success 200 {object} map[string]interface{} "Rutas optimizadas agrupadas por zona"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/routes [get]
func (h *RouteHandler) GetAllOptimizedRoutes(c *gin.Context) {
	routes, err := h.routeService.GetAllRoutes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo rutas"})
		return
	}

	// Agrupar por zona
	routesByZone := make(map[uint]interface{})
	for _, route := range routes {
		coordinates, err := h.routeService.DecodePolyline(route.Geometry)
		if err != nil {
			continue // Skip rutas con errores
		}

		routesByZone[route.ZoneID] = gin.H{
			"zone_id":     route.ZoneID,
			"request_id":  route.RequestID,
			"distance_m":  route.Distance,
			"duration_s":  route.Duration,
			"coordinates": coordinates,
			"created_at":  route.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_zones": len(routesByZone),
		"routes":      routesByZone,
	})
}

// HealthCheck godoc
// @Summary Health check del servicio
// @Description Verifica el estado del servicio de routing
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *RouteHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "routing-service",
	})
}
