package auth

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT token claims
type Claims struct {
	Sub   string `json:"sub"`
	Role  string `json:"role"`
	Email string `json:"email"`
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

	return claims, nil
}

// IsOperatorRole checks if user has operator role
func IsOperatorRole(role string) bool {
	operatorRoles := map[string]bool{
		"operador": true,
		"admin":    true,
	}
	return operatorRoles[role]
}
