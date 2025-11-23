package repository

import (
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"gorm.io/gorm"
)

// IPendingItemRepository define operaciones sobre incidentes pendientes.
type IPendingItemRepository interface {
	Create(item *models.PendingItem) error
	FindByZone(zoneID uint, status string) ([]models.PendingItem, error)
	UpdateStatus(incidentID string, newStatus string) error
	CountByZoneAndStatus(zoneID uint, status string) (int64, error)
}

type PendingItemRepository struct {
	db *gorm.DB
}

func NewPendingItemRepository(db *gorm.DB) *PendingItemRepository {
	return &PendingItemRepository{db: db}
}

// Create inserta un nuevo incidente pendiente.
func (r *PendingItemRepository) Create(item *models.PendingItem) error {
	return r.db.Create(item).Error
}

// FindByZone devuelve todos los incidentes de una zona con un status específico.
// Si status es "", devuelve todos sin filtrar.
func (r *PendingItemRepository) FindByZone(zoneID uint, status string) ([]models.PendingItem, error) {
	var items []models.PendingItem
	query := r.db.Where("zone_id = ?", zoneID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// UpdateStatus cambia el estado de un incidente específico.
func (r *PendingItemRepository) UpdateStatus(incidentID string, newStatus string) error {
	return r.db.Model(&models.PendingItem{}).
		Where("incident_id = ?", incidentID).
		Update("status", newStatus).Error
}

// CountByZoneAndStatus cuenta cuántos incidentes pendientes tiene una zona.
func (r *PendingItemRepository) CountByZoneAndStatus(zoneID uint, status string) (int64, error) {
	var count int64
	query := r.db.Model(&models.PendingItem{}).Where("zone_id = ?", zoneID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
