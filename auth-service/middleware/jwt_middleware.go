package middleware

import (
	"net/http"
	"strings"

	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/auth"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "bearer"
	ErrorMissingAuth    = "missing authorization header"
	ErrorInvalidAuth    = "invalid authorization header"
	ErrorInvalidToken   = "invalid token"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := extractTokenFromHeader(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrorMissingAuth})
			return
		}

		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrorInvalidToken})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func extractTokenFromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return "", http.ErrNoLocation
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != BearerPrefix {
		return "", http.ErrNoLocation
	}

	return parts[1], nil
}
