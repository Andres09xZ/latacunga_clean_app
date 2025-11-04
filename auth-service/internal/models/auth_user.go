package models

import (
	"time"
)

// @name User
type User struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	// Make Email and PasswordHash nullable so OTP-only users can have NULL values
	Email           *string   `gorm:"uniqueIndex;size:255" json:"email,omitempty"`
	IDUser          string    `gorm:"size:100;uniqueIndex" json:"id_user"` // Identificador adicional de usuario
	PasswordHash    *string   `gorm:"size:128" json:"-"`
	Name            string    `gorm:"size:255" json:"name"`
	Role            string    `gorm:"size:50;default:user" json:"role"`
	Dni             string    `gorm:"size:20" json:"dni"`
	TelephoneNumber string    `gorm:"size:20" json:"telephone_number"`
	Birthday        time.Time `json:"birthday"` // usa time.Time para fechas

	// OTP fields for phone-based authentication
	OTPCode       string     `gorm:"size:10" json:"-"`
	OTPExpiresAt  *time.Time `json:"-"`
	PhoneVerified bool       `gorm:"default:false" json:"phone_verified"`
}
