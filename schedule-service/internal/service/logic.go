package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// GeoPoint representa un punto geográfico con información del incidente.
type GeoPoint struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	IncidentID    string  `json:"incident_id"`
	GravityPoints int     `json:"gravity_points"`
}

// RoutingRequest es el payload que se envía a RabbitMQ para solicitar planificación de ruta.
type RoutingRequest struct {
	RequestID string     `json:"request_id"`
	ZoneID    int        `json:"zone_id"`
	ZoneName  string     `json:"zone_name"`
	Points    []GeoPoint `json:"points"`
	Timestamp time.Time  `json:"timestamp"`
}

// TriggerLogic maneja la lógica de disparo cuando se alcanza el umbral.
type TriggerLogic struct {
	schedulerRepo repository.ISchedulerRepository
	pendingRepo   repository.IPendingItemRepository
	zoneRepo      repository.IZoneRepository
	rabbitConn    *amqp.Connection
	rabbitChannel *amqp.Channel
}

// NewTriggerLogic crea una nueva instancia de la lógica de disparo.
func NewTriggerLogic(
	schedulerRepo repository.ISchedulerRepository,
	pendingRepo repository.IPendingItemRepository,
	zoneRepo repository.IZoneRepository,
	rabbitConn *amqp.Connection,
) (*TriggerLogic, error) {
	channel, err := rabbitConn.Channel()
	if err != nil {
		return nil, fmt.Errorf("error creando canal RabbitMQ: %w", err)
	}

	// Declarar exchange para planificación
	err = channel.ExchangeDeclare(
		"city.cleaning.planning", // name
		"topic",                  // type
		true,                     // durable
		false,                    // auto-deleted
		false,                    // internal
		false,                    // no-wait
		nil,                      // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("error declarando exchange: %w", err)
	}

	return &TriggerLogic{
		schedulerRepo: schedulerRepo,
		pendingRepo:   pendingRepo,
		zoneRepo:      zoneRepo,
		rabbitConn:    rabbitConn,
		rabbitChannel: channel,
	}, nil
}

// EvaluateAndTrigger evalúa si el nuevo puntaje alcanzó el umbral y dispara la planificación.
func (tl *TriggerLogic) EvaluateAndTrigger(zoneID int, newScore int, threshold int, zoneName string) error {
	// Verificar si se alcanzó el umbral
	if newScore < threshold {
		log.Printf("⏳ Zona %d (%s): Puntaje %d no alcanza umbral %d", zoneID, zoneName, newScore, threshold)
		return nil
	}

	log.Printf("🔔 TRIGGER ACTIVADO - Zona %d (%s): Puntaje %d >= Umbral %d", zoneID, zoneName, newScore, threshold)

	// 1. Llamar a repo.GetPendingPointsByZone(zoneID) para "cosechar" puntos
	pendingItems, err := tl.schedulerRepo.GetPendingPointsByZone(zoneID)
	if err != nil {
		return fmt.Errorf("error obteniendo puntos pendientes: %w", err)
	}

	if len(pendingItems) == 0 {
		log.Printf("⚠️ No hay puntos pendientes para zona %d", zoneID)
		return nil
	}

	log.Printf("📦 Cosechados %d puntos pendientes de zona %d", len(pendingItems), zoneID)

	// 2. Construir payload con los puntos
	// IMPORTANTE: Insertar punto de partida fijo (EPAGAL) al inicio
	points := make([]GeoPoint, 0, len(pendingItems)+1)

	// Insertar EPAGAL como primer punto (DEPOT_START)
	depotPoint := GeoPoint{
		Lat:           -0.9364043,
		Lon:           -78.6087099,
		IncidentID:    "DEPOT_START",
		GravityPoints: 0, // El depósito no tiene puntaje
	}
	points = append(points, depotPoint)
	log.Printf("🏢 Punto de partida agregado: EPAGAL (Lat: %.7f, Lon: %.7f)", depotPoint.Lat, depotPoint.Lon)

	// Agregar los puntos de incidentes
	for _, item := range pendingItems {
		points = append(points, GeoPoint{
			Lat:           item.Lat,
			Lon:           item.Lon,
			IncidentID:    item.IncidentID,
			GravityPoints: item.GravityPoints,
		})
	}

	requestID := fmt.Sprintf("routing-req-%s", uuid.New().String())
	routingRequest := RoutingRequest{
		RequestID: requestID,
		ZoneID:    zoneID,
		ZoneName:  zoneName,
		Points:    points,
		Timestamp: time.Now(),
	}

	// 3. Publicar a RabbitMQ
	err = tl.publishRoutingRequest(routingRequest)
	if err != nil {
		return fmt.Errorf("error publicando solicitud de planificación: %w", err)
	}

	log.Printf("✅ Publicado request %s a RabbitMQ con %d puntos (1 DEPOT + %d incidentes)", requestID, len(points), len(pendingItems))

	// 4. Actualizar estado de items a 'PROCESSING'
	err = tl.updatePendingItemsStatus(zoneID, "PROCESSING")
	if err != nil {
		return fmt.Errorf("error actualizando estado de items: %w", err)
	}

	log.Printf("✅ Actualizados %d items a estado PROCESSING", len(pendingItems))

	// 5. RESETEAR el score de la zona a 0 después de emitir el mensaje
	err = tl.zoneRepo.ResetScore(uint(zoneID))
	if err != nil {
		log.Printf("⚠️ Error reseteando score de zona %d: %v", zoneID, err)
		// No fallar el proceso si no se puede resetear
	} else {
		log.Printf("🔄 Score de zona %d (%s) reseteado a 0", zoneID, zoneName)
	}

	return nil
}

// publishRoutingRequest publica el request al exchange de RabbitMQ.
func (tl *TriggerLogic) publishRoutingRequest(request RoutingRequest) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error serializando payload: %w", err)
	}

	err = tl.rabbitChannel.Publish(
		"city.cleaning.planning",    // exchange
		"planning.run.requested.v1", // routing key
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent, // mensaje persistente
			MessageId:    request.RequestID,
		},
	)

	if err != nil {
		return fmt.Errorf("error publicando mensaje: %w", err)
	}

	return nil
}

// updatePendingItemsStatus actualiza el estado de todos los items pendientes de una zona.
func (tl *TriggerLogic) updatePendingItemsStatus(zoneID int, newStatus string) error {
	// Obtener todos los items pendientes de la zona
	items, err := tl.pendingRepo.FindByZone(uint(zoneID), "PENDING")
	if err != nil {
		return err
	}

	// Actualizar cada item
	for _, item := range items {
		err := tl.pendingRepo.UpdateStatus(item.IncidentID, newStatus)
		if err != nil {
			log.Printf("⚠️ Error actualizando item %s: %v", item.IncidentID, err)
			// Continuar con los demás items
		}
	}

	return nil
}

// Close cierra la conexión del canal de RabbitMQ.
func (tl *TriggerLogic) Close() error {
	if tl.rabbitChannel != nil {
		return tl.rabbitChannel.Close()
	}
	return nil
}
