package handlers

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"

	// strings no es necesario actualmente
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Andres09xZ/latacunga_clean_app/internal/auth"
	"github.com/Andres09xZ/latacunga_clean_app/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/internal/models"
	"gorm.io/gorm"
)

type registerRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=6"`
	Name            string `json:"name"`
	Dni             string `json:"dni"`
	TelephoneNumber string `json:"telephone_number"`
	Birthday        string `json:"birthday"`                                     // RFC3339
	Role            string `json:"role" binding:"required,oneof=admin operator"` // only admin or operator allowed here
}

// Register crea un nuevo usuario (genera IDUser) y retorna datos básicos.
// @Summary Register a new user
// @Description Create a new user account. Returns the generated id_user and basic profile.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body registerRequest true "Register payload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/auth/register [post]
func Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// El role debe venir en el payload y ser 'admin' o 'operator'
	if req.Role != "admin" && req.Role != "operator" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'admin' or 'operator'"})
		return
	}
	var bday time.Time
	if req.Birthday != "" {
		// Accept RFC3339 (e.g. 2006-01-02T15:04:05Z07:00) or common short date YYYY-MM-DD
		t, err := time.Parse(time.RFC3339, req.Birthday)
		if err != nil {
			// try short date
			t2, err2 := time.Parse("2006-01-02", req.Birthday)
			if err2 != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid birthday format, use RFC3339 or YYYY-MM-DD"})
				return
			}
			bday = t2
		} else {
			bday = t
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Prepare pointers for nullable fields
	emailPtr := req.Email
	passPtr := string(hashed)

	user := models.User{
		Email:           &emailPtr,
		IDUser:          uuid.NewString(),
		PasswordHash:    &passPtr,
		Name:            req.Name,
		Role:            req.Role,
		Dni:             req.Dni,
		TelephoneNumber: req.TelephoneNumber,
		Birthday:        bday,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// emailPtr holds the original string used for the Email pointer
	c.JSON(http.StatusCreated, gin.H{
		"id_user": user.IDUser,
		"email":   emailPtr,
		"name":    user.Name,
		"role":    user.Role,
	})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login verifica credenciales y emite access/refresh tokens.
// @Summary Login user
// @Description Verify user credentials and return access and refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body loginRequest true "Login payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/auth/login [post]
func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure password hash exists for this user
	if user.PasswordHash == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Only admin and operator roles can use this email/password login route
	if user.Role != "admin" && user.Role != "operator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "email/password login is restricted to admin or operator roles"})
		return
	}

	// prepare email string for tokens (nullable)
	emailStr := ""
	if user.Email != nil {
		emailStr = *user.Email
	}

	access, refresh, err := auth.GenerateTokens(user.IDUser, emailStr, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"user": gin.H{
			"id_user": user.IDUser,
			"email":   emailStr,
			"name":    user.Name,
			"role":    user.Role,
		},
	})
}

// sendOTPRequest for sending an OTP to a phone number
type sendOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// SendOTP generates an OTP for the provided phone number and (for now) logs it.
// If the phone is not associated to a user, a new user with role "user" is created.
// @Summary Send OTP to phone
// @Description Send a numeric OTP to the provided phone number. Creates a user if none exists. Intended for citizen users.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body sendOTPRequest true "Phone payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/auth/otp/send [post]
func SendOTP(c *gin.Context) {
	var req sendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Basic normalization could be added; for now use exactly what's provided
	var user models.User
	err := database.DB.Where("telephone_number = ?", req.Phone).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// create a minimal user record for this phone with role 'user'
			user = models.User{
				IDUser: uuid.NewString(),
				// Email and PasswordHash intentionally left nil so DB stores NULL
				Name:            "",
				Role:            "user",
				TelephoneNumber: req.Phone,
			}
			if err := database.DB.Create(&user).Error; err != nil {
				log.Printf("db create user error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}

		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// If user exists, only allow OTP flow for role 'user'
		if user.Role != "user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "otp flow is only allowed for users with role 'user'"})
			return
		}
	}

	// generate 6-digit OTP
	code := generateOTPCode()
	expires := time.Now().Add(5 * time.Minute)

	user.OTPCode = code
	user.OTPExpiresAt = &expires
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save otp"})
		return
	}

	// TODO: integrate SMS provider (Twilio, AWS SNS). For now, output code in logs for dev.
	log.Printf("OTP for %s = %s (expires %s)", req.Phone, code, expires.Format(time.RFC3339))

	c.JSON(http.StatusOK, gin.H{"message": "otp_sent"})
}

// verifyOTPRequest verifies the OTP code for a phone number
type verifyOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// VerifyOTP checks the code and returns tokens for the user (role user).
// @Summary Verify OTP
// @Description Verify an OTP sent to phone and return JWT tokens for the user.
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body verifyOTPRequest true "Verify payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/auth/otp/verify [post]
func VerifyOTP(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("telephone_number = ?", req.Phone).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid phone or code"})
		return
	}

	if user.OTPCode == "" || user.OTPExpiresAt == nil || time.Now().After(*user.OTPExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "otp expired or not found"})
		return
	}

	if req.Code != user.OTPCode {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid code"})
		return
	}

	// mark phone verified and clear OTP
	user.PhoneVerified = true
	user.OTPCode = ""
	user.OTPExpiresAt = nil
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	// prepare email string for tokens (nullable)
	emailStr := ""
	if user.Email != nil {
		emailStr = *user.Email
	}

	// Issue tokens (role should already be "user")
	access, refresh, err := auth.GenerateTokens(user.IDUser, emailStr, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"user": gin.H{
			"id_user":        user.IDUser,
			"email":          emailStr,
			"name":           user.Name,
			"role":           user.Role,
			"phone_verified": user.PhoneVerified,
		},
	})
}

// generateOTPCode returns a secure random 6-digit numeric code as string.
func generateOTPCode() string {
	var code string
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			// fallback to pseudo-random digit
			return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code
}
