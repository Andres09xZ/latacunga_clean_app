package models

import "time"

// Incident persisted event contributing to zone score
type Incident struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ZoneID       *uint     `json:"zone_id"`
	ZoneName     string    `json:"zone_name"`
	IncidentType string    `json:"incident_type"`
	Points       int       `json:"points"`
	Lat          float64   `json:"lat"`
	Lon          float64   `json:"lon"`
	CreatedAt    time.Time `json:"created_at"`
	WorkOrderID  *uint     `json:"work_order_id"`
}

func (Incident) TableName() string { return "incidents" }
