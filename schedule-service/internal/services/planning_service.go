package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/schedule"
)

// Incident weights (simple map; can be externalized later)
var incidentWeights = map[string]int{
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
	zone, err := s.repo.FindZoneByPoint(lat, lon)
	if err != nil {
		return nil, errors.New("zona no encontrada para las coordenadas proporcionadas (lat: " + fmt.Sprintf("%.6f", lat) + ", lon: " + fmt.Sprintf("%.6f", lon) + ")")
	}
	weight := incidentWeights[strings.ToUpper(incidentType)]
	if weight == 0 {
		return nil, errors.New("tipo de incidente desconocido: " + incidentType + ". Tipos válidos: SENSOR_LLENO, REPORTE_CIUDADANO, DESBORDAMIENTO")
	}

	// Generar ID único para el incidente (timestamp + tipo)
	incidentID := fmt.Sprintf("%s_%d_%s", incidentType, time.Now().UnixNano(), zone.ZoneName)

	// Guardar incidente pendiente en la base de datos
	pendingItem := &models.PendingItem{
		Lat:           lat,
		Lon:           lon,
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

		// NUEVO: Usar TriggerLogic para publicar coordenadas a RabbitMQ
		if s.triggerLogic != nil {
			if err := s.triggerLogic.EvaluateAndTrigger(int(zone.ID), metrics.CurrentScore, metrics.Threshold, zone.ZoneName); err != nil {
				// Log el error pero no fallar el proceso
				fmt.Printf("⚠️ Error en trigger logic: %v\n", err)
			}
		} else {
			// Fallback al comportamiento anterior si no hay TriggerLogic
			// Marcar todos los incidentes pendientes como PROCESSED
			pendingItems, _ := s.pendingRepo.FindByZone(zone.ID, "PENDING")
			for _, item := range pendingItems {
				_ = s.pendingRepo.UpdateStatus(item.IncidentID, "PROCESSED")
			}

			// Emit event notifying resources needed due to threshold exceeded
			if s.publisher != nil {
				_ = s.publisher.PublishResourceRequested(zone.ZoneName, nextTime, "threshold_exceeded")
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
