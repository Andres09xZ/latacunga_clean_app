package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Point representa una coordenada geográfica (lat, lon)
// Acepta tanto "lat"/"lon" como "latitude"/"longitude" para compatibilidad
type Point struct {
	Latitude      float64 `json:"lat"`
	Longitude     float64 `json:"lon"`
	IncidentID    string  `json:"incident_id,omitempty"`
	GravityPoints int     `json:"gravity_points,omitempty"`
}

// UnmarshalJSON maneja ambos formatos: lat/lon y latitude/longitude
func (p *Point) UnmarshalJSON(data []byte) error {
	// Intentar con el formato estándar primero (lat/lon)
	type Alias Point
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Si lat/lon están vacíos, intentar con latitude/longitude
	if p.Latitude == 0 && p.Longitude == 0 {
		var altFormat struct {
			Latitude      float64 `json:"latitude"`
			Longitude     float64 `json:"longitude"`
			IncidentID    string  `json:"incident_id,omitempty"`
			GravityPoints int     `json:"gravity_points,omitempty"`
		}
		if err := json.Unmarshal(data, &altFormat); err == nil {
			p.Latitude = altFormat.Latitude
			p.Longitude = altFormat.Longitude
			p.IncidentID = altFormat.IncidentID
			p.GravityPoints = altFormat.GravityPoints
		}
	}

	return nil
}

// RouteRequest representa una solicitud de optimización de ruta
type RouteRequest struct {
	RequestID string  `json:"request_id" binding:"required"`
	ZoneID    uint    `json:"zone_id" binding:"required"`
	Points    []Point `json:"points" binding:"required,min=2"`
}

// RoutePlan mapea a la tabla route_plans en PostgreSQL
type RoutePlan struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RequestID     string    `gorm:"size:100;not null;index" json:"request_id"`
	ZoneID        uint      `gorm:"not null;index" json:"zone_id"`
	Distance      float64   `gorm:"type:decimal(10,2);not null" json:"distance_meters"`  // en metros
	Duration      float64   `gorm:"type:decimal(10,2);not null" json:"duration_seconds"` // en segundos
	Geometry      string    `gorm:"type:text;not null" json:"geometry"`                  // Polyline codificado
	WaypointOrder string    `gorm:"type:text" json:"waypoint_order"`                     // JSON array de índices
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName especifica el nombre de la tabla
func (RoutePlan) TableName() string {
	return "route_plans"
}

// RouteResponse representa la respuesta final que se envía via RabbitMQ
type RouteResponse struct {
	RequestID     string  `json:"request_id"`
	ZoneID        uint    `json:"zone_id"`
	Distance      float64 `json:"distance_meters"`
	Duration      float64 `json:"duration_seconds"`
	Geometry      string  `json:"geometry"`
	WaypointOrder []int   `json:"waypoint_order"`
	OptimizedAt   string  `json:"optimized_at"`
}
