package models

import (
	"time"

	"github.com/google/uuid"
)

// ShiftStatus represents the status of a shift
type ShiftStatus string

const (
	ShiftStatusActive   ShiftStatus = "active"
	ShiftStatusInactive ShiftStatus = "inactive"
)

// Shift represents a work shift (turno)
type Shift struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Date      time.Time      `gorm:"type:date;not null;index" json:"date"`
	StartTime time.Time      `gorm:"type:time;not null" json:"start_time"`
	EndTime   time.Time      `gorm:"type:time;not null" json:"end_time"`
	MaxTasks  int            `gorm:"default:10" json:"max_tasks"`
	Status    ShiftStatus    `gorm:"type:text;default:'active'" json:"status"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	Tasks     []ScheduleTask `gorm:"foreignKey:ShiftID" json:"tasks,omitempty"`
}

// TaskStatus represents the status of a scheduled task
type TaskStatus string

const (
	TaskStatusUnscheduled TaskStatus = "no_planificada"
	TaskStatusScheduled   TaskStatus = "planificada"
	TaskStatusInProgress  TaskStatus = "en_progreso"
	TaskStatusCompleted   TaskStatus = "completada"
)

// ScheduleTask represents a task scheduled for a shift
type ScheduleTask struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID    string     `gorm:"type:text;not null;index" json:"task_id"`
	ShiftID   *uuid.UUID `gorm:"type:uuid;index" json:"shift_id"`
	Date      time.Time  `gorm:"type:date;not null;index" json:"date"`
	Status    TaskStatus `gorm:"type:text;default:'no_planificada'" json:"status"`
	TaskType  string     `gorm:"type:text" json:"task_type"`
	Priority  string     `gorm:"type:text;default:'medium'" json:"priority"`
	Location  string     `gorm:"type:text" json:"location"`
	Latitude  *float64   `gorm:"type:numeric" json:"latitude"`
	Longitude *float64   `gorm:"type:numeric" json:"longitude"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	Shift     *Shift     `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
}

// DTOs for API requests/responses

// CreateShiftRequest for creating a new shift
type CreateShiftRequest struct {
	Date      time.Time `json:"date" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	MaxTasks  int       `json:"max_tasks" binding:"required,min=1"`
}

// ShiftResponse for returning shift data
type ShiftResponse struct {
	ID        uuid.UUID `json:"id"`
	Date      time.Time `json:"date"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	MaxTasks  int       `json:"max_tasks"`
	Status    string    `json:"status"`
	TaskCount int       `json:"task_count"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskCreatedEvent represents an event from task-service
type TaskCreatedEvent struct {
	ID         string    `json:"id"`
	IncidentID string    `json:"incident_id"`
	Status     string    `json:"status"`
	TaskType   string    `json:"task_type"`
	Priority   string    `json:"priority"`
	Location   string    `json:"location"`
	Latitude   *float64  `json:"latitude"`
	Longitude  *float64  `json:"longitude"`
	ReportDay  time.Time `json:"report_day"`
	CreatedAt  time.Time `json:"created_at"`
	Timestamp  time.Time `json:"timestamp"`
}

// ScheduleTaskResponse for returning scheduled task data
type ScheduleTaskResponse struct {
	ID        uuid.UUID  `json:"id"`
	TaskID    string     `json:"task_id"`
	ShiftID   *uuid.UUID `json:"shift_id"`
	Date      time.Time  `json:"date"`
	Status    string     `json:"status"`
	TaskType  string     `json:"task_type"`
	Priority  string     `json:"priority"`
	Location  string     `json:"location"`
	Latitude  *float64   `json:"latitude"`
	Longitude *float64   `json:"longitude"`
	CreatedAt time.Time  `json:"created_at"`
}

// DailyScheduleResponse for returning daily schedule
type DailyScheduleResponse struct {
	Date   time.Time              `json:"date"`
	Shifts []ShiftResponse        `json:"shifts"`
	Tasks  []ScheduleTaskResponse `json:"tasks"`
}

// ToResponse converts Shift to ShiftResponse
func (s *Shift) ToResponse() ShiftResponse {
	return ShiftResponse{
		ID:        s.ID,
		Date:      s.Date,
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		MaxTasks:  s.MaxTasks,
		Status:    string(s.Status),
		TaskCount: len(s.Tasks),
		CreatedAt: s.CreatedAt,
	}
}

// ToResponse converts ScheduleTask to ScheduleTaskResponse
func (t *ScheduleTask) ToResponse() ScheduleTaskResponse {
	return ScheduleTaskResponse{
		ID:        t.ID,
		TaskID:    t.TaskID,
		ShiftID:   t.ShiftID,
		Date:      t.Date,
		Status:    string(t.Status),
		TaskType:  t.TaskType,
		Priority:  t.Priority,
		Location:  t.Location,
		Latitude:  t.Latitude,
		Longitude: t.Longitude,
		CreatedAt: t.CreatedAt,
	}
}
