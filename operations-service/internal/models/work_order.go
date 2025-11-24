package models

import (
	"time"

	"github.com/google/uuid"
)

// WorkOrderStatus representa el estado de una orden de trabajo
type WorkOrderStatus string

const (
	WorkOrderStatusAssigned   WorkOrderStatus = "ASIGNADA"
	WorkOrderStatusInProgress WorkOrderStatus = "EN_PROGRESO"
	WorkOrderStatusCompleted  WorkOrderStatus = "COMPLETADA"
)

// WorkOrder representa una orden de trabajo para un conductor
type WorkOrder struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RequestID      string          `json:"request_id" gorm:"not null;index"` // Referencia del Scheduler
	ZoneID         int             `json:"zone_id" gorm:"not null;index"`
	DriverID       uuid.UUID       `json:"driver_id" gorm:"type:uuid;not null;index"`
	TruckPlate     string          `json:"truck_plate" gorm:"not null"`
	Status         WorkOrderStatus `json:"status" gorm:"type:varchar(20);not null;default:'ASIGNADA'"`
	RoutePolyline  string          `json:"route_polyline" gorm:"type:text"` // Geometría de OSRM
	TotalStops     int             `json:"total_stops" gorm:"not null;default:0"`
	CompletedStops int             `json:"completed_stops" gorm:"not null;default:0"`
	AssignedAt     time.Time       `json:"assigned_at" gorm:"not null"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`

	// Relación con paradas
	Stops []WorkOrderStop `json:"stops,omitempty" gorm:"foreignKey:WorkOrderID;constraint:OnDelete:CASCADE"`
}

// StopStatus representa el estado de una parada
type StopStatus string

const (
	StopStatusPending      StopStatus = "PENDIENTE"
	StopStatusCollected    StopStatus = "RECOGIDO"
	StopStatusNotCollected StopStatus = "NO_RECOGIDO"
)

// WorkOrderStop representa una parada en la orden de trabajo
type WorkOrderStop struct {
	ID            uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	WorkOrderID   uuid.UUID  `json:"work_order_id" gorm:"type:uuid;not null;index"`
	WorkOrder     *WorkOrder `json:"-" gorm:"foreignKey:WorkOrderID"`
	IncidentRefID uuid.UUID  `json:"incident_ref_id" gorm:"type:uuid;not null"`
	Latitude      float64    `json:"latitude" gorm:"not null"`
	Longitude     float64    `json:"longitude" gorm:"not null"`
	Address       string     `json:"address,omitempty"`
	SequenceOrder int        `json:"sequence_order" gorm:"not null"` // Orden de visita
	Status        StopStatus `json:"status" gorm:"type:varchar(20);not null;default:'PENDIENTE'"`
	ServicedAt    *time.Time `json:"serviced_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName especifica el nombre de la tabla
func (WorkOrder) TableName() string {
	return "work_orders"
}

// TableName especifica el nombre de la tabla
func (WorkOrderStop) TableName() string {
	return "work_order_stops"
}
