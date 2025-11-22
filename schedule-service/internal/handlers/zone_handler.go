package handlers

import (
	"net/http"
	"strconv"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ZoneHandler maneja las solicitudes HTTP relacionadas con zonas
type ZoneHandler struct {
	zoneRepo *repository.ZoneRepository
}

// NewZoneHandler crea una nueva instancia del handler
func NewZoneHandler(db *gorm.DB) *ZoneHandler {
	return &ZoneHandler{
		zoneRepo: repository.NewZoneRepository(db),
	}
}

// GetAllZones godoc
// @Summary Lista todas las zonas de recolección
// @Description Retorna todas las zonas geográficas de recolección
// @Tags zones
// @Accept json
// @Produce json
// @Success 200 {array} models.CleaningZone
// @Failure 500 {object} map[string]string
// @Router /api/zones [get]
func (h *ZoneHandler) GetAllZones(c *gin.Context) {
	zones, err := h.zoneRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener zonas"})
		return
	}

	c.JSON(http.StatusOK, zones)
}

// GetZoneByID godoc
// @Summary Obtiene una zona por ID
// @Description Retorna los detalles de una zona específica
// @Tags zones
// @Accept json
// @Produce json
// @Param id path int true "Zone ID"
// @Success 200 {object} models.CleaningZone
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/zones/{id} [get]
func (h *ZoneHandler) GetZoneByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	zone, err := h.zoneRepo.FindByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Zona no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener zona"})
		return
	}

	c.JSON(http.StatusOK, zone)
}

// GetZonesByRoute godoc
// @Summary Lista zonas por ruta
// @Description Retorna todas las zonas de una ruta específica
// @Tags zones
// @Accept json
// @Produce json
// @Param route_name path string true "Route Name" example(RUTA_1)
// @Success 200 {array} models.CleaningZone
// @Failure 500 {object} map[string]string
// @Router /api/zones/route/{route_name} [get]
func (h *ZoneHandler) GetZonesByRoute(c *gin.Context) {
	routeName := c.Param("route_name")

	zones, err := h.zoneRepo.FindByRouteName(routeName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener zonas"})
		return
	}

	c.JSON(http.StatusOK, zones)
}

// GetZonesByDay godoc
// @Summary Lista zonas por día
// @Description Retorna todas las zonas de un día específico (0=Dom, 1=Lun, ..., 6=Sáb)
// @Tags zones
// @Accept json
// @Produce json
// @Param day path int true "Day of Week" example(1)
// @Success 200 {array} models.CleaningZone
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/zones/day/{day} [get]
func (h *ZoneHandler) GetZonesByDay(c *gin.Context) {
	day, err := strconv.Atoi(c.Param("day"))
	if err != nil || day < 0 || day > 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Día inválido (debe ser 0-6)"})
		return
	}

	zones, err := h.zoneRepo.FindByScheduleDay(day)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener zonas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"day":      day,
		"day_name": models.GetDayName(day),
		"zones":    zones,
	})
}

// SearchZoneByPoint godoc
// @Summary Busca zona por coordenadas
// @Description Encuentra la zona que contiene un punto geográfico específico
// @Tags zones
// @Accept json
// @Produce json
// @Param lat query number true "Latitude" example(-0.933)
// @Param lon query number true "Longitude" example(-78.614)
// @Success 200 {object} models.ZoneSearchResult
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/zones/search [get]
func (h *ZoneHandler) SearchZoneByPoint(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	if latStr == "" || lonStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Se requieren parámetros 'lat' y 'lon'"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Latitud inválida"})
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Longitud inválida"})
		return
	}

	result, err := h.zoneRepo.FindZoneByPoint(lat, lon)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No se encontró ninguna zona para estas coordenadas",
				"lat":   lat,
				"lon":   lon,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en búsqueda espacial"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetNearestZone godoc
// @Summary Busca zona más cercana
// @Description Encuentra la zona más cercana a un punto geográfico (aunque no lo contenga)
// @Tags zones
// @Accept json
// @Produce json
// @Param lat query number true "Latitude" example(-0.933)
// @Param lon query number true "Longitude" example(-78.614)
// @Success 200 {object} models.ZoneSearchResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/zones/nearest [get]
func (h *ZoneHandler) GetNearestZone(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	if latStr == "" || lonStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Se requieren parámetros 'lat' y 'lon'"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Latitud inválida"})
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Longitud inválida"})
		return
	}

	result, err := h.zoneRepo.FindNearestZone(lat, lon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en búsqueda espacial"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetZonesSummaryByRoute godoc
// @Summary Resumen de zonas por ruta
// @Description Retorna estadísticas agrupadas por ruta
// @Tags zones
// @Accept json
// @Produce json
// @Success 200 {array} models.ZonesByRoute
// @Failure 500 {object} map[string]string
// @Router /api/zones/summary/routes [get]
func (h *ZoneHandler) GetZonesSummaryByRoute(c *gin.Context) {
	results, err := h.zoneRepo.GetZonesByRoute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener resumen"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetZonesSummaryByDay godoc
// @Summary Resumen de zonas por día
// @Description Retorna estadísticas agrupadas por día de la semana
// @Tags zones
// @Accept json
// @Produce json
// @Success 200 {array} models.ZonesByDay
// @Failure 500 {object} map[string]string
// @Router /api/zones/summary/days [get]
func (h *ZoneHandler) GetZonesSummaryByDay(c *gin.Context) {
	results, err := h.zoneRepo.GetZonesByDay()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener resumen"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetZonesGeoJSON godoc
// @Summary Exporta zonas como GeoJSON
// @Description Retorna todas las zonas en formato GeoJSON FeatureCollection
// @Tags zones
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/zones/geojson [get]
func (h *ZoneHandler) GetZonesGeoJSON(c *gin.Context) {
	geojson, err := h.zoneRepo.GetAllZonesAsGeoJSON()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar GeoJSON"})
		return
	}

	// Retornar como JSON raw
	c.Header("Content-Type", "application/geo+json")
	c.String(http.StatusOK, geojson)
}
