package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	DEPOT_LATITUDE  = -0.9364043
	DEPOT_LONGITUDE = -78.6087099
	DEPOT_ID        = "DEPOT_START"
)

// Orchestrator coordina el flujo de planificación usando patrón Saga
type Orchestrator struct {
	rpcClient     *messaging.RPCClient
	pendingRepo   repository.IPendingItemRepository
	zoneRepo      repository.IZoneRepository
	rabbitChannel *amqp.Channel
}

// NewOrchestrator crea una nueva instancia del orquestador
func NewOrchestrator(
	rpcClient *messaging.RPCClient,
	pendingRepo repository.IPendingItemRepository,
	zoneRepo repository.IZoneRepository,
	rabbitConn *amqp.Connection,
) (*Orchestrator, error) {
	channel, err := rabbitConn.Channel()
	if err != nil {
		return nil, fmt.Errorf("error creando canal RabbitMQ: %w", err)
	}

	// Declarar exchange para operaciones
	err = channel.ExchangeDeclare(
		"city.cleaning.operations", // name
		"topic",                    // type
		true,                       // durable
		false,                      // auto-deleted
		false,                      // internal
		false,                      // no-wait
		nil,                        // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("error declarando exchange: %w", err)
	}

	return &Orchestrator{
		rpcClient:     rpcClient,
		pendingRepo:   pendingRepo,
		zoneRepo:      zoneRepo,
		rabbitChannel: channel,
	}, nil
}

// TriggerZone orquesta la planificación completa para una zona
func (o *Orchestrator) TriggerZone(zoneID int, zoneName string) error {
	log.Printf("🚀 [ORCHESTRATOR] Iniciando planificación para Zona %d (%s)", zoneID, zoneName)

	// ========================================
	// PASO 0: Bloquear incidentes (PROCESSING)
	// ========================================
	incidents, err := o.lockIncidents(zoneID)
	if err != nil {
		return fmt.Errorf("error bloqueando incidentes: %w", err)
	}

	if len(incidents) == 0 {
		log.Printf("⚠️  No hay incidentes pendientes para zona %d", zoneID)
		return nil
	}

	log.Printf("🔒 %d incidentes marcados como PROCESSING", len(incidents))

	// Variables para rollback
	var fleetResponse *models.FleetResponse
	var routingResponse *models.RoutingResponse

	// ========================================
	// PASO 1: Solicitar recursos a Fleet
	// ========================================
	fleetRequest := models.FleetRequest{
		ZoneID:       zoneID,
		RequiredType: "STANDARD", // TODO: determinar dinámicamente según zona
	}

	fleetResponse, err = o.rpcClient.RequestFleet(fleetRequest)
	if err != nil {
		log.Printf("❌ Error en RequestFleet: %v", err)
		o.unlockIncidents(zoneID)
		return fmt.Errorf("fallo al solicitar recursos de flota: %w", err)
	}

	log.Printf("✅ Recursos asignados: Driver=%s, Truck=%s", fleetResponse.DriverID, fleetResponse.TruckPlate)

	// ========================================
	// PASO 2: Solicitar optimización de ruta
	// ========================================
	routingRequest := o.buildRoutingRequest(zoneID, zoneName, incidents)

	routingResponse, err = o.rpcClient.RequestRouting(routingRequest)
	if err != nil {
		log.Printf("❌ Error en RequestRouting: %v", err)

		// ROLLBACK: Liberar recursos de Fleet
		releaseReq := models.FleetReleaseRequest{
			DriverID:   fleetResponse.DriverID,
			TruckPlate: fleetResponse.TruckPlate,
			Reason:     "ROUTING_FAILED",
		}
		if releaseErr := o.rpcClient.ReleaseFleet(releaseReq); releaseErr != nil {
			log.Printf("⚠️  Error en compensación Fleet: %v", releaseErr)
		}

		o.unlockIncidents(zoneID)
		return fmt.Errorf("fallo al optimizar ruta: %w", err)
	}

	log.Printf("✅ Ruta optimizada: Distance=%.2fkm, Duration=%dmin",
		routingResponse.DistanceKm, routingResponse.DurationMin)

	// ========================================
	// PASO 3: Publicar Work Order a Operaciones
	// ========================================
	workOrder := o.buildWorkOrder(zoneID, zoneName, fleetResponse, routingResponse, incidents)

	if err := o.publishWorkOrder(workOrder); err != nil {
		log.Printf("❌ Error publicando Work Order: %v", err)

		// ROLLBACK: Liberar recursos de Fleet
		releaseReq := models.FleetReleaseRequest{
			DriverID:   fleetResponse.DriverID,
			TruckPlate: fleetResponse.TruckPlate,
			Reason:     "WORKORDER_PUBLISH_FAILED",
		}
		o.rpcClient.ReleaseFleet(releaseReq)

		o.unlockIncidents(zoneID)
		return fmt.Errorf("fallo al publicar Work Order: %w", err)
	}

	log.Printf("✅ Work Order publicado: OrderID=%s", workOrder.OrderID)

	// ========================================
	// PASO 4: Resetear Score de la zona
	// ========================================
	if err := o.resetZoneScore(zoneID); err != nil {
		log.Printf("⚠️  Error reseteando Score de zona %d: %v", zoneID, err)
		// No es crítico, continuar
	}

	log.Printf("🎉 [ORCHESTRATOR] Planificación completada exitosamente para Zona %d", zoneID)
	return nil
}

