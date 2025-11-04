package models

import (
	"time"
)

// @name Report
type Report struct {
	ID          string    `gorm:"type:uuid;default:uuid_generate_v4();primarykey" json:"id"`
	UserID      string    `gorm:"type:uuid;not null" json:"user_id"`
	Type        string    `gorm:"type:varchar(20);not null;check:type IN ('acopio','critico')" json:"type"`
	Location    string    `gorm:"type:geography(POINT,4326)" json:"location,omitempty"` // WKT format, e.g., "POINT(-122.4194 37.7749)"
	PhotoURL    string    `gorm:"type:text" json:"photo_url,omitempty"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Status      string    `gorm:"type:varchar(20);default:'Pendiente';check:status IN ('Pendiente','Procesado','Aprobado','Rechazado')" json:"status"`
	Synced      bool      `gorm:"default:false" json:"synced"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
