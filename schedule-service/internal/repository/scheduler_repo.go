package repository

import (
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"gorm.io/gorm"
)

// ISchedulerRepository define operaciones para el scheduler de recolección.
type ISchedulerRepository interface {
	GetPendingPointsByZone(zoneID int) ([]models.PendingItem, error)
}

type SchedulerRepository struct {
	db *gorm.DB
}

// NewSchedulerRepository crea una nueva instancia del repositorio scheduler.
func NewSchedulerRepository(db *gorm.DB) *SchedulerRepository {
	return &SchedulerRepository{db: db}
}

// GetPendingPointsByZone "cosecha" todos los puntos pendientes de una zona específica.
// Retorna todos los incidentes con status 'PENDING' ordenados por fecha de creación.
func (r *SchedulerRepository) GetPendingPointsByZone(zoneID int) ([]models.PendingItem, error) {
	var items []models.PendingItem

	err := r.db.
		Where("zone_id = ? AND status = ?", zoneID, "PENDING").
		Order("created_at ASC").
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}
