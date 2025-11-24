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

// AllocationService maneja la lógica de asignación de conductores
type AllocationService struct {
	publisher *EventPublisher
}

// NewAllocationService crea una nueva instancia del servicio de asignación
func NewAllocationService(publisher *EventPublisher) *AllocationService {
	return &AllocationService{
		publisher: publisher,
	}
}

// DriverAllocation representa el resultado de una asignación
type DriverAllocation struct {
	DriverID   uuid.UUID `json:"driver_id"`
	DriverName string    `json:"driver_name"`
	TruckID    uint      `json:"truck_id"`
	TruckPlate string    `json:"truck_plate"`
	TruckType  string    `json:"truck_type"`
	ShiftID    uuid.UUID `json:"shift_id"`
}

// FindBestDriver busca y asigna el mejor conductor disponible según criterios
// CRITERIOS:
// 1. Conductor con turno activo (IsActive=true)
// 2. Conduciendo el tipo de camión requerido (truckType)
// 3. Estado 'DISPONIBLE' (no 'OCUPADO')
// 4. PRIORIDAD: Conductor con zona preferida = zoneID solicitado
// 5. Desempate: Random
// 6. LOCKING: Actualiza estado a 'OCUPADO' dentro de la misma transacción
func (s *AllocationService) FindBestDriver(zoneID int, truckType string) (*DriverAllocation, error) {
	db := database.GetDB()

	// Validar tipo de camión
	if truckType != string(models.TruckTypeCargaLateral) && truckType != string(models.TruckTypeCargaPosterior) {
		return nil, fmt.Errorf("tipo de camión inválido: %s", truckType)
	}

	// Iniciar transacción para garantizar atomicidad
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Query SQL compleja con priorización
	// La subconsulta obtiene candidatos elegibles
	// El ORDER BY prioriza por zona preferida y randomiza desempates
	query := `
		WITH eligible_drivers AS (
			SELECT 
				d.id as driver_id,
				d.full_name as driver_name,
				d.status as driver_status,
				t.id as truck_id,
				t.plate as truck_plate,
				t.type as truck_type,
				op.preferred_zone_id,
				a_s.id as shift_id,
				CASE 
					WHEN op.preferred_zone_id = ? THEN 1 
					ELSE 2 
				END as priority
			FROM drivers d
			INNER JOIN active_shifts a_s ON d.id = a_s.driver_id AND a_s.is_active = true
			INNER JOIN trucks t ON a_s.truck_id = t.id
			INNER JOIN operator_profiles op ON d.id = op.driver_id
			WHERE 
				d.status = ?
				AND t.type = ?
				AND (
					(t.type = 'CARGA_LATERAL' AND op.can_drive_lateral = true)
					OR
					(t.type = 'CARGA_POSTERIOR' AND op.can_drive_posterior = true)
				)
		)
		SELECT * FROM eligible_drivers
		ORDER BY priority ASC, RANDOM()
		LIMIT 1
	`

	var result struct {
		DriverID        uuid.UUID `gorm:"column:driver_id"`
		DriverName      string    `gorm:"column:driver_name"`
		DriverStatus    string    `gorm:"column:driver_status"`
		TruckID         uint      `gorm:"column:truck_id"`
		TruckPlate      string    `gorm:"column:truck_plate"`
		TruckType       string    `gorm:"column:truck_type"`
		PreferredZoneID int       `gorm:"column:preferred_zone_id"`
		ShiftID         uuid.UUID `gorm:"column:shift_id"`
		Priority        int       `gorm:"column:priority"`
	}

	err := tx.Raw(query, zoneID, models.DriverStatusDisponible, truckType).Scan(&result).Error

	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("⚠ No se encontró conductor disponible para zona %d y tipo %s", zoneID, truckType)
			return nil, models.ErrNoDriverAvailable
		}
		log.Printf("Error en query de búsqueda: %v", err)
		return nil, fmt.Errorf("error al buscar conductor: %w", err)
	}

	// Verificar que se encontró un conductor
	if result.DriverID == uuid.Nil {
		tx.Rollback()
		log.Printf("⚠ No hay conductores disponibles para zona %d y tipo %s", zoneID, truckType)
		return nil, models.ErrNoDriverAvailable
	}

	// CRÍTICO: Actualizar estado del conductor a OCUPADO con locking
	// Esto evita que otro proceso asigne el mismo conductor
	updateResult := tx.Model(&models.Driver{}).
		Where("id = ? AND status = ?", result.DriverID, models.DriverStatusDisponible).
		Update("status", models.DriverStatusOcupado)

	if updateResult.Error != nil {
		tx.Rollback()
		log.Printf("Error al bloquear conductor: %v", updateResult.Error)
		return nil, fmt.Errorf("error al asignar conductor: %w", updateResult.Error)
	}

	// Verificar que se actualizó exactamente 1 registro (evita race conditions)
	if updateResult.RowsAffected == 0 {
		tx.Rollback()
		log.Printf("⚠ El conductor %s ya no está disponible (race condition)", result.DriverID)
		return nil, models.ErrAllocationFailed
	}

	// Commit de la transacción
	if err := tx.Commit().Error; err != nil {
		log.Printf("Error al confirmar asignación: %v", err)
		return nil, fmt.Errorf("error al confirmar asignación: %w", err)
	}

	allocation := &DriverAllocation{
		DriverID:   result.DriverID,
		DriverName: result.DriverName,
		TruckID:    result.TruckID,
		TruckPlate: result.TruckPlate,
		TruckType:  result.TruckType,
		ShiftID:    result.ShiftID,
	}

	priorityText := "sin prioridad"
	if result.Priority == 1 {
		priorityText = fmt.Sprintf("⭐ CON PRIORIDAD (zona %d)", zoneID)
	}

	log.Printf("✓ Conductor asignado: %s (%s) - Camión: %s (%s) - %s",
		result.DriverName, result.DriverID, result.TruckPlate, result.TruckType, priorityText)

	return allocation, nil
}

// ReleaseDriver libera un conductor (cambia su estado a DISPONIBLE)
func (s *AllocationService) ReleaseDriver(driverID uuid.UUID) error {
	db := database.GetDB()

	result := db.Model(&models.Driver{}).
		Where("id = ? AND status = ?", driverID, models.DriverStatusOcupado).
		Update("status", models.DriverStatusDisponible)

	if result.Error != nil {
		log.Printf("Error al liberar conductor %s: %v", driverID, result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("⚠ Conductor %s no estaba en estado OCUPADO", driverID)
		return fmt.Errorf("conductor no estaba ocupado")
	}

	log.Printf("✓ Conductor liberado: %s (ahora DISPONIBLE)", driverID)
	return nil
}
