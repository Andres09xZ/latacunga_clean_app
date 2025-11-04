package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "bearer"
	ErrorMissingAuth    = "missing authorization header"
	ErrorInvalidAuth    = "invalid authorization header"
	ErrorInvalidToken   = "invalid token"
	ErrorAuthService    = "auth service error"
)

type validateResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

func JWTAuth() gin.HandlerFunc {
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		panic("AUTH_SERVICE_URL environment variable is not set")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	return func(c *gin.Context) {
		tokenStr, err := extractTokenFromHeader(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Call auth-service to validate token
		req, err := http.NewRequest("POST", authServiceURL+"/api/v1/auth/validate-token", nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": ErrorAuthService})
			return
		}
		req.Header.Set("Authorization", "Bearer "+tokenStr)

		resp, err := client.Do(req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": ErrorAuthService})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrorInvalidToken})
			return
		}

		var valResp validateResponse
		if err := json.NewDecoder(resp.Body).Decode(&valResp); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": ErrorAuthService})
			return
		}

		c.Set("user_id", valResp.UserID)
		c.Set("role", valResp.Role)
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
