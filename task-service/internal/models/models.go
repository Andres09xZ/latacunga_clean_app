package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================
// Actor: Workers/Operators
// ============================================================

// Actor represents a worker or operator
type Actor struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	ActorType  string     `json:"actor_type" gorm:"not null"` // WORKER, OPERATOR
	Location   string     `json:"location"`                   // lat,lng as string for simplicity
	ShiftStart *time.Time `json:"shift_start"`
	ShiftEnd   *time.Time `json:"shift_end"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	Status     string     `json:"status" gorm:"default:ACTIVO"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ============================================================
// Task: Represents a task created from a novedad or report
// ============================================================

// Task state constants
const (
	TaskStatePendiente      = "PENDIENTE"
	TaskStateEnProgreso     = "EN_PROGRESO"
	TaskStateCompletada     = "COMPLETADA"
	TaskStateCancelada      = "CANCELADA"
	TaskStatePendienteAsign = "PENDIENTE_ASIGNAR"
)

// Task source constants
const (
	TaskSourceNovedad = "novedad"
	TaskSourceReport  = "report"
)

// Task type constants
const (
	TaskTypePuntoAcopio = "PUNTO_ACOPIO"
	TaskTypeZonaCritica = "ZONA_CRITICA"
	TaskTypeLimpieza    = "LIMPIEZA"
	TaskTypeRecoleccion = "RECOLECCION"
)

// Task represents a task assigned to an actor
type Task struct {
	ID           uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	NovedadID    *uuid.UUID    `json:"novedad_id" gorm:"type:uuid;index"`        // Foreign key to novedad if source=novedad
	ReportID     *uuid.UUID    `json:"report_id" gorm:"type:uuid;index"`         // Foreign key to report if source=report
	ActorID      *uuid.UUID    `json:"actor_id" gorm:"type:uuid;index"`          // Assigned worker/operator
	Source       string        `json:"source" gorm:"not null;default:'novedad'"` // 'novedad' or 'report'
	Type         string        `json:"type" gorm:"not null;default:'LIMPIEZA'"`  // PUNTO_ACOPIO, ZONA_CRITICA, LIMPIEZA, RECOLECCION
	State        string        `json:"state" gorm:"not null;default:'PENDIENTE'"`
	Priority     int           `json:"priority" gorm:"default:0"`
	Instructions string        `json:"instructions"`
	Description  string        `json:"description"`
	Latitude     *float64      `json:"latitude"`
	Longitude    *float64      `json:"longitude"`
	PhotoURL     *string       `json:"photo_url,omitempty"`
	Evidence     []string      `json:"evidence,omitempty" gorm:"type:jsonb;serializer:json"` // Photo URLs for completed tasks
	StartedAt    *time.Time    `json:"started_at"`
	CompletedAt  *time.Time    `json:"completed_at"`
	Version      int           `json:"version" gorm:"default:1"` // For optimistic locking
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Histories    []TaskHistory `json:"histories,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
}

// ============================================================
// TaskHistory: Audit trail for task state changes
// ============================================================

type TaskHistory struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID    uuid.UUID `json:"task_id" gorm:"type:uuid;not null;index"`
	ActorID   uuid.UUID `json:"actor_id" gorm:"type:uuid;not null"` // Who made the change
	OldState  string    `json:"old_state" gorm:"not null"`
	NewState  string    `json:"new_state" gorm:"not null"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ============================================================
// Event Processing: Idempotency and Audit
// ============================================================

// ProcessedEvent tracks which RabbitMQ events have been processed
type ProcessedEvent struct {
	EventID     uuid.UUID `json:"event_id" gorm:"type:uuid;primary_key"`
	Consumer    string    `json:"consumer" gorm:"not null;index"`
	EventType   string    `json:"event_type" gorm:"not null"` // e.g., 'novedad.verificada', 'novedad.creada'
	SourceID    uuid.UUID `json:"source_id" gorm:"type:uuid"` // ID of the novedad/report that triggered the event
	ProcessedAt time.Time `json:"processed_at"`
}

// ============================================================
// DTOs for Request/Response
// ============================================================

// CreateTaskRequest represents a request to create a task manually
type CreateTaskRequest struct {
	NovedadID    *uuid.UUID `json:"novedad_id" binding:"omitempty"`
	ReportID     *uuid.UUID `json:"report_id" binding:"omitempty"`
	ActorID      *uuid.UUID `json:"actor_id"`
	Source       string     `json:"source" binding:"required,oneof=novedad report"`
	Type         string     `json:"type" binding:"required"`
	Priority     int        `json:"priority" binding:"min=0,max=100"`
	Instructions string     `json:"instructions"`
	Description  string     `json:"description"`
	Latitude     *float64   `json:"latitude"`
	Longitude    *float64   `json:"longitude"`
}

// UpdateTaskStatusRequest represents a request to update task status
type UpdateTaskStatusRequest struct {
	State       string     `json:"state" binding:"required,oneof=PENDIENTE EN_PROGRESO COMPLETADA CANCELADA"`
	Reason      *string    `json:"reason,omitempty"`
	Evidence    []string   `json:"evidence,omitempty"` // Photo URLs
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TaskResponse represents a task in response
type TaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	NovedadID   *uuid.UUID `json:"novedad_id,omitempty"`
	ReportID    *uuid.UUID `json:"report_id,omitempty"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	Source      string     `json:"source"`
	Type        string     `json:"type"`
	State       string     `json:"state"`
	Priority    int        `json:"priority"`
	Description string     `json:"description"`
	Latitude    *float64   `json:"latitude,omitempty"`
	Longitude   *float64   `json:"longitude,omitempty"`
	PhotoURL    *string    `json:"photo_url,omitempty"`
	Evidence    []string   `json:"evidence,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TaskEventPayload represents the payload of a task event
type TaskEventPayload struct {
	TaskID    uuid.UUID  `json:"task_id"`
	Source    string     `json:"source"`
	Type      string     `json:"type"`
	State     string     `json:"state"`
	ActorID   *uuid.UUID `json:"actor_id,omitempty"`
	NovedadID *uuid.UUID `json:"novedad_id,omitempty"`
	ReportID  *uuid.UUID `json:"report_id,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// NovedadEvent represents an incoming event from the novedades service
type NovedadEvent struct {
	ID        uuid.UUID              `json:"id"`
	NovedadID uuid.UUID              `json:"novedad_id"`
	Type      string                 `json:"type"` // 'novedad.creada', 'novedad.verificada'
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}
