package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/Andres09xZ/latacunga_clean_app/internal/auth"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "bearer"
	ErrorMissingAuth    = "missing authorization header"
	ErrorInvalidAuth    = "invalid authorization header"
	ErrorInvalidaToken  = "invalid token"
	ErrorInvalidClaims  = "invalid token claims"
	ErrorMissingUserID  = "user_id missing in token"
)

func JWTAuth() gin.HandlerFunc {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is not set")
	}

	return func(c *gin.Context) {
		tokenStr, err := extractTokenFromHeader(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrorInvalidaToken})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func extractTokenFromHeader(c *gin.Context) (string, error) {
	auth := c.GetHeader(AuthorizationHeader)
	if auth == "" {
		return "", http.ErrNoLocation
	}

	parts := strings.Fields(auth)
	if len(parts) != 2 || strings.ToLower(parts[0]) != BearerPrefix {
		return "", http.ErrNoLocation
	}

	return parts[1], nil
}
