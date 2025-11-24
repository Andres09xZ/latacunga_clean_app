package services

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/schedule"
)

// Incident weights (simple map; can be externalized later)
var incidentWeights = map[string]int{
	// Tipos reales del sistema (lowercase para coincidir con incident-service)
	"zona_critica":   8,
	"animal_muerto":  6,
	"punto_acopio":   10,
	"zona_reciclaje": 3,
	// Legacy types
	"SENSOR_LLENO":   5,
	"REPORTE_VECINO": 3,
	"DESBORDE":       10,
}

// PlanningService aplica la lógica de acumulación y triggers.
type PlanningService struct {
	repo          repository.IZoneRepository
	pendingRepo   repository.IPendingItemRepository
	schedulerRepo repository.ISchedulerRepository
	publisher     *messaging.PlanningPublisher
	triggerLogic  interface {
		EvaluateAndTrigger(zoneID int, newScore int, threshold int, zoneName string) error
	}
}

func NewPlanningService(repo repository.IZoneRepository, pendingRepo repository.IPendingItemRepository) *PlanningService {
	return &PlanningService{
		repo:        repo,
		pendingRepo: pendingRepo,
		publisher:   messaging.InitPlanningMessaging(),
	}
}

// NewPlanningServiceWithPublisher permite inyectar un publisher (mock en pruebas).
func NewPlanningServiceWithPublisher(repo repository.IZoneRepository, pendingRepo repository.IPendingItemRepository, pub *messaging.PlanningPublisher) *PlanningService {
	return &PlanningService{repo: repo, pendingRepo: pendingRepo, publisher: pub}
}

// SetTriggerLogic inyecta la lógica de trigger (para desacople)
func (s *PlanningService) SetTriggerLogic(tl interface {
	EvaluateAndTrigger(zoneID int, newScore int, threshold int, zoneName string) error
}) {
	s.triggerLogic = tl
}

// SetSchedulerRepo inyecta el repositorio de scheduler
func (s *PlanningService) SetSchedulerRepo(repo repository.ISchedulerRepository) {
	s.schedulerRepo = repo
}

// ProcessIncident localiza la zona y acumula puntaje.
func (s *PlanningService) ProcessIncident(lat, lon float64, incidentType string) (*models.PlanningResult, error) {
	// Buscar peso del incidente (lowercase)
	weight := incidentWeights[strings.ToLower(incidentType)]
	if weight == 0 {
		return nil, errors.New("tipo de incidente desconocido: " + incidentType + ". Tipos válidos: zona_critica, animal_muerto, punto_acopio, zona_reciclaje")
	}

	// Si las coordenadas son 0,0 o inválidas, usar zona por defecto (ID 1)
	var zone *models.CleaningZone
	var err error

	if lat == 0 && lon == 0 {
		log.Println("⚠️ Coordenadas 0,0 detectadas, usando zona por defecto (ID: 1)")
		zones, err := s.repo.ListZones()
		if err != nil || len(zones) == 0 {
			return nil, errors.New("no hay zonas disponibles en el sistema")
		}
		zone = &zones[0] // Usar primera zona por defecto
	} else {
		zone, err = s.repo.FindZoneByPoint(lat, lon)
		if err != nil {
			return nil, errors.New("zona no encontrada para las coordenadas proporcionadas (lat: " + fmt.Sprintf("%.6f", lat) + ", lon: " + fmt.Sprintf("%.6f", lon) + ")")
		}
	}

	// Generar ID único para el incidente (timestamp + tipo)
	incidentID := fmt.Sprintf("%s_%d_%s", incidentType, time.Now().UnixNano(), zone.ZoneName)

	// Guardar incidente pendiente en la base de datos
	latPtr := &lat
	lonPtr := &lon
	pendingItem := &models.PendingItem{
		Latitude:      latPtr,
		Longitude:     lonPtr,
		IncidentID:    incidentID,
		ZoneID:        zone.ID,
		GravityPoints: weight,
		Status:        "PENDING",
	}
	if err := s.pendingRepo.Create(pendingItem); err != nil {
		return nil, fmt.Errorf("error guardando incidente pendiente: %w", err)
	}

	metrics, triggered, err := s.repo.(*repository.ZoneRepository).IncrementScore(zone.ID, weight)
	if err != nil {
		return nil, err
	}

	nextTime, label := schedule.ParseSchedule(zone.ScheduleConfig, time.Now())
	status := zone.Status
	if triggered {
		status = "LISTO_PARA_RECOLECCION"
		_ = s.repo.UpdateStatus(zone.ID, status)

		// Obtener todos los incidentes PENDING de esta zona
		pendingItems, err := s.pendingRepo.FindByZone(zone.ID, "PENDING")
		if err != nil {
			log.Printf("⚠️ Error obteniendo incidentes pendientes: %v", err)
		}

		// Solicitar ruta optimizada al routing-service
		if len(pendingItems) >= 2 {
			log.Printf("🗺️  Solicitando ruta optimizada para zona %s con %d incidentes", zone.ZoneName, len(pendingItems))
			if err := s.requestOptimizedRoute(zone.ID, pendingItems); err != nil {
				log.Printf("⚠️ Error solicitando ruta optimizada: %v", err)
			}
		} else {
			log.Printf("⚠️ Solo hay %d incidente(s) en zona %s, se requieren al menos 2 para optimizar ruta", len(pendingItems), zone.ZoneName)
		}

		// NUEVO: Usar TriggerLogic para orquestación completa (Saga RPC)
		if s.triggerLogic != nil {
			if err := s.triggerLogic.EvaluateAndTrigger(int(zone.ID), metrics.CurrentScore, metrics.Threshold, zone.ZoneName); err != nil {
				// Log el error pero no fallar el proceso
				log.Printf("⚠️ Error en trigger logic (orquestación): %v", err)
				// Si falla la orquestación, los incidentes se mantienen PENDING automáticamente
			} else {
				// Si la orquestación fue exitosa, el score ya fue reseteado por el orchestrator
				log.Printf("✅ Orquestación exitosa para zona %d (%s)", zone.ID, zone.ZoneName)
				metrics.CurrentScore = 0 // Actualizar métricas en respuesta
			}
		} else {
			// Fallback: lógica legacy (si no hay orchestrator)
			log.Printf("⚠️  No hay orchestrator configurado, usando lógica legacy")

			// Marcar todos los incidentes pendientes como PROCESSING
			for _, item := range pendingItems {
				_ = s.pendingRepo.UpdateStatus(item.IncidentID, "PROCESSING")
			}

			// Emit event notifying resources needed due to threshold exceeded
			if s.publisher != nil {
				_ = s.publisher.PublishResourceRequested(zone.ZoneName, nextTime, "threshold_exceeded")
			}

			// RESETEAR el score de la zona a 0 después del trigger
			if err := s.repo.ResetScore(zone.ID); err != nil {
				log.Printf("⚠️ Error reseteando score de zona %d: %v", zone.ID, err)
			} else {
				log.Printf("🔄 Score de zona %d (%s) reseteado a 0", zone.ID, zone.ZoneName)
				// Actualizar las métricas en la respuesta
				metrics.CurrentScore = 0
			}
		}
	}

	return &models.PlanningResult{
		ZoneID:         zone.ID,
		ZoneName:       zone.ZoneName,
		NewScore:       metrics.CurrentScore,
		Threshold:      metrics.Threshold,
		Triggered:      triggered,
		Status:         status,
		ScheduledTime:  nextTime,
		ScheduledLabel: label,
	}, nil
}

