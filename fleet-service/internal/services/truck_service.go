package services

import (
	"errors"
	"fmt"
	"log"

	"github.com/fleet-service/internal/database"
	"github.com/fleet-service/internal/models"
	"gorm.io/gorm"
)

// TruckService maneja la lógica de negocio para camiones
type TruckService struct{}

// NewTruckService crea una nueva instancia del servicio de camiones
func NewTruckService() *TruckService {
	return &TruckService{}
}

// CreateTruck crea un nuevo camión en la base de datos
func (s *TruckService) CreateTruck(truck *models.Truck) error {
	// Validar campos
	if err := truck.Validate(); err != nil {
		return err
	}

	db := database.GetDB()

	// Verificar si ya existe un camión con esa placa
	var existing models.Truck
	err := db.Where("plate = ?", truck.Plate).First(&existing).Error
	if err == nil {
		return fmt.Errorf("ya existe un camión con la placa '%s'", truck.Plate)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error al verificar placa: %w", err)
	}

	// Crear el camión
	if err := db.Create(truck).Error; err != nil {
		log.Printf("❌ Error al crear camión: %v", err)
		return fmt.Errorf("error al crear camión: %w", err)
	}

	log.Printf("✅ Camión creado: ID=%d, Placa=%s, Tipo=%s", truck.ID, truck.Plate, truck.Type)
	return nil
}

// GetTruckByID obtiene un camión por su ID
func (s *TruckService) GetTruckByID(id uint) (*models.Truck, error) {
	db := database.GetDB()

	var truck models.Truck
	if err := db.First(&truck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("camión con ID %d no encontrado", id)
		}
		return nil, fmt.Errorf("error al buscar camión: %w", err)
	}

	return &truck, nil
}

// GetTruckByPlate obtiene un camión por su placa
func (s *TruckService) GetTruckByPlate(plate string) (*models.Truck, error) {
	db := database.GetDB()

	var truck models.Truck
	if err := db.Where("plate = ?", plate).First(&truck).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("camión con placa '%s' no encontrado", plate)
		}
		return nil, fmt.Errorf("error al buscar camión: %w", err)
	}

	return &truck, nil
}

// ListTrucks lista todos los camiones con filtros opcionales
func (s *TruckService) ListTrucks(status *models.TruckStatus, truckType *models.TruckType) ([]models.Truck, error) {
	db := database.GetDB()

	query := db.Model(&models.Truck{})

	// Aplicar filtros si se proporcionan
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if truckType != nil {
		query = query.Where("type = ?", *truckType)
	}

	var trucks []models.Truck
	if err := query.Order("created_at DESC").Find(&trucks).Error; err != nil {
		return nil, fmt.Errorf("error al listar camiones: %w", err)
	}

	return trucks, nil
}

// UpdateTruck actualiza un camión existente
func (s *TruckService) UpdateTruck(id uint, updates map[string]interface{}) (*models.Truck, error) {
	db := database.GetDB()

	// Verificar que el camión existe
	var truck models.Truck
	if err := db.First(&truck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("camión con ID %d no encontrado", id)
		}
		return nil, fmt.Errorf("error al buscar camión: %w", err)
	}

	// Validar campos permitidos
	allowedFields := map[string]bool{
		"status": true,
		"type":   true,
		"plate":  true,
	}

	for key := range updates {
		if !allowedFields[key] {
			return nil, fmt.Errorf("campo '%s' no permitido para actualización", key)
		}
	}

	// Si se actualiza la placa, verificar que no exista otra con esa placa
	if newPlate, ok := updates["plate"].(string); ok {
		var existing models.Truck
		err := db.Where("plate = ? AND id != ?", newPlate, id).First(&existing).Error
		if err == nil {
			return nil, fmt.Errorf("ya existe otro camión con la placa '%s'", newPlate)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("error al verificar placa: %w", err)
		}
	}

	// Actualizar
	if err := db.Model(&truck).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("error al actualizar camión: %w", err)
	}

	// Recargar para devolver el modelo actualizado
	if err := db.First(&truck, id).Error; err != nil {
		return nil, fmt.Errorf("error al recargar camión: %w", err)
	}

	log.Printf("✅ Camión actualizado: ID=%d, Placa=%s", truck.ID, truck.Plate)
	return &truck, nil
}

// DeleteTruck elimina un camión (solo si no tiene turnos activos)
func (s *TruckService) DeleteTruck(id uint) error {
	db := database.GetDB()

	// Verificar que el camión existe
	var truck models.Truck
	if err := db.First(&truck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("camión con ID %d no encontrado", id)
		}
		return fmt.Errorf("error al buscar camión: %w", err)
	}

	// Verificar que no tenga turnos activos
	var activeShiftCount int64
	if err := db.Model(&models.ActiveShift{}).Where("truck_id = ? AND is_active = ?", id, true).Count(&activeShiftCount).Error; err != nil {
		return fmt.Errorf("error al verificar turnos activos: %w", err)
	}

	if activeShiftCount > 0 {
		return fmt.Errorf("no se puede eliminar el camión: tiene %d turno(s) activo(s)", activeShiftCount)
	}

	// Eliminar
	if err := db.Delete(&truck).Error; err != nil {
		return fmt.Errorf("error al eliminar camión: %w", err)
	}

	log.Printf("✅ Camión eliminado: ID=%d, Placa=%s", truck.ID, truck.Plate)
	return nil
}

// GetAvailableTrucks obtiene camiones disponibles por tipo
func (s *TruckService) GetAvailableTrucks(truckType models.TruckType) ([]models.Truck, error) {
	status := models.TruckStatusDisponible
	return s.ListTrucks(&status, &truckType)
}
