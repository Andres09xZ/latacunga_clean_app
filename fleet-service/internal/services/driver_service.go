package services

import (
	"errors"
	"fmt"
	"log"

	"github.com/fleet-service/internal/database"
	"github.com/fleet-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DriverService maneja la lógica de negocio para conductores
type DriverService struct{}

// NewDriverService crea una nueva instancia del servicio de conductores
func NewDriverService() *DriverService {
	return &DriverService{}
}

// GetDriverByID obtiene un conductor por su ID con su perfil de operador
func (s *DriverService) GetDriverByID(id uuid.UUID) (*models.Driver, error) {
	db := database.GetDB()

	var driver models.Driver
	if err := db.Preload("OperatorProfile").Preload("ActiveShifts").First(&driver, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("conductor con ID %s no encontrado", id)
		}
		return nil, fmt.Errorf("error al buscar conductor: %w", err)
	}

	return &driver, nil
}

// ListDrivers lista todos los conductores con filtros opcionales
func (s *DriverService) ListDrivers(status *models.DriverStatus, zoneID *int) ([]models.Driver, error) {
	db := database.GetDB()

	query := db.Model(&models.Driver{}).Preload("OperatorProfile").Preload("ActiveShifts")

	// Aplicar filtros si se proporcionan
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// Filtrar por zona preferida a través del perfil de operador
	if zoneID != nil {
		query = query.Joins("JOIN operator_profiles ON operator_profiles.driver_id = drivers.id").
			Where("operator_profiles.preferred_zone_id = ?", *zoneID)
	}

	var drivers []models.Driver
	if err := query.Order("created_at DESC").Find(&drivers).Error; err != nil {
		return nil, fmt.Errorf("error al listar conductores: %w", err)
	}

	return drivers, nil
}

// UpdateDriver actualiza un conductor existente
func (s *DriverService) UpdateDriver(id uuid.UUID, updates map[string]interface{}) (*models.Driver, error) {
	db := database.GetDB()

	// Verificar que el conductor existe
	var driver models.Driver
	if err := db.First(&driver, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("conductor con ID %s no encontrado", id)
		}
		return nil, fmt.Errorf("error al buscar conductor: %w", err)
	}

	// Validar campos permitidos
	allowedFields := map[string]bool{
		"full_name": true,
		"status":    true,
	}

	for key := range updates {
		if !allowedFields[key] {
			return nil, fmt.Errorf("campo '%s' no permitido para actualización", key)
		}
	}

	// Actualizar
	if err := db.Model(&driver).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("error al actualizar conductor: %w", err)
	}

	// Recargar con relaciones
	if err := db.Preload("OperatorProfile").Preload("ActiveShifts").First(&driver, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("error al recargar conductor: %w", err)
	}

	log.Printf("✅ Conductor actualizado: ID=%s, Nombre=%s", driver.ID, driver.FullName)
	return &driver, nil
}

// UpdateOperatorProfile actualiza el perfil de operador de un conductor
func (s *DriverService) UpdateOperatorProfile(driverID uuid.UUID, updates map[string]interface{}) (*models.OperatorProfile, error) {
	db := database.GetDB()

	// Verificar que el conductor existe
	var driver models.Driver
	if err := db.First(&driver, "id = ?", driverID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("conductor con ID %s no encontrado", driverID)
		}
		return nil, fmt.Errorf("error al buscar conductor: %w", err)
	}

	// Buscar el perfil de operador
	var profile models.OperatorProfile
	if err := db.Where("driver_id = ?", driverID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("perfil de operador no encontrado para conductor %s", driverID)
		}
		return nil, fmt.Errorf("error al buscar perfil: %w", err)
	}

	// Validar campos permitidos
	allowedFields := map[string]bool{
		"preferred_zone_id":   true,
		"can_drive_lateral":   true,
		"can_drive_posterior": true,
	}

	for key := range updates {
		if !allowedFields[key] {
			return nil, fmt.Errorf("campo '%s' no permitido para actualización", key)
		}
	}

	// Actualizar
	if err := db.Model(&profile).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("error al actualizar perfil: %w", err)
	}

	// Recargar
	if err := db.First(&profile, profile.ID).Error; err != nil {
		return nil, fmt.Errorf("error al recargar perfil: %w", err)
	}

	log.Printf("✅ Perfil de operador actualizado: DriverID=%s, Zona=%d", driverID, profile.PreferredZoneID)
	return &profile, nil
}

