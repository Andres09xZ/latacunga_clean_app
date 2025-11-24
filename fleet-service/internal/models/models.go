package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TruckType representa los tipos de camiones disponibles
type TruckType string

const (
	TruckTypeCargaLateral   TruckType = "CARGA_LATERAL"
	TruckTypeCargaPosterior TruckType = "CARGA_POSTERIOR"
)

// TruckStatus representa los estados posibles de un camión
type TruckStatus string

const (
	TruckStatusDisponible    TruckStatus = "DISPONIBLE"
	TruckStatusEnUso         TruckStatus = "EN_USO"
	TruckStatusMantenimiento TruckStatus = "MANTENIMIENTO"
)

// DriverStatus representa los estados posibles de un conductor
type DriverStatus string

const (
	DriverStatusOffline    DriverStatus = "OFFLINE"
	DriverStatusDisponible DriverStatus = "DISPONIBLE"
	DriverStatusOcupado    DriverStatus = "OCUPADO"
)

// Truck representa un camión de la flota
type Truck struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	Plate     string      `gorm:"uniqueIndex;not null;size:20" json:"plate"`
	Type      TruckType   `gorm:"type:varchar(50);not null" json:"type"`
	Status    TruckStatus `gorm:"type:varchar(50);not null;default:'DISPONIBLE'" json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// TableName especifica el nombre de la tabla
func (Truck) TableName() string {
	return "trucks"
}

// Validate valida los campos del camión
func (t *Truck) Validate() error {
	if t.Plate == "" {
		return ErrInvalidPlate
	}
	if t.Type != TruckTypeCargaLateral && t.Type != TruckTypeCargaPosterior {
		return ErrInvalidTruckType
	}
	if t.Status != TruckStatusDisponible && t.Status != TruckStatusEnUso && t.Status != TruckStatusMantenimiento {
		return ErrInvalidTruckStatus
	}
	return nil
}

// Driver representa un conductor de la flota
type Driver struct {
	ID        uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	FullName  string       `gorm:"not null;size:200" json:"full_name"`
	Status    DriverStatus `gorm:"type:varchar(50);not null;default:'OFFLINE'" json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`

	// Relaciones
	OperatorProfile *OperatorProfile `gorm:"foreignKey:DriverID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"operator_profile,omitempty"`
	ActiveShifts    []ActiveShift    `gorm:"foreignKey:DriverID" json:"active_shifts,omitempty"`
}

// TableName especifica el nombre de la tabla
func (Driver) TableName() string {
	return "drivers"
}

// BeforeCreate hook para generar UUID si no existe
func (d *Driver) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// Validate valida los campos del conductor
func (d *Driver) Validate() error {
	if d.FullName == "" {
		return ErrInvalidDriverName
	}
	if d.Status != DriverStatusOffline && d.Status != DriverStatusDisponible && d.Status != DriverStatusOcupado {
		return ErrInvalidDriverStatus
	}
	return nil
}

// OperatorProfile representa el perfil de operador de un conductor
type OperatorProfile struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	DriverID          uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"driver_id"`
	UserID            *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"` // ID del usuario en auth service (opcional)
	PreferredZoneID   int        `gorm:"not null" json:"preferred_zone_id"`
	CanDriveLateral   bool       `gorm:"not null;default:false" json:"can_drive_lateral"`
	CanDrivePosterior bool       `gorm:"not null;default:false" json:"can_drive_posterior"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Relaciones
	Driver *Driver `gorm:"foreignKey:DriverID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"driver,omitempty"`
}

// TableName especifica el nombre de la tabla
func (OperatorProfile) TableName() string {
	return "operator_profiles"
}

// Validate valida los campos del perfil
func (op *OperatorProfile) Validate() error {
	if op.DriverID == uuid.Nil {
		return ErrInvalidDriverID
	}
	if op.PreferredZoneID <= 0 {
		return ErrInvalidZoneID
	}
	if !op.CanDriveLateral && !op.CanDrivePosterior {
		return ErrNoTruckTypePermission
	}
	return nil
}

// ActiveShift representa un turno activo de un conductor
type ActiveShift struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	DriverID  uuid.UUID  `gorm:"type:uuid;not null;index:idx_driver_active" json:"driver_id"`
	TruckID   uint       `gorm:"not null;index" json:"truck_id"`
	StartTime time.Time  `gorm:"not null" json:"start_time"`
	EndTime   *time.Time `gorm:"index" json:"end_time,omitempty"`
	IsActive  bool       `gorm:"not null;default:true;index:idx_driver_active" json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// Relaciones
	Driver *Driver `gorm:"foreignKey:DriverID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"driver,omitempty"`
	Truck  *Truck  `gorm:"foreignKey:TruckID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"truck,omitempty"`
}

// TableName especifica el nombre de la tabla
func (ActiveShift) TableName() string {
	return "active_shifts"
}

// BeforeCreate hook para generar UUID y setear StartTime
func (as *ActiveShift) BeforeCreate(tx *gorm.DB) error {
	if as.ID == uuid.Nil {
		as.ID = uuid.New()
	}
	if as.StartTime.IsZero() {
		as.StartTime = time.Now()
	}
	return nil
}

// Validate valida los campos del turno
func (as *ActiveShift) Validate() error {
	if as.DriverID == uuid.Nil {
		return ErrInvalidDriverID
	}
	if as.TruckID == 0 {
		return ErrInvalidTruckID
	}
	if as.StartTime.IsZero() {
		return ErrInvalidStartTime
	}
	return nil
}

// CloseShift cierra el turno activo
func (as *ActiveShift) CloseShift() {
	now := time.Now()
	as.EndTime = &now
	as.IsActive = false
}
