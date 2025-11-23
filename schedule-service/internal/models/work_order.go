package models

import "time"

// WorkOrder persisted record created when threshold reached or manual trigger
type WorkOrder struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ZoneID      uint      `json:"zone_id"`
	ZoneName    string    `json:"zone_name"`
	GeneratedAt time.Time `json:"generated_at"`
	TotalPoints int       `json:"total_points"`
	Threshold   int       `json:"threshold"`
}

func (WorkOrder) TableName() string { return "work_orders" }
