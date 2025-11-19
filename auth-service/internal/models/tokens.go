package models

import (
	"time"
)

// RefreshToken represents a refresh token stored in the database
type RefreshToken struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;size:512;not null" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `gorm:"default:false" json:"revoked"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
