package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ValidationStatus enum
type ValidationStatus string

const (
	StatusPendiente ValidationStatus = "pendiente"
	StatusValido    ValidationStatus = "valido"
	StatusRechazado ValidationStatus = "rechazado"
)

func (vs ValidationStatus) String() string {
	return string(vs)
}

// ValidatorKind enum
type ValidatorKind string

const (
	ValidatorManual     ValidatorKind = "manual"
	ValidatorAutomatico ValidatorKind = "automatico"
)

func (vk ValidatorKind) String() string {
	return string(vk)
}

// Validation model
type Validation struct {
	ID            uuid.UUID        `gorm:"type:uuid;primaryKey"`
	IncidentID    string           `gorm:"index;not null"`
	Status        ValidationStatus `gorm:"type:text;check:status IN ('pendiente','valido','rechazado');not null"`
	Reason        *string          // Null si está válido
	ValidatorKind ValidatorKind    `gorm:"type:text;check:validator_kind IN ('manual','automatico');default:'manual'"`
	RequestedAt   time.Time        `gorm:"autoCreateTime:milli;not null"`
	DecidedAt     *time.Time
	CreatedAt     time.Time `gorm:"autoCreateTime:milli"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime:milli"`
}

// TableName specifies the table name
func (Validation) TableName() string {
	return "validacion.validations"
}

// IdempotencyKey for preventing duplicates
type IdempotencyKey struct {
	Key        string    `gorm:"primaryKey;type:text"`
	IncidentID string    `gorm:"index;not null"`
	Action     string    `gorm:"type:text;not null"` // "mark_valid", "reject"
	CreatedAt  time.Time `gorm:"autoCreateTime:milli;not null"`
}

// TableName specifies the table name
func (IdempotencyKey) TableName() string {
	return "validacion.idempotency_keys"
}

// OutboxEvent for event publishing
type OutboxEvent struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	AggregateType string    `gorm:"type:text;not null"`
	AggregateID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Type          string    `gorm:"type:text;not null"` // "incidencia_validada", "incidencia_rechazada"
	Payload       []byte    `gorm:"type:jsonb;not null"`
	Status        string    `gorm:"type:text;check:status IN ('pending','published','failed');default:'pending';index"`
	CreatedAt     time.Time `gorm:"autoCreateTime:milli;not null;index"`
	PublishedAt   *time.Time
}

// TableName specifies the table name
func (OutboxEvent) TableName() string {
	return "validacion.outbox_events"
}

// ===== DTOs =====

// CreateValidationRequest - Manual validation
type CreateValidationRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// RejectValidationRequest - Rejection with reason
type RejectValidationRequest struct {
	Reason         string `json:"reason" binding:"required"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// ValidationResponse - Return validation data
type ValidationResponse struct {
	ID             uuid.UUID  `json:"id"`
	IncidentID     string     `json:"incident_id"`
	Status         string     `json:"status"`
	Reason         *string    `json:"reason,omitempty"`
	ValidatorKind  string     `json:"validator_kind"`
	RequestedAt    time.Time  `json:"requested_at"`
	DecidedAt      *time.Time `json:"decided_at,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
}

// ValidationEventPayload - Payload for RabbitMQ
type ValidationEventPayload struct {
	ValidationID   uuid.UUID `json:"validation_id"`
	IncidentID     string    `json:"incident_id"`
	Status         string    `json:"status"`
	Reason         *string   `json:"reason,omitempty"`
	ValidatorKind  string    `json:"validator_kind"`
	EventTimestamp time.Time `json:"event_timestamp"`
}

// BeforeCreate hook for UUID
func (v *Validation) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for OutboxEvent UUID
func (oe *OutboxEvent) BeforeCreate(tx *gorm.DB) error {
	if oe.ID == uuid.Nil {
		oe.ID = uuid.New()
	}
	return nil
}

// Scan implements Scanner interface for ValidationStatus
func (vs *ValidationStatus) Scan(value interface{}) error {
	*vs = ValidationStatus(value.(string))
	return nil
}

// Value implements Valuer interface for ValidationStatus
func (vs ValidationStatus) Value() (driver.Value, error) {
	return vs.String(), nil
}

// Scan implements Scanner interface for ValidatorKind
func (vk *ValidatorKind) Scan(value interface{}) error {
	*vk = ValidatorKind(value.(string))
	return nil
}

// Value implements Valuer interface for ValidatorKind
func (vk ValidatorKind) Value() (driver.Value, error) {
	return vk.String(), nil
}

// MarshalJSON for ValidationStatus
func (vs ValidationStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(vs.String())
}

// MarshalJSON for ValidatorKind
func (vk ValidatorKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(vk.String())
}
