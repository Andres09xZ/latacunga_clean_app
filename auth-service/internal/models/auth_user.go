package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email        *string   `json:"email,omitempty" gorm:"uniqueIndex"`
	Phone        *string   `json:"phone,omitempty" gorm:"uniqueIndex"`
	PasswordHash *string   `json:"-" gorm:"size:128"`
	Role         string    `json:"role" gorm:"not null"`
	DisplayName  string    `json:"display_name"`
	Status       string    `json:"status" gorm:"default:ACTIVE"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// OTPCode represents OTP codes for phone authentication
type OTPCode struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Phone       string    `json:"phone" gorm:"not null"`
	CodeHash    string    `json:"-" gorm:"not null"`
	Attempts    int       `json:"-" gorm:"default:0"`
	MaxAttempts int       `json:"-" gorm:"default:5"`
	IssuedAt    time.Time `json:"-" gorm:"default:now()"`
	ExpiresAt   time.Time `json:"-" gorm:"not null"`
	Consumed    bool      `json:"-" gorm:"default:false"`
	Purpose     string    `json:"-" gorm:"default:'LOGIN'"`
}

// OperatorProfile represents additional information for operators
type OperatorProfile struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	User      User      `json:"-" gorm:"foreignKey:UserID"`
	BadgeID   *string   `json:"badge_id,omitempty"`
	Status    string    `json:"status" gorm:"default:ACTIVE"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
