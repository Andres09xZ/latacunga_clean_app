package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"gorm.io/gorm"
)

// IZoneRepository define las operaciones requeridas por el servicio de planificación.
type IZoneRepository interface {
	FindZoneByPoint(lat, lon float64) (*models.CleaningZone, error)
	ListZones() ([]models.CleaningZone, error)
	GetMetrics(zoneID uint) (*models.ZoneMetrics, error)
	UpsertMetrics(m *models.ZoneMetrics) error
	UpdateStatus(zoneID uint, status string) error
}

type ZoneRepository struct{ db *gorm.DB }

func NewZoneRepository(db *gorm.DB) *ZoneRepository { return &ZoneRepository{db: db} }

func (r *ZoneRepository) ListZones() ([]models.CleaningZone, error) {
	var zones []models.CleaningZone

	// Usar query raw para evitar el error de cached plan en Neon PostgreSQL
	query := `SELECT id, zone_name, route_name, schedule_day, points_count, area_km2, 
	          geom, created_at, updated_at, schedule_config, status 
	          FROM cleaning_zones ORDER BY id`

	if err := r.db.Raw(query).Scan(&zones).Error; err != nil {
		return nil, err
	}
	return zones, nil
}

// FindZoneByPoint: búsqueda espacial usando ST_Contains sobre MULTIPOLYGON.
func (r *ZoneRepository) FindZoneByPoint(lat, lon float64) (*models.CleaningZone, error) {
	var zone models.CleaningZone
	// ST_MakePoint recibe (lon, lat) - orden X,Y
	q := `SELECT * FROM cleaning_zones WHERE ST_Contains(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)) LIMIT 1`
	if err := r.db.Raw(q, lon, lat).Scan(&zone).Error; err != nil {
		return nil, err
	}
	if zone.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &zone, nil
}

func (r *ZoneRepository) GetMetrics(zoneID uint) (*models.ZoneMetrics, error) {
	var m models.ZoneMetrics
	if err := r.db.First(&m, zoneID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ZoneRepository) UpsertMetrics(m *models.ZoneMetrics) error {
	// Use Save for upsert behavior
	return r.db.Save(m).Error
}

func (r *ZoneRepository) UpdateStatus(zoneID uint, status string) error {
	if status == "" {
		return errors.New("status vacío")
	}
	return r.db.Model(&models.CleaningZone{}).Where("id = ?", zoneID).Update("status", status).Error
}

// IncrementScore aplica acumulación y determina si se supera el umbral.
func (r *ZoneRepository) IncrementScore(zoneID uint, delta int) (*models.ZoneMetrics, bool, error) {
	m, err := r.GetMetrics(zoneID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m = &models.ZoneMetrics{ZoneID: zoneID, Threshold: 50, CurrentScore: 0}
		} else {
			return nil, false, err
		}
	}
	m.CurrentScore += delta
	triggered := m.CurrentScore >= m.Threshold
	if triggered {
		now := time.Now().UTC()
		m.LastTrigger = &now
	}
	if err := r.UpsertMetrics(m); err != nil {
		return nil, false, err
	}
	return m, triggered, nil
}

// ForceTrigger marca trigger sin alterar score necesariamente.
func (r *ZoneRepository) ForceTrigger(zoneID uint, reason string) (*models.ZoneMetrics, error) {
	m, err := r.GetMetrics(zoneID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m = &models.ZoneMetrics{ZoneID: zoneID, Threshold: 50, CurrentScore: 0}
		} else {
			return nil, err
		}
	}
	now := time.Now().UTC()
	m.LastTrigger = &now
	if err := r.UpsertMetrics(m); err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateThresholds permite actualizar múltiples umbrales.
func (r *ZoneRepository) UpdateThresholds(th map[string]int) error {
	for zoneName, value := range th {
		if value <= 0 {
			return fmt.Errorf("threshold inválido para %s", zoneName)
		}
		// Obtener zone id
		var z models.CleaningZone
		if err := r.db.Where("zone_name = ?", zoneName).First(&z).Error; err != nil {
			return err
		}
		var m models.ZoneMetrics
		if err := r.db.First(&m, z.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				m = models.ZoneMetrics{ZoneID: z.ID, Threshold: value}
				if err := r.db.Create(&m).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if err := r.db.Model(&m).Update("threshold", value).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
