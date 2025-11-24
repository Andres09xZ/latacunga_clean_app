package models

import "errors"

// Errores de validación
var (
	// Errores de Truck
	ErrInvalidPlate       = errors.New("placa inválida o vacía")
	ErrInvalidTruckType   = errors.New("tipo de camión inválido")
	ErrInvalidTruckStatus = errors.New("estado de camión inválido")
	ErrTruckNotFound      = errors.New("camión no encontrado")
	ErrTruckNotAvailable  = errors.New("camión no está disponible")

	// Errores de Driver
	ErrInvalidDriverName    = errors.New("nombre de conductor inválido o vacío")
	ErrInvalidDriverStatus  = errors.New("estado de conductor inválido")
	ErrInvalidDriverID      = errors.New("ID de conductor inválido")
	ErrDriverNotFound       = errors.New("conductor no encontrado")
	ErrDriverHasActiveShift = errors.New("conductor ya tiene un turno activo")
	ErrDriverNoActiveShift  = errors.New("conductor no tiene turno activo")

	// Errores de OperatorProfile
	ErrInvalidZoneID         = errors.New("ID de zona inválido")
	ErrNoTruckTypePermission = errors.New("conductor debe poder manejar al menos un tipo de camión")
	ErrProfileNotFound       = errors.New("perfil de operador no encontrado")

	// Errores de ActiveShift
	ErrInvalidTruckID   = errors.New("ID de camión inválido")
	ErrInvalidStartTime = errors.New("hora de inicio inválida")
	ErrShiftNotFound    = errors.New("turno no encontrado")

	// Errores de asignación
	ErrNoDriverAvailable = errors.New("no hay conductores disponibles para los criterios especificados")
	ErrAllocationFailed  = errors.New("fallo al asignar conductor")
)
