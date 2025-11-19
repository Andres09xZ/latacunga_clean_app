package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/auth"
	"github.com/gin-gonic/gin"
)

// JWTAuth middleware validates JWT token
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("❌ Missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			log.Println("❌ Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(token, jwtSecret)
		if err != nil {
			log.Printf("❌ Token validation failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("user_id", claims.Sub)
		c.Set("user_role", claims.Role)
		c.Set("user_email", claims.Email)

		c.Next()
	}
}

// RequireOperatorRole middleware requires user to have operator role
func RequireOperatorRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			log.Println("❌ User role not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user role not found"})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			log.Println("❌ User role is not a string")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid role type"})
			c.Abort()
			return
		}

		if !auth.IsOperatorRole(roleStr) {
			log.Printf("❌ User role %s is not an operator", roleStr)
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}
