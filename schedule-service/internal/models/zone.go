package models

import (
	"encoding/json"
	"log"
	"time"

	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/geojson"
	"gorm.io/gorm"
)

// CleaningZone representa una zona geográfica de recolección
type CleaningZone struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ZoneName    string    `gorm:"size:100;not null" json:"zone_name"`
	RouteName   string    `gorm:"size:50;not null" json:"route_name"`
	ScheduleDay int       `gorm:"not null" json:"schedule_day"` // 0=Dom, 1=Lun, 2=Mar, 3=Mié, 4=Jue, 5=Vie, 6=Sáb
	PointsCount int       `json:"points_count"`
	AreaKm2     float64   `gorm:"type:decimal(10,4)" json:"area_km2"`
	Geom        []byte    `gorm:"type:geometry(MULTIPOLYGON,4326);not null" json:"-"`
	GeoJSON     *string   `gorm:"-" json:"geojson,omitempty"` // Campo virtual para respuestas API
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName especifica el nombre de la tabla en la base de datos
func (CleaningZone) TableName() string {
	return "cleaning_zones"
}

// AfterFind hook de GORM para convertir geometría a GeoJSON después de consultar
func (z *CleaningZone) AfterFind(tx *gorm.DB) error {
	if len(z.Geom) == 0 {
		return nil
	}

	// Intentar decodificar WKB plano. Muchos drivers PostGIS retornan EWKB (extendido con SRID),
	// si falla ignoramos la conversión y no bloqueamos la respuesta.
	geom, err := wkb.Unmarshal(z.Geom)
	if err != nil {
		log.Printf("⚠️  WKB parse failed for zone id=%d: %v (falling back without geojson)", z.ID, err)
		return nil // NO propagamos error para evitar 500 en el listado
	}

	feature := geojson.NewFeature(geom)
	feature.Properties = map[string]interface{}{
		"zone_name":    z.ZoneName,
		"route_name":   z.RouteName,
		"schedule_day": z.ScheduleDay,
		"area_km2":     z.AreaKm2,
	}

	geojsonBytes, err := json.Marshal(feature)
	if err != nil {
		log.Printf("⚠️  GeoJSON marshal failed for zone id=%d: %v", z.ID, err)
		return nil
	}

	geojsonStr := string(geojsonBytes)
	z.GeoJSON = &geojsonStr
	return nil
}

// ZoneSearchResult representa el resultado de una búsqueda espacial
type ZoneSearchResult struct {
	ZoneID         uint    `json:"zone_id"`
	ZoneName       string  `json:"zone_name"`
	RouteName      string  `json:"route_name"`
	DayName        string  `json:"day_name"`
	DistanceMeters float64 `json:"distance_meters"`
}

// ZonesByRoute vista agregada por ruta
type ZonesByRoute struct {
	RouteName        string  `json:"route_name"`
	TotalZones       int     `json:"total_zones"`
	TotalAreaKm2     float64 `json:"total_area_km2"`
	AvgPointsPerZone float64 `json:"avg_points_per_zone"`
	Zones            string  `json:"zones"`
}

// ZonesByDay vista agregada por día
type ZonesByDay struct {
	ScheduleDay  int     `json:"schedule_day"`
	DayName      string  `json:"day_name"`
	TotalZones   int     `json:"total_zones"`
	TotalAreaKm2 float64 `json:"total_area_km2"`
	Routes       string  `json:"routes"`
}

// GetDayName retorna el nombre del día en español
func GetDayName(day int) string {
	days := []string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}
	if day >= 0 && day < len(days) {
		return days[day]
	}
	return "Desconocido"
}