// ForceTrigger ejecuta trigger manual ignorando puntaje.
func (s *PlanningService) ForceTrigger(zoneID uint, reason string) (*models.PlanningResult, error) {
	// Obtener zona
	var zone *models.CleaningZone
	zones, err := s.repo.ListZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].ID == zoneID {
			zone = &zones[i]
			break
		}
	}
	if zone == nil {
		return nil, errors.New("zona no encontrada")
	}
	metrics, err := s.repo.(*repository.ZoneRepository).ForceTrigger(zoneID, reason)
	if err != nil {
		return nil, err
	}
	_ = s.repo.UpdateStatus(zoneID, "LISTO_PARA_RECOLECCION")
	nextTime, label := schedule.ParseSchedule(zone.ScheduleConfig, time.Now())
	if s.publisher != nil {
		_ = s.publisher.PublishResourceRequested(zone.ZoneName, nextTime, reason)
	}
	return &models.PlanningResult{
		ZoneID: zone.ID, ZoneName: zone.ZoneName,
		NewScore: metrics.CurrentScore, Threshold: metrics.Threshold,
		Triggered: true, Status: "LISTO_PARA_RECOLECCION",
		ScheduledTime: nextTime, ScheduledLabel: label, Reason: reason,
	}, nil
}

// requestOptimizedRoute solicita una ruta optimizada al routing-service via RabbitMQ
func (s *PlanningService) requestOptimizedRoute(zoneID uint, pendingItems []models.PendingItem) error {
	if s.publisher == nil {
		return errors.New("publisher no inicializado")
	}

	// Generar request ID único
	requestID := fmt.Sprintf("route_request_%d_%d", zoneID, time.Now().UnixNano())

	// Extraer coordenadas de los incidentes
	type Point struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	// IMPORTANTE: Insertar EPAGAL como primer punto
	points := make([]Point, 0, len(pendingItems)+1)

	// Agregar DEPOT (EPAGAL) como primer punto
	points = append(points, Point{
		Latitude:  -0.9364043,
		Longitude: -78.6087099,
	})
	log.Printf("🏢 DEPOT agregado a la ruta (EPAGAL)")

	// Agregar los puntos de incidentes
	for _, item := range pendingItems {
		if item.Latitude != nil && item.Longitude != nil {
			points = append(points, Point{
				Latitude:  *item.Latitude,
				Longitude: *item.Longitude,
			})
		}
	}

	// Crear mensaje de solicitud de ruta
	routeRequest := map[string]interface{}{
		"request_id": requestID,
		"zone_id":    zoneID,
		"points":     points,
	}

	// Publicar a RabbitMQ
	if err := s.publisher.PublishRouteRequest(routeRequest); err != nil {
		return fmt.Errorf("error publicando solicitud de ruta: %w", err)
	}

	log.Printf("✅ Solicitud de ruta publicada: RequestID=%s ZoneID=%d Points=%d", requestID, zoneID, len(points))
	return nil
}
