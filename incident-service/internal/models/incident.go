package models

import (
	"encoding/json"
	"time"
)

// IncidentType representa el tipo de incidente
type IncidentType string

const (
	IncidentTypePuntoAcopio   IncidentType = "punto_acopio"
	IncidentTypeZonaCritica   IncidentType = "zona_critica"
	IncidentTypeAnimalMuerto  IncidentType = "animal_muerto"
	IncidentTypeZonaReciclaje IncidentType = "zona_reciclaje"
)

// IncidentStatus representa el estado del incidente
type IncidentStatus string

const (
	IncidentStatusNoValidado IncidentStatus = "incidente_no_validado"
	IncidentStatusValido     IncidentStatus = "incidente_valido"
	IncidentStatusRechazado  IncidentStatus = "incidente_rechazado"
)

// GeoPoint representa un punto geográfico simple (para parsing)
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Incident representa un incidente reportado
type Incident struct {
	ID           string               `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ReporterKind string               `gorm:"type:text;not null;check:reporter_kind IN ('ciudadano','operador')" json:"reporter_kind"`
	ReporterID   *string              `gorm:"type:uuid" json:"reporter_id,omitempty"`
	Type         IncidentType         `gorm:"type:text;not null" json:"type"`
	Description  *string              `gorm:"type:text" json:"description,omitempty"`
	Location     string               `gorm:"type:geography(Point,4326);not null" json:"location"`
	Address      *string              `gorm:"type:text" json:"address,omitempty"`
	Status       IncidentStatus       `gorm:"type:text;not null;default:'emitido'" json:"status"`
	IncidentDay  time.Time            `gorm:"type:date;not null;default:((now() AT TIME ZONE 'UTC')::date)" json:"incident_day"`
	PhotosCount  int                  `gorm:"default:0" json:"photos_count"`
	Attachments  []IncidentAttachment `gorm:"foreignKey:IncidentID;constraint:OnDelete:CASCADE" json:"attachments,omitempty"`
	Events       []IncidentEvent      `gorm:"foreignKey:IncidentID;constraint:OnDelete:CASCADE" json:"events,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// TableName especifica el nombre de la tabla
func (Incident) TableName() string {
	return "incidentes.incidents"
}

// IncidentAttachment representa un adjunto (foto/archivo) del incidente
type IncidentAttachment struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	IncidentID string    `gorm:"type:uuid;not null" json:"incident_id"`
	FileURL    string    `gorm:"type:text;not null" json:"file_url"`
	MimeType   *string   `gorm:"type:text" json:"mime_type,omitempty"`
	SizeBytes  *int64    `gorm:"type:bigint" json:"size_bytes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName especifica el nombre de la tabla
func (IncidentAttachment) TableName() string {
	return "incidentes.incident_attachments"
}

// IncidentEvent representa un evento en el histórico del incidente
type IncidentEvent struct {
	ID         string          `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	IncidentID string          `gorm:"type:uuid;not null" json:"incident_id"`
	EventType  string          `gorm:"type:text;not null" json:"event_type"`
	Payload    json.RawMessage `gorm:"type:jsonb" json:"payload,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// TableName especifica el nombre de la tabla
func (IncidentEvent) TableName() string {
	return "incidentes.incident_events"
}

// IdempotencyKey representa una clave de idempotencia para operación offline-first
type IdempotencyKey struct {
	Key        string    `gorm:"type:text;primaryKey" json:"key"`
	ResourceID *string   `gorm:"type:uuid" json:"resource_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName especifica el nombre de la tabla
func (IdempotencyKey) TableName() string {
	return "incidentes.idempotency_keys"
}

// OutboxEvent representa un evento pendiente de publicar
type OutboxEvent struct {
	ID            string          `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AggregateType string          `gorm:"type:text;not null" json:"aggregate_type"`
	AggregateID   string          `gorm:"type:uuid;not null" json:"aggregate_id"`
	Type          string          `gorm:"type:text;not null" json:"type"`
	Payload       json.RawMessage `gorm:"type:jsonb;not null" json:"payload"`
	Status        string          `gorm:"type:text;not null;default:'pending'" json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	PublishedAt   *time.Time      `json:"published_at,omitempty"`
}

// TableName especifica el nombre de la tabla
func (OutboxEvent) TableName() string {
	return "incidentes.outbox_events"
}

// === DTOs (Data Transfer Objects) ===

// CreateIncidentRequest es el DTO para crear un incidente
type CreateIncidentRequest struct {
	IdempotencyKey string   `json:"idempotency_key,omitempty"` // Para offline-first
	Type           string   `json:"type" binding:"required"`
	Title          string   `json:"title" binding:"required"`
	Description    *string  `json:"description,omitempty"`
	Latitude       *float64 `json:"latitude" binding:"required"`
	Longitude      *float64 `json:"longitude" binding:"required"`
	Address        *string  `json:"address,omitempty"`
	PhotoURL       *string  `json:"photo_url,omitempty"` // Evidencia inicial (foto)
}

// UpdateIncidentStatusRequest es el DTO para actualizar el estado
type UpdateIncidentStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=emitido valido rechazado convertido_en_tarea cerrado"`
}

// AddIncidentAttachmentRequest es el DTO para agregar un adjunto
type AddIncidentAttachmentRequest struct {
	FileURL   string `json:"file_url" binding:"required"`
	MimeType  string `json:"mime_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

// LocationResponse es el DTO para la ubicación en la respuesta
type LocationResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// IncidentResponse es el DTO de respuesta de un incidente
type IncidentResponse struct {
	ID             string                       `json:"id"`
	IdempotencyKey string                       `json:"idempotency_key,omitempty"` // Solo si se proporcionó
	ReporterKind   string                       `json:"reporter_kind"`
	ReporterID     *string                      `json:"reporter_id,omitempty"`
	Type           string                       `json:"type"`
	Description    *string                      `json:"description,omitempty"`
	Location       LocationResponse             `json:"location"`
	Address        *string                      `json:"address,omitempty"`
	Status         string                       `json:"status"`
	IncidentDay    time.Time                    `json:"incident_day"`
	PhotosCount    int                          `json:"photos_count"`
	Attachments    []IncidentAttachmentResponse `json:"attachments,omitempty"`
	CreatedAt      time.Time                    `json:"created_at"`
	UpdatedAt      time.Time                    `json:"updated_at"`
}

// IncidentAttachmentResponse es el DTO de respuesta de un adjunto
type IncidentAttachmentResponse struct {
	ID        string    `json:"id"`
	FileURL   string    `json:"file_url"`
	MimeType  *string   `json:"mime_type,omitempty"`
	SizeBytes *int64    `json:"size_bytes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// IncidentEventResponse es el DTO de respuesta de un evento
type IncidentEventResponse struct {
	ID        string          `json:"id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// ListIncidentsResponse es el DTO de respuesta para listar incidentes
type ListIncidentsResponse struct {
	Incidents  []IncidentResponse `json:"incidents"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// IncidentErrorResponse es el DTO de respuesta de error
type IncidentErrorResponse struct {
	Error string `json:"error"`
}
