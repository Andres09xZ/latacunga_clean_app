package auth

import (
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret            = []byte(os.Getenv("JWT_SECRET"))
	accessExpiration, _  = strconv.Atoi(os.Getenv("JWT_EXPIRATION_HOURS"))
	refreshExpiration, _ = strconv.Atoi(os.Getenv("REFRESH_EXPIRATION_HOURS"))
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Helper to compute expiry times
func AccessExpiry() time.Time {
	if accessExpiration <= 0 {
		accessExpiration = 1 // default 1 hour
	}
	return time.Now().Add(time.Duration(accessExpiration) * time.Hour)
}

func RefreshExpiry() time.Time {
	if refreshExpiration <= 0 {
		refreshExpiration = 24 * 7 // default 7 days (in hours)
	}
	return time.Now().Add(time.Duration(refreshExpiration) * time.Hour)
}

// GenerateTokens crea access y refresh tokens para un usuario
func GenerateTokens(userID, email, role string) (string, string, error) {
	// Access token
	accessClaims := Claims{
		UserID: userID,
		Email:  email,
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
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// Verificar método de firma
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenUnverifiable
		}
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