// lockIncidents marca todos los incidentes pendientes de una zona como PROCESSING
func (o *Orchestrator) lockIncidents(zoneID int) ([]models.PendingItem, error) {
	incidents, err := o.pendingRepo.FindByZoneID(zoneID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo incidentes: %w", err)
	}

	// Marcar como PROCESSING
	for i := range incidents {
		incidents[i].Status = "PROCESSING"
		if err := o.pendingRepo.Update(&incidents[i]); err != nil {
			return nil, fmt.Errorf("error actualizando incidente %s: %w", incidents[i].IncidentID, err)
		}
	}

	return incidents, nil
}

// unlockIncidents libera los incidentes marcándolos como PENDING nuevamente
func (o *Orchestrator) unlockIncidents(zoneID int) {
	incidents, err := o.pendingRepo.FindByZoneID(zoneID)
	if err != nil {
		log.Printf("⚠️  Error obteniendo incidentes para desbloquear: %v", err)
		return
	}

	for i := range incidents {
		if incidents[i].Status == "PROCESSING" {
			incidents[i].Status = "PENDING"
			if err := o.pendingRepo.Update(&incidents[i]); err != nil {
				log.Printf("⚠️  Error desbloqueando incidente %s: %v", incidents[i].IncidentID, err)
			}
		}
	}

	log.Printf("🔓 Incidentes de zona %d desbloqueados", zoneID)
}

// buildRoutingRequest construye la solicitud de ruta con DEPOT + incidentes
func (o *Orchestrator) buildRoutingRequest(zoneID int, zoneName string, incidents []models.PendingItem) models.RoutingRequest {
	requestID := fmt.Sprintf("route_request_%d_%d", zoneID, time.Now().UnixNano())

	// Punto 0: DEPOT (punto de inicio obligatorio)
	points := []models.GeoPoint{
		{
			Lat:           DEPOT_LATITUDE,
			Lon:           DEPOT_LONGITUDE,
			IncidentID:    DEPOT_ID,
			GravityPoints: 0,
		},
	}

	// Agregar incidentes
	for _, inc := range incidents {
		if inc.Latitude != nil && inc.Longitude != nil {
			points = append(points, models.GeoPoint{
				Lat:           *inc.Latitude,
				Lon:           *inc.Longitude,
				IncidentID:    inc.IncidentID,
				GravityPoints: inc.GravityPoints,
			})
		}
	}

	return models.RoutingRequest{
		RequestID: requestID,
		ZoneID:    zoneID,
		ZoneName:  zoneName,
		Points:    points,
	}
}

// buildWorkOrder construye el evento de orden de trabajo
func (o *Orchestrator) buildWorkOrder(
	zoneID int,
	zoneName string,
	fleet *models.FleetResponse,
	routing *models.RoutingResponse,
	incidents []models.PendingItem,
) models.WorkOrderEvent {
	incidentIDs := make([]string, len(incidents))
	for i, inc := range incidents {
		incidentIDs[i] = inc.IncidentID
	}

	return models.WorkOrderEvent{
		OrderID:     uuid.New(),
		ZoneID:      zoneID,
		ZoneName:    zoneName,
		DriverID:    fleet.DriverID,
		AssistantID: fleet.AssistantID,
		TruckPlate:  fleet.TruckPlate,
		RouteID:     routing.RequestID,
		Polyline:    routing.Polyline,
		DistanceKm:  routing.DistanceKm,
		DurationMin: routing.DurationMin,
		IncidentIDs: incidentIDs,
		Status:      "PENDING",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
}

// publishWorkOrder publica el evento de Work Order a Operaciones
func (o *Orchestrator) publishWorkOrder(order models.WorkOrderEvent) error {
	body, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("error serializando WorkOrder: %w", err)
	}

	err = o.rabbitChannel.Publish(
		"city.cleaning.operations", // exchange
		"workorder.created",        // routing key
		false,                      // mandatory
		false,                      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("error publicando WorkOrder: %w", err)
	}

	return nil
}

// resetZoneScore resetea el Score de la zona a 0
func (o *Orchestrator) resetZoneScore(zoneID int) error {
	zone, err := o.zoneRepo.FindByID(zoneID)
	if err != nil {
		return fmt.Errorf("error obteniendo zona: %w", err)
	}

	zone.Score = 0
	if err := o.zoneRepo.Update(zone); err != nil {
		return fmt.Errorf("error actualizando zona: %w", err)
	}

	log.Printf("🔄 Score de zona %d reseteado a 0", zoneID)
	return nil
}
