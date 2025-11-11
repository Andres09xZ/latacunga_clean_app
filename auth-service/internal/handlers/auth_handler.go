package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/auth"
	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=user operador admin"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type OTPRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type OTPVerifyRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required,len=6"`
}

// Register creates a new user account (only for operador/admin)
//
//	@Summary	Register a new user (operador/admin only)
//	@Description	Create a new user account with email, password and role (only operador/admin allowed)
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		RegisterRequest	true	"Register request"
//	@Success	201		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/auth/register [post]
//
// @Tagsauth
// @Acceptjson
// @Producejson
// @ParamrequestbodyRegisterRequesttrue"Register request""
// @Success201{object}map[string]interface{}
// @Failure400{object}map[string]string
// @Failure500{object}map[string]string
// @Router/auth/register [post]
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Reject registration for "user" role - must use OTP
	if req.Role == "user" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Registro de ciudadanos solo por OTP (teléfono)"})
		return
	}

	// Check if user already exists
	var existing models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already exists"})
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user := models.User{
		Email:        &req.Email,
		PasswordHash: stringPtr(string(hashed)),
		Role:         req.Role,
		DisplayName:  req.Email, // Use email as display name for now
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// If role is operador, create operator profile
	if req.Role == "operador" {
		operatorProfile := models.OperatorProfile{
			UserID: user.ID,
		}
		if err := database.DB.Create(&operatorProfile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create operator profile"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"email": *user.Email,
		"role":  user.Role,
	})
}

// Login authenticates a user and returns tokens
//
// @Summary Login user
// @Description Authenticate user with email and password, return access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password
	if user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := auth.GenerateTokens(user.ID.String(), *user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// RequestOTP sends OTP to phone number
//
// @Summary Request OTP
// @Description Request OTP code for phone authentication (citizens only)
// @Tags otp
// @Accept json
// @Produce json
// @Param request body OTPRequest true "OTP request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /auth/otp/send [post]
func RequestOTP(c *gin.Context) {
	var req OTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate phone format (E.164)
	if !isValidPhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Formato de teléfono inválido. Use E.164"})
		return
	}

	// Check if phone belongs to non-user role
	var user models.User
	if err := database.DB.Where("phone = ?", req.Phone).First(&user).Error; err == nil {
		if user.Role != "user" {
			c.JSON(http.StatusForbidden, gin.H{"message": "Método de autenticación no permitido para este rol"})
			return
		}
	}

	// Generate 6-digit OTP
	otpCode := generateOTP()

	// Hash the code
	hashedCode, err := bcrypt.GenerateFromPassword([]byte(otpCode), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Save OTP code
	otp := models.OTPCode{
		Phone:       req.Phone,
		CodeHash:    string(hashedCode),
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		MaxAttempts: 5,
	}

	if err := database.DB.Create(&otp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save OTP"})
		return
	}

	// TODO: Send OTP via SMS (placeholder)
	fmt.Printf("OTP for %s: %s\n", req.Phone, otpCode)

	c.JSON(http.StatusOK, gin.H{"message": "OTP enviado"})
}

// VerifyOTP verifies OTP code and creates/logs in user
//
// @Summary Verify OTP
// @Description Verify OTP code and authenticate/create user
// @Tags otp
// @Accept json
// @Produce json
// @Param request body OTPVerifyRequest true "OTP verify request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Router /auth/otp/verify [post]
func VerifyOTP(c *gin.Context) {
	var req OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find latest OTP for phone
	var otp models.OTPCode
	if err := database.DB.Where("phone = ? AND consumed = false AND expires_at > ?", req.Phone, time.Now()).
		Order("issued_at DESC").First(&otp).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "OTP inválido"})
		return
	}

	// Check attempts
	if otp.Attempts >= otp.MaxAttempts {
		c.JSON(http.StatusTooManyRequests, gin.H{"message": "Límite de intentos excedido, solicite un nuevo OTP"})
		return
	}

	// Increment attempts
	otp.Attempts++
	database.DB.Save(&otp)

	// Verify code
	if bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(req.Code)) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "OTP inválido"})
		return
	}

	// Mark as consumed
	otp.Consumed = true
	database.DB.Save(&otp)

	// Find or create user
	var user models.User
	if err := database.DB.Where("phone = ?", req.Phone).First(&user).Error; err != nil {
		// Create new user
		user = models.User{
			Phone:       &req.Phone,
			Role:        "user",
			DisplayName: req.Phone, // Use phone as display name
		}
		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	}

	// Generate tokens
	accessToken, refreshToken, err := auth.GenerateTokens(user.ID.String(), *user.Phone, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func isValidPhone(phone string) bool {
	// E.164 format: +[country code][number]
	match, _ := regexp.MatchString(`^\+[1-9]\d{7,14}$`, phone)
	return match
}

func generateOTP() string {
	const digits = "0123456789"
	code := make([]byte, 6)
	rand.Read(code)
	for i := range code {
		code[i] = digits[int(code[i])%10]
	}
	return string(code)
}
