package models

import (
	"time"
)

// =============== OPERATORS ===============

// Operator representa un operador interno del sistema (operador, despachador, admin)
type Operator struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name         string    `gorm:"type:text;not null" json:"name"`
	Email        string    `gorm:"type:citext;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:text;not null" json:"-"`
	Role         string    `gorm:"type:varchar(50);not null;check:role IN ('operador','despachador','admin')" json:"role"`
	Active       bool      `gorm:"default:true;index" json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Operator) TableName() string {
	return "usuario.operators"
}

// =============== CITIZENS ===============

// Citizen representa un ciudadano que se autentica por OTP
type Citizen struct {
	ID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PhoneE164  string     `gorm:"type:text;uniqueIndex;not null" json:"phone_e164"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (Citizen) TableName() string {
	return "usuario.citizens"
}

// =============== OTP REQUESTS ===============

// OTPRequest registra cada solicitud de OTP para auditoría y rate limiting
type OTPRequest struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PhoneE164   string     `gorm:"type:text;not null;index" json:"phone_e164"`
	Provider    string     `gorm:"type:text;default:'twilio'" json:"provider"`
	Status      string     `gorm:"type:varchar(50);not null;check:status IN ('requested','verified','failed','expired')" json:"status"`
	Attempts    int        `gorm:"default:0" json:"attempts"`
	RequestedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP;index" json:"requested_at"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	ErrorCode   *string    `json:"error_code,omitempty"`
}

func (OTPRequest) TableName() string {
	return "usuario.otp_requests"
}

// =============== IDEMPOTENCY ===============

// IdempotencyKey previene duplicados en endpoints sensibles
type IdempotencyKey struct {
	Key                string    `gorm:"type:text;primaryKey" json:"key"`
	RequestFingerprint string    `gorm:"type:text;not null" json:"request_fingerprint"`
	ResponsePayload    []byte    `gorm:"type:jsonb" json:"response_payload,omitempty"`
	CreatedAt          time.Time `gorm:"default:CURRENT_TIMESTAMP;index" json:"created_at"`
}

func (IdempotencyKey) TableName() string {
	return "usuario.idempotency_keys"
}

// =============== OUTBOX EVENTS ===============

// OutboxEvent implementa el patrón Outbox para event sourcing
type OutboxEvent struct {
	ID            string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AggregateType string     `gorm:"type:text;not null;index" json:"aggregate_type"`
	AggregateID   string     `gorm:"type:uuid;not null;index" json:"aggregate_id"`
	Type          string     `gorm:"type:text;not null;index" json:"type"`
	Payload       []byte     `gorm:"type:jsonb;not null" json:"payload"`
	Status        string     `gorm:"type:varchar(50);default:'pending';check:status IN ('pending','published','failed');index" json:"status"`
	CreatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP;index" json:"created_at"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
}

func (OutboxEvent) TableName() string {
	return "usuario.outbox_events"
}

// =============== DTOs - REQUESTS ===============

// RegisterOperatorRequest es el payload para registrar un nuevo operador
type RegisterOperatorRequest struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=operador despachador admin"`
}

// LoginOperatorRequest es el payload para login de operador
type LoginOperatorRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RequestOTPRequest es el payload para solicitar OTP
type RequestOTPRequest struct {
	PhoneE164 string `json:"phone_e164" binding:"required,regex=^\\+[1-9]\\d{7,14}$"`
}

// VerifyOTPRequest es el payload para verificar OTP
type VerifyOTPRequest struct {
	PhoneE164 string `json:"phone_e164" binding:"required,regex=^\\+[1-9]\\d{7,14}$"`
	Code      string `json:"code" binding:"required,len=6,numeric"`
}

// =============== DTOs - RESPONSES ===============

// OperatorResponse es la respuesta con datos del operador
type OperatorResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenResponse es la respuesta con JWT token
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// AuthResponse es la respuesta completa de autenticación
type AuthResponse struct {
	Token    TokenResponse     `json:"token"`
	Operator *OperatorResponse `json:"operator,omitempty"`
	Citizen  *CitizenResponse  `json:"citizen,omitempty"`
}

// CitizenResponse es la respuesta con datos del ciudadano
type CitizenResponse struct {
	ID         string     `json:"id"`
	PhoneE164  string     `json:"phone_e164"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// OTPResponse es la respuesta al solicitar OTP
type OTPResponse struct {
	Message   string `json:"message"`
	PhoneE164 string `json:"phone_e164"`
	ExpiresIn int    `json:"expires_in"` // en segundos
}

// ErrorResponse es la respuesta de error estándar
type ErrorResponse struct {
	Error       string `json:"error"`
	Message     string `json:"message,omitempty"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}
