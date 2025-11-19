package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RouteStatus represents the status of a route
type RouteStatus string

const (
	RouteStatusPending    RouteStatus = "pendiente"
	RouteStatusGenerating RouteStatus = "generando"
	RouteStatusGenerated  RouteStatus = "generada"
	RouteStatusFailed     RouteStatus = "fallida"
	RouteStatusHeuristic  RouteStatus = "heuristica"
)

// Route represents an optimized route for an operator
type Route struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OperatorID     uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex:idx_operator_date" json:"operator_id"`
	WorkDate       time.Time   `gorm:"type:date;not null;uniqueIndex:idx_operator_date" json:"work_date"`
	ShiftID        *uuid.UUID  `gorm:"type:uuid" json:"shift_id,omitempty"`
	Status         RouteStatus `gorm:"type:text;default:'generada'" json:"status"`
	TotalDistanceM *int64      `gorm:"type:bigint" json:"total_distance_m,omitempty"`
	TotalDurationS *int64      `gorm:"type:bigint" json:"total_duration_s,omitempty"`
	Polyline       *string     `gorm:"type:text" json:"polyline,omitempty"`
	CreatedAt      time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
	Steps          []RouteStep `gorm:"foreignKey:RouteID;constraint:OnDelete:CASCADE" json:"steps,omitempty"`
}

// RouteStep represents a stop/task in a route
type RouteStep struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RouteID       uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_route_seq" json:"route_id"`
	TaskID        uuid.UUID  `gorm:"type:uuid;not null" json:"task_id"`
	Seq           int        `gorm:"not null;uniqueIndex:idx_route_seq" json:"seq"`
	ArrivalETA    *time.Time `gorm:"type:timestamptz" json:"arrival_eta,omitempty"`
	DepartureTime *time.Time `gorm:"type:timestamptz" json:"departure_time,omitempty"`
	LegDistanceM  *int64     `gorm:"type:bigint" json:"leg_distance_m,omitempty"`
	LegDurationS  *int64     `gorm:"type:bigint" json:"leg_duration_s,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	Route         *Route     `gorm:"foreignKey:RouteID" json:"-"`
}

// DistanceMatrixCache caches distance matrices
type DistanceMatrixCache struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	KeyHash   string          `gorm:"type:text;uniqueIndex" json:"key_hash"`
	Payload   json.RawMessage `gorm:"type:jsonb" json:"payload"`
	TTLUntil  time.Time       `gorm:"type:timestamptz" json:"ttl_until"`
	CreatedAt time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

// OutboxEvent represents an event to be published
type OutboxEvent struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AggregateType string          `gorm:"type:text;not null" json:"aggregate_type"`
	AggregateID   uuid.UUID       `gorm:"type:uuid;not null" json:"aggregate_id"`
	Type          string          `gorm:"type:text;not null" json:"type"`
	Payload       json.RawMessage `gorm:"type:jsonb" json:"payload"`
	Status        string          `gorm:"type:text;default:'pending'" json:"status"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
	PublishedAt   *time.Time      `gorm:"type:timestamptz" json:"published_at,omitempty"`
}

// --- DTOs ---

// CreateRouteRequest for requesting route generation
type CreateRouteRequest struct {
	OperatorID uuid.UUID `json:"operator_id" binding:"required"`
	WorkDate   time.Time `json:"work_date" binding:"required"`
}

// RouteResponse for returning route data
type RouteResponse struct {
	ID             uuid.UUID           `json:"id"`
	OperatorID     uuid.UUID           `json:"operator_id"`
	WorkDate       time.Time           `json:"work_date"`
	ShiftID        *uuid.UUID          `json:"shift_id,omitempty"`
	Status         string              `json:"status"`
	TotalDistanceM *int64              `json:"total_distance_m,omitempty"`
	TotalDurationS *int64              `json:"total_duration_s,omitempty"`
	Polyline       *string             `json:"polyline,omitempty"`
	Steps          []RouteStepResponse `json:"steps,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// RouteStepResponse for returning route step data
type RouteStepResponse struct {
	ID            uuid.UUID  `json:"id"`
	TaskID        uuid.UUID  `json:"task_id"`
	Seq           int        `json:"seq"`
	ArrivalETA    *time.Time `json:"arrival_eta,omitempty"`
	DepartureTime *time.Time `json:"departure_time,omitempty"`
	LegDistanceM  *int64     `json:"leg_distance_m,omitempty"`
	LegDurationS  *int64     `json:"leg_duration_s,omitempty"`
}

// Task represents a task to be included in a route
type Task struct {
	ID          uuid.UUID  `json:"id"`
	OperatorID  uuid.UUID  `json:"operator_id"`
	Date        time.Time  `json:"date"`
	Location    string     `json:"location"`
	Latitude    float64    `json:"latitude"`
	Longitude   float64    `json:"longitude"`
	Duration    int        `json:"duration"` // in seconds
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	WindowStart *time.Time `json:"window_start,omitempty"` // time window constraint
	WindowEnd   *time.Time `json:"window_end,omitempty"`
}

// RouteCalculatedEvent published when route is calculated
type RouteCalculatedEvent struct {
	RouteID    uuid.UUID           `json:"route_id"`
	OperatorID uuid.UUID           `json:"operator_id"`
	WorkDate   time.Time           `json:"work_date"`
	Status     string              `json:"status"`
	Steps      []RouteStepResponse `json:"steps"`
	Distance   *int64              `json:"total_distance_m,omitempty"`
	Duration   *int64              `json:"total_duration_s,omitempty"`
	Timestamp  time.Time           `json:"timestamp"`
}

// Scan implements sql.Scanner interface
func (r *Route) Scan(value interface{}) error {
	bytes, _ := value.([]byte)
	return json.Unmarshal(bytes, &r)
}

// Value implements driver.Valuer interface
func (r Route) Value() (driver.Value, error) {
	return json.Marshal(r)
}
