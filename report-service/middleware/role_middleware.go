package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireRole(allowed ...string) gin.HandlerFunc {
	// Se inicializa un mapa para admitir los roles
	allowedMap := map[string]bool{}
	// Se itera en los roles enviados a la funcion y se coloca un true en el mapa
	for _, r := range allowed {
		allowedMap[strings.ToLower(r)] = true
	}

	// Se retorna la funcion middleware
	return func(c *gin.Context) {
		// Se valida que en la solicitud del token exista el rol
		v, ok := c.Get("role")
		if !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		role := strings.ToLower(v.(string))
		if !allowedMap[role] {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
