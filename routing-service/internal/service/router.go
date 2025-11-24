package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/osrm"
	"gorm.io/gorm"
)

// RouteService maneja la lógica de negocio de rutas
type RouteService struct {
	db         *gorm.DB
	osrmClient *osrm.OSRMClient
}

// NewRouteService crea una nueva instancia del servicio
func NewRouteService(db *gorm.DB, osrmClient *osrm.OSRMClient) *RouteService {
	return &RouteService{
		db:         db,
		osrmClient: osrmClient,
	}
}

// OptimizeAndSave optimiza una ruta usando OSRM y guarda el resultado en la BD
func (s *RouteService) OptimizeAndSave(req models.RouteRequest) (*models.RouteResponse, error) {
	log.Printf("[INFO] Optimizando ruta para %d puntos (RequestID: %s, ZoneID: %d)...",
		len(req.Points), req.RequestID, req.ZoneID)

	// Validación: si solo hay 1 punto, agregar DEPOT automáticamente (retrocompatibilidad)
	points := req.Points
	if len(points) == 1 {
		log.Printf("⚠️  Solo 1 punto recibido, agregando DEPOT (EPAGAL) como punto de inicio...")
		depotPoint := models.Point{
			Latitude:  -0.9364043,
			Longitude: -78.6087099,
		}
		// Crear nuevo slice con DEPOT primero
		points = append([]models.Point{depotPoint}, points...)
		log.Printf("✅ DEPOT agregado. Total puntos: %d", len(points))
	}

	// 1. Llamar a OSRM para optimizar
	osrmResp, err := s.osrmClient.OptimizeRoute(points)
	if err != nil {
		return nil, fmt.Errorf("error al optimizar con OSRM: %w", err)
	}

	trip := osrmResp.Trips[0]

	// 2. Extraer el orden de los waypoints
	waypointOrder := osrm.ExtractWaypointOrder(osrmResp)
	waypointOrderJSON, err := json.Marshal(waypointOrder)
	if err != nil {
		return nil, fmt.Errorf("error al serializar waypoint_order: %w", err)
	}

	// Log del orden optimizado de los waypoints
	log.Printf("📊 [RUTA OPTIMIZADA] Zona %d - Orden de visita: %v", req.ZoneID, waypointOrder)
	log.Printf("   📏 Distancia total: %.2f metros (%.2f km)", trip.Distance, trip.Distance/1000)
	log.Printf("   ⏱️  Duración estimada: %.2f segundos (%.1f minutos)", trip.Duration, trip.Duration/60)

	// Log detallado de los puntos en el orden optimizado
	log.Printf("   🗺️  Secuencia de puntos optimizada:")
	for i, waypointIdx := range waypointOrder {
		if waypointIdx < len(points) {
			point := points[waypointIdx]
			log.Printf("      %d. Waypoint[%d] - Incident: %s (%.6f, %.6f)",
				i+1, waypointIdx, point.IncidentID, point.Latitude, point.Longitude)
		}
	}

	// 3. Crear el registro en la base de datos
	routePlan := models.RoutePlan{
		RequestID:     req.RequestID,
		ZoneID:        req.ZoneID,
		Distance:      trip.Distance,
		Duration:      trip.Duration,
		Geometry:      trip.Geometry,
		WaypointOrder: string(waypointOrderJSON),
	}

	if err := s.db.Create(&routePlan).Error; err != nil {
		return nil, fmt.Errorf("error al guardar en la base de datos: %w", err)
	}

	log.Printf("✅ [SUCCESS] Ruta guardada en BD: ID=%s", routePlan.ID)

	// 4. Construir la respuesta para RabbitMQ
	response := &models.RouteResponse{
		RequestID:     req.RequestID,
		ZoneID:        req.ZoneID,
		Distance:      trip.Distance,
		Duration:      trip.Duration,
		Geometry:      trip.Geometry,
		WaypointOrder: waypointOrder,
		OptimizedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	return response, nil
}

// GetRoutePlanByRequestID recupera un plan de ruta por su RequestID
func (s *RouteService) GetRoutePlanByRequestID(requestID string) (*models.RoutePlan, error) {
	var plan models.RoutePlan
	err := s.db.Where("request_id = ?", requestID).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetRoutePlansByZoneID recupera todos los planes de una zona
func (s *RouteService) GetRoutePlansByZoneID(zoneID uint) ([]models.RoutePlan, error) {
	var plans []models.RoutePlan
	err := s.db.Where("zone_id = ?", zoneID).Order("created_at DESC").Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// GetLatestRouteByZone recupera la ruta más reciente de una zona
func (s *RouteService) GetLatestRouteByZone(zoneID uint) (*models.RoutePlan, error) {
	var plan models.RoutePlan
	err := s.db.Where("zone_id = ?", zoneID).Order("created_at DESC").First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetAllRoutes recupera todas las rutas optimizadas
func (s *RouteService) GetAllRoutes() ([]models.RoutePlan, error) {
	var plans []models.RoutePlan
	err := s.db.Order("created_at DESC").Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// DecodePolyline decodifica una geometría polyline a coordenadas lat/lon
func (s *RouteService) DecodePolyline(encoded string) ([]map[string]float64, error) {
	coordinates := make([]map[string]float64, 0)

	index := 0
	lat := 0
	lon := 0

	for index < len(encoded) {
		var result int
		var shift uint
		var b int

		// Decodificar latitud
		for {
			b = int(encoded[index]) - 63
			index++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}

		var dlat int
		if (result & 1) != 0 {
			dlat = ^(result >> 1)
		} else {
			dlat = result >> 1
		}
		lat += dlat

		// Decodificar longitud
		result = 0
		shift = 0
		for {
			b = int(encoded[index]) - 63
			index++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}

		var dlon int
		if (result & 1) != 0 {
			dlon = ^(result >> 1)
		} else {
			dlon = result >> 1
		}
		lon += dlon

		coordinates = append(coordinates, map[string]float64{
			"latitude":  float64(lat) / 1e5,
			"longitude": float64(lon) / 1e5,
		})
	}

	return coordinates, nil
}
