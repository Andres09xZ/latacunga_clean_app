package server

import (
	"net/http"
	"github.com/gin-gonic/gin"
)


func Start() {
	auth_server := gin.Default()

	auth_server.GET("/health", func(c *gin.Context){
		c.JSON(http.StatusOK, gin.H{"status": "Servidor de autenticacion de usuarios levantado"})
	})

	auth_server.GET("/hello-world", func(c *gin.Context){
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	auth_server.Run(":8080") // Escucha en el puerto 8080
}