// DeleteDriver elimina un conductor (solo si no tiene turnos activos)
func (s *DriverService) DeleteDriver(id uuid.UUID) error {
	db := database.GetDB()

	// Verificar que el conductor existe
	var driver models.Driver
	if err := db.First(&driver, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("conductor con ID %s no encontrado", id)
		}
		return fmt.Errorf("error al buscar conductor: %w", err)
	}

	// Verificar que no tenga turnos activos
	var activeShiftCount int64
	if err := db.Model(&models.ActiveShift{}).Where("driver_id = ? AND is_active = ?", id, true).Count(&activeShiftCount).Error; err != nil {
		return fmt.Errorf("error al verificar turnos activos: %w", err)
	}

	if activeShiftCount > 0 {
		return fmt.Errorf("no se puede eliminar el conductor: tiene %d turno(s) activo(s)", activeShiftCount)
	}

	// Eliminar (cascade eliminará el perfil de operador automáticamente)
	if err := db.Delete(&driver).Error; err != nil {
		return fmt.Errorf("error al eliminar conductor: %w", err)
	}

	log.Printf("✅ Conductor eliminado: ID=%s, Nombre=%s", driver.ID, driver.FullName)
	return nil
}

// GetAvailableDrivers obtiene conductores disponibles con turno activo
func (s *DriverService) GetAvailableDrivers(zoneID *int) ([]models.Driver, error) {
	db := database.GetDB()

	query := db.Model(&models.Driver{}).
		Preload("OperatorProfile").
		Preload("ActiveShifts").
		Joins("JOIN active_shifts ON active_shifts.driver_id = drivers.id AND active_shifts.is_active = true").
		Where("drivers.status = ?", models.DriverStatusDisponible)

	// Filtrar por zona preferida si se proporciona
	if zoneID != nil {
		query = query.Joins("JOIN operator_profiles ON operator_profiles.driver_id = drivers.id").
			Where("operator_profiles.preferred_zone_id = ?", *zoneID)
	}

	var drivers []models.Driver
	if err := query.Find(&drivers).Error; err != nil {
		return nil, fmt.Errorf("error al buscar conductores disponibles: %w", err)
	}

	return drivers, nil
}

// GetDriverStats obtiene estadísticas de un conductor
func (s *DriverService) GetDriverStats(driverID uuid.UUID) (map[string]interface{}, error) {
	db := database.GetDB()

	// Verificar que el conductor existe
	var driver models.Driver
	if err := db.First(&driver, "id = ?", driverID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("conductor con ID %s no encontrado", driverID)
		}
		return nil, fmt.Errorf("error al buscar conductor: %w", err)
	}

	// Contar turnos totales
	var totalShifts int64
	db.Model(&models.ActiveShift{}).Where("driver_id = ?", driverID).Count(&totalShifts)

	// Contar turnos activos
	var activeShifts int64
	db.Model(&models.ActiveShift{}).Where("driver_id = ? AND is_active = ?", driverID, true).Count(&activeShifts)

	// Contar turnos completados
	var completedShifts int64
	db.Model(&models.ActiveShift{}).Where("driver_id = ? AND is_active = ?", driverID, false).Count(&completedShifts)

	stats := map[string]interface{}{
		"driver_id":         driverID,
		"full_name":         driver.FullName,
		"status":            driver.Status,
		"total_shifts":      totalShifts,
		"active_shifts":     activeShifts,
		"completed_shifts":  completedShifts,
	}

	return stats, nil
}
