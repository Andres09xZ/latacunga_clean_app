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
}

// ValidateToken validates and parses a JWT token
func ValidateToken(tokenString string, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return &Claims{
		Sub:   getString(claims, "sub"),
		Role:  getString(claims, "role"),
		Email: getString(claims, "email"),
	}, nil
}

// IsOperatorRole checks if the role is operator or admin
func IsOperatorRole(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == "operador" || role == "admin" || role == "operator" || role == "despachador"
}

func getString(claims *jwt.MapClaims, key string) string {
	if val, ok := (*claims)[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
