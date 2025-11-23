package repository

import (
	"errors"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"gorm.io/gorm"
)

type IncidentRepository struct{ db *gorm.DB }

func NewIncidentRepository(db *gorm.DB) *IncidentRepository { return &IncidentRepository{db: db} }

// CreateIncident persists a new incident row
func (r *IncidentRepository) CreateIncident(zone *models.CleaningZone, incidentType string, points int, lat, lon float64) (*models.Incident, error) {
	inc := &models.Incident{ZoneID: &zone.ID, ZoneName: zone.ZoneName, IncidentType: incidentType, Points: points, Lat: lat, Lon: lon, CreatedAt: time.Now().UTC()}
	if err := r.db.Create(inc).Error; err != nil {
		return nil, err
	}
	return inc, nil
}

// GetPendingIncidents returns incidents without work_order_id for a zone
func (r *IncidentRepository) GetPendingIncidents(zoneID uint) ([]models.Incident, error) {
	var list []models.Incident
	if err := r.db.Where("zone_id = ? AND work_order_id IS NULL", zoneID).Order("created_at ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// CreateWorkOrderWithPending creates a work order and attaches all pending incidents atomically.
func (r *IncidentRepository) CreateWorkOrderWithPending(zone *models.CleaningZone, threshold int) (*models.WorkOrder, []models.Incident, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, nil, tx.Error
	}
	var incidents []models.Incident
	if err := tx.Where("zone_id = ? AND work_order_id IS NULL", zone.ID).Find(&incidents).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	// if no incidents, create synthetic trigger
	if len(incidents) == 0 {
		synth := models.Incident{ZoneID: &zone.ID, ZoneName: zone.ZoneName, IncidentType: "TRIGGER_MANUAL", Points: 0, Lat: 0, Lon: 0, CreatedAt: time.Now().UTC()}
		if err := tx.Create(&synth).Error; err != nil {
			tx.Rollback()
			return nil, nil, err
		}
		incidents = append(incidents, synth)
	}
	total := 0
	for _, i := range incidents {
		total += i.Points
	}
	wo := &models.WorkOrder{ZoneID: zone.ID, ZoneName: zone.ZoneName, GeneratedAt: time.Now().UTC(), TotalPoints: total, Threshold: threshold}
	if err := tx.Create(wo).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	if err := tx.Model(&models.Incident{}).Where("zone_id = ? AND work_order_id IS NULL", zone.ID).Update("work_order_id", wo.ID).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, nil, err
	}
	// refresh with updated work_order_id
	if err := r.db.Where("work_order_id = ?", wo.ID).Order("created_at ASC").Find(&incidents).Error; err != nil {
		return wo, incidents, err
	}
	return wo, incidents, nil
}

// ResetZoneIncidents optional helper to detach pending incidents (not used currently)
func (r *IncidentRepository) ResetZoneIncidents(zoneID uint) error {
	return r.db.Model(&models.Incident{}).Where("zone_id = ? AND work_order_id IS NULL", zoneID).Update("work_order_id", nil).Error
}

// GetWorkOrder returns a work order with its incidents
func (r *IncidentRepository) GetWorkOrder(id uint) (*models.WorkOrder, []models.Incident, error) {
	var wo models.WorkOrder
	if err := r.db.First(&wo, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var incidents []models.Incident
	if err := r.db.Where("work_order_id = ?", wo.ID).Order("created_at ASC").Find(&incidents).Error; err != nil {
		return &wo, nil, err
	}
	return &wo, incidents, nil
}
