package models

import "github.com/google/uuid"

// ========================
// FLEET SERVICE DTOs
// ========================

// FleetRequest solicita recursos (chofer, camión) para una zona
type FleetRequest struct {
	ZoneID       int    `json:"zone_id"`
	RequiredType string `json:"required_type"` // Ej: "STANDARD", "COMPACTOR"
}

// FleetResponse contiene los recursos asignados
type FleetResponse struct {
	DriverID    uuid.UUID  `json:"driver_id"`
	AssistantID *uuid.UUID `json:"assistant_id,omitempty"` // Opcional
	TruckPlate  string     `json:"truck_plate"`
	Status      string     `json:"status"` // "ALLOCATED"
}

// FleetReleaseRequest libera recursos asignados (compensación)
type FleetReleaseRequest struct {
	DriverID   uuid.UUID `json:"driver_id"`
	TruckPlate string    `json:"truck_plate"`
	Reason     string    `json:"reason"` // Ej: "ROUTING_FAILED"
}

// ========================
// ROUTING SERVICE DTOs
// ========================

// GeoPoint representa un punto geográfico con información del incidente
type GeoPoint struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	IncidentID    string  `json:"incident_id"`
	GravityPoints int     `json:"gravity_points"`
}

// RoutingRequest solicita optimización de ruta
type RoutingRequest struct {
	RequestID string     `json:"request_id"`
	ZoneID    int        `json:"zone_id"`
	ZoneName  string     `json:"zone_name"`
	Points    []GeoPoint `json:"points"` // Debe incluir DEPOT al inicio
}

// RoutingResponse contiene la ruta optimizada
type RoutingResponse struct {
	RequestID      string  `json:"request_id"`
	Polyline       string  `json:"polyline"`        // Geometría codificada
	DistanceKm     float64 `json:"distance_km"`     // Distancia en kilómetros
	DurationMin    int     `json:"duration_min"`    // Duración en minutos
	OptimizedOrder []int   `json:"optimized_order"` // Orden optimizado de waypoints
	WaypointOrder  string  `json:"waypoint_order"`  // JSON string del orden
}

// ========================
// OPERATIONS SERVICE DTOs
// ========================

// WorkOrderEvent es el evento final que se envía a Operaciones
type WorkOrderEvent struct {
	OrderID     uuid.UUID  `json:"order_id"`
	ZoneID      int        `json:"zone_id"`
	ZoneName    string     `json:"zone_name"`
	DriverID    uuid.UUID  `json:"driver_id"`
	AssistantID *uuid.UUID `json:"assistant_id,omitempty"`
	TruckPlate  string     `json:"truck_plate"`
	RouteID     string     `json:"route_id"`
	Polyline    string     `json:"polyline"`
	DistanceKm  float64    `json:"distance_km"`
	DurationMin int        `json:"duration_min"`
	IncidentIDs []string   `json:"incident_ids"`
	Status      string     `json:"status"` // "PENDING"
	CreatedAt   string     `json:"created_at"`
}
