package handlers

import "github.com/gin-gonic/gin"

// HealthHandler responde con OK; implementación mínima para pruebas.
func HealthHandler(c *gin.Context) {
    c.String(200, "ok")
}

// HelloHandler devuelve un mensaje simple para comprobaciones.
func HelloHandler(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Hello, World!"})
}
