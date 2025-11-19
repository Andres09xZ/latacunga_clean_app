package auth

import (
	"fmt"
	"log"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT token claims
type Claims struct {
	Sub         string `json:"sub"`
	Role        string `json:"role"`
	Email       string `json:"email"`
	ReporterKid string `json:"reporter_kid,omitempty"`
	jwt.RegisteredClaims
}

// ValidateToken validates JWT token and returns claims
func ValidateToken(tokenString string, jwtSecret string) (*Claims, error) {
	// Remove "Bearer " prefix if present
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	log.Printf("✅ Token validated for user: %s (role: %s)", claims.Sub, claims.Role)

	return claims, nil
}

// IsValidatorRole checks if user has validator role
func IsValidatorRole(role string) bool {
	validRoles := map[string]bool{
		"operador":  true,
		"admin":     true,
		"validador": true,
	}
	return validRoles[role]
}
