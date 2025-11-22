package repository

import (
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"gorm.io/gorm"
)

// ZoneRepository maneja las operaciones de base de datos para zonas
type ZoneRepository struct {
	db *gorm.DB
}

// NewZoneRepository crea una nueva instancia del repositorio
func NewZoneRepository(db *gorm.DB) *ZoneRepository {
	return &ZoneRepository{db: db}
}

// FindAll retorna todas las zonas
func (r *ZoneRepository) FindAll() ([]models.CleaningZone, error) {
	var zones []models.CleaningZone
	// Omitir la columna geom para evitar errores de decodificación WKB en listados
	err := r.db.Omit("Geom").Find(&zones).Error
	return zones, err
}

// FindByID retorna una zona por su ID
func (r *ZoneRepository) FindByID(id uint) (*models.CleaningZone, error) {
	var zone models.CleaningZone
	err := r.db.Omit("Geom").First(&zone, id).Error
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

// FindByRouteName retorna todas las zonas de una ruta específica
func (r *ZoneRepository) FindByRouteName(routeName string) ([]models.CleaningZone, error) {
	var zones []models.CleaningZone
	err := r.db.Omit("Geom").Where("route_name = ?", routeName).Find(&zones).Error
	return zones, err
}

// FindByScheduleDay retorna todas las zonas de un día específico
func (r *ZoneRepository) FindByScheduleDay(day int) ([]models.CleaningZone, error) {
	var zones []models.CleaningZone
	err := r.db.Omit("Geom").Where("schedule_day = ?", day).Find(&zones).Error
	return zones, err
}

// FindByRouteAndDay retorna la zona específica de una ruta en un día
func (r *ZoneRepository) FindByRouteAndDay(routeName string, day int) (*models.CleaningZone, error) {
	var zone models.CleaningZone
	err := r.db.Omit("Geom").Where("route_name = ? AND schedule_day = ?", routeName, day).First(&zone).Error
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

// FindZoneByPoint busca la zona que contiene un punto geográfico
// Usa función PostGIS ST_Contains para búsqueda espacial
func (r *ZoneRepository) FindZoneByPoint(latitude, longitude float64) (*models.ZoneSearchResult, error) {
	var result models.ZoneSearchResult

	query := `
		SELECT 
			cz.id as zone_id,
			cz.zone_name,
			cz.route_name,
			CASE cz.schedule_day
				WHEN 0 THEN 'Domingo'
				WHEN 1 THEN 'Lunes'
				WHEN 2 THEN 'Martes'
				WHEN 3 THEN 'Miércoles'
				WHEN 4 THEN 'Jueves'
				WHEN 5 THEN 'Viernes'
				WHEN 6 THEN 'Sábado'
			END as day_name,
			ST_Distance(
				cz.geom::geography,
				ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
			) as distance_meters
		FROM cleaning_zones cz
		WHERE ST_Contains(
			cz.geom,
			ST_SetSRID(ST_MakePoint($2, $1), 4326)
		)
		ORDER BY distance_meters
		LIMIT 1
	`

	err := r.db.Raw(query, latitude, longitude).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	// Si no se encontró ninguna zona (result vacío), retornar error
	if result.ZoneID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return &result, nil
}

// FindNearestZone busca la zona más cercana a un punto (aunque no lo contenga)
func (r *ZoneRepository) FindNearestZone(latitude, longitude float64) (*models.ZoneSearchResult, error) {
	var result models.ZoneSearchResult

	query := `
		SELECT 
			cz.id as zone_id,
			cz.zone_name,
			cz.route_name,
			CASE cz.schedule_day
				WHEN 0 THEN 'Domingo'
				WHEN 1 THEN 'Lunes'
				WHEN 2 THEN 'Martes'
				WHEN 3 THEN 'Miércoles'
				WHEN 4 THEN 'Jueves'
				WHEN 5 THEN 'Viernes'
				WHEN 6 THEN 'Sábado'
			END as day_name,
			ST_Distance(
				cz.geom::geography,
				ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
			) as distance_meters
		FROM cleaning_zones cz
		ORDER BY cz.geom::geography <-> ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
		LIMIT 1
	`

	err := r.db.Raw(query, latitude, longitude).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetZonesByRoute retorna resumen agrupado por ruta
func (r *ZoneRepository) GetZonesByRoute() ([]models.ZonesByRoute, error) {
	var results []models.ZonesByRoute
	err := r.db.Table("zones_by_route").Find(&results).Error
	return results, err
}

// GetZonesByDay retorna resumen agrupado por día
func (r *ZoneRepository) GetZonesByDay() ([]models.ZonesByDay, error) {
	var results []models.ZonesByDay
	err := r.db.Table("zones_by_day").Order("schedule_day").Find(&results).Error
	return results, err
}

// GetZoneWithGeoJSON retorna una zona con su geometría en formato GeoJSON
func (r *ZoneRepository) GetZoneWithGeoJSON(id uint) (map[string]interface{}, error) {
	var result map[string]interface{}

	query := `
		SELECT 
			id,
			zone_name,
			route_name,
			schedule_day,
			points_count,
			area_km2,
			ST_AsGeoJSON(geom) as geojson,
			created_at,
			updated_at
		FROM cleaning_zones
		WHERE id = $1
	`

	err := r.db.Raw(query, id).Scan(&result).Error
	return result, err
}

// GetAllZonesAsGeoJSON retorna todas las zonas en formato GeoJSON FeatureCollection
func (r *ZoneRepository) GetAllZonesAsGeoJSON() (string, error) {
	var result string

	query := `
		SELECT jsonb_build_object(
			'type', 'FeatureCollection',
			'features', jsonb_agg(feature)
		)::text
		FROM (
			SELECT jsonb_build_object(
				'type', 'Feature',
				'id', id,
				'geometry', ST_AsGeoJSON(geom)::jsonb,
				'properties', jsonb_build_object(
					'zone_name', zone_name,
					'route_name', route_name,
					'schedule_day', schedule_day,
					'points_count', points_count,
					'area_km2', area_km2
				)
			) as feature
			FROM cleaning_zones
		) features
	`

	err := r.db.Raw(query).Scan(&result).Error
	return result, err
}
