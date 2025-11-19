package auth

import (
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "tu_secreto_muy_largo_y_seguro_123456789" // fallback
	}
	return []byte(secret)
}

func getAccessExpiration() int {
	val, _ := strconv.Atoi(os.Getenv("JWT_EXPIRATION_HOURS"))
	if val <= 0 {
		val = 1 // default 1 hour
	}
	return val
}

func getRefreshExpiration() int {
	val, _ := strconv.Atoi(os.Getenv("REFRESH_EXPIRATION_HOURS"))
	if val <= 0 {
		val = 24 * 7 // default 7 days (in hours)
	}
	return val
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Helper to compute expiry times
func AccessExpiry() time.Time {
	expiration := getAccessExpiration()
	return time.Now().Add(time.Duration(expiration) * time.Hour)
}

func RefreshExpiry() time.Time {
	expiration := getRefreshExpiration()
	return time.Now().Add(time.Duration(expiration) * time.Hour)
}

// GenerateTokens crea access y refresh tokens para un usuario
func GenerateTokens(userID, email, role string) (string, string, error) {
	jwtSecret := getJWTSecret()

	// Access token
	accessClaims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(AccessExpiry()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := at.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshClaims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(RefreshExpiry()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := rt.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// ValidateToken valida y parsea un token JWT retornando las claims
func ValidateToken(tokenStr string) (*Claims, error) {
	jwtSecret := getJWTSecret()
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))
	token, err := parser.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

// ValidateRefreshToken valida un refresh token
func ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return ValidateToken(tokenStr) // Same validation for now
}
