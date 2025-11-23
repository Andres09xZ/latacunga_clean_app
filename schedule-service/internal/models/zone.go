package models

import (
	"encoding/json"
	"log"
	"time"

	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/geojson"
	"gorm.io/gorm"
)

// CleaningZone representa una zona macro de planificación.
// Campos según tabla cleaning_zones (migrations 005 + 012).
type CleaningZone struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ZoneName       string    `gorm:"size:100;not null" json:"zone_name"`       // Ej: "RUTA 1 LUNES"
	RouteName      string    `gorm:"size:50;not null" json:"route_name"`       // Ej: "RUTA_1"
	ScheduleDay    int       `gorm:"not null" json:"schedule_day"`             // 0=Domingo, 1=Lunes... 6=Sábado
	ScheduleConfig string    `gorm:"type:text" json:"schedule_config"`         // Ej: "NOCTURNO (21:00)", "DIURNO (Mar-Jue-Sab)"
	Status         string    `gorm:"size:40;default:ACUMULANDO" json:"status"` // ACUMULANDO | LISTO_PARA_RECOLECCION | EN_PROGRESO
	PointsCount    int       `gorm:"default:0" json:"points_count"`            // Cantidad de puntos en geometría
	AreaKm2        *float64  `gorm:"type:decimal(10,4)" json:"area_km2"`       // Área aproximada
	Geom           []byte    `gorm:"type:geometry(MULTIPOLYGON,4326)" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (CleaningZone) TableName() string { return "cleaning_zones" }

// AfterFind: convierte la geometría a GeoJSON simple.
func (z *CleaningZone) AfterFind(tx *gorm.DB) error {
	if len(z.Geom) == 0 {
		return nil
	}
	geom, err := wkb.Unmarshal(z.Geom)
	if err != nil {
		log.Printf("geom parse failed id=%d: %v", z.ID, err)
		return nil
	}
	feature := geojson.NewFeature(geom)
	feature.Properties = map[string]interface{}{
		"zone_name":       z.ZoneName,
		"route_name":      z.RouteName,
		"schedule_day":    z.ScheduleDay,
		"schedule_config": z.ScheduleConfig,
		"status":          z.Status,
		"points_count":    z.PointsCount,
	}
	b, err := json.Marshal(feature)
	if err != nil {
		return nil
	}
	geo := string(b)
	// expose via pointer if needed by handler
	_ = geo
	return nil
}

// ZoneMetrics representa el estado de acumulación separado de la definición geográfica.
type ZoneMetrics struct {
	ZoneID       uint       `gorm:"primaryKey" json:"zone_id"`
	CurrentScore int        `gorm:"not null;default:0" json:"current_score"`
	Threshold    int        `gorm:"not null;default:50" json:"threshold"`
	LastTrigger  *time.Time `json:"last_trigger"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (ZoneMetrics) TableName() string { return "zone_metrics" }

// SimulateIncidentRequest representa el payload para simular un incidente.
type SimulateIncidentRequest struct {
	Lat  float64 `json:"lat" example:"-0.935" binding:"required"`   // Centro de URBANO_CENTRAL
	Lon  float64 `json:"lon" example:"-78.6175" binding:"required"` // Centro de URBANO_CENTRAL
	Type string  `json:"type" example:"SENSOR_LLENO" binding:"required"`
}

// PlanningResult representa la respuesta de evaluar un incidente.
type PlanningResult struct {
	ZoneID         uint      `json:"zone_id"`
	ZoneName       string    `json:"zone_name"`
	NewScore       int       `json:"score"`
	Threshold      int       `json:"threshold"`
	Triggered      bool      `json:"triggered"`
	Status         string    `json:"status"`
	ScheduledTime  time.Time `json:"scheduled_time"`
	ScheduledLabel string    `json:"scheduled_label"`
	Reason         string    `json:"reason,omitempty"`
}

// PendingItem representa un incidente pendiente almacenado temporalmente.
// Se guarda cada incidente recibido mientras la zona acumula puntos.
type PendingItem struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Lat           float64   `gorm:"not null" json:"lat"`                                                       // Latitud del incidente
	Lon           float64   `gorm:"not null" json:"lon"`                                                       // Longitud del incidente
	IncidentID    string    `gorm:"size:100;uniqueIndex" json:"incident_id"`                                   // ID único del incidente
	ZoneID        uint      `gorm:"not null;index:idx_pending_zone" json:"zone_id"`                            // Zona donde ocurrió
	GravityPoints int       `gorm:"not null" json:"gravity_points"`                                            // Puntos asignados según tipo
	Status        string    `gorm:"size:20;not null;default:'PENDING';index:idx_pending_status" json:"status"` // PENDING | PROCESSED | CANCELLED
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (PendingItem) TableName() string { return "pending_items" }
