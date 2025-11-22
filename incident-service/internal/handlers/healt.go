package handlers

import (
	"context"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/messaging"
	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status string            `json:"status"`
	Deps   map[string]string `json:"deps"`
}

// CheckHealth verifica el estado de la base de datos y RabbitMQ
// @Summary Health check endpoint
// @Description Verifica el estado de las dependencias del servicio (DB y RabbitMQ)
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health [get]
func CheckHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status: "ok",
		Deps:   make(map[string]string),
	}

	// Check Database (crítico)
	sqlDB, err := database.DB.DB()
	if err != nil {
		response.Deps["db"] = "down"
		response.Status = "degraded"
	} else if err := sqlDB.PingContext(ctx); err != nil {
		response.Deps["db"] = "down"
		response.Status = "degraded"
	} else {
		response.Deps["db"] = "up"
	}

	// Check RabbitMQ (opcional - no afecta el status general)
	if messaging.RabbitConn == nil || messaging.RabbitConn.IsClosed() {
		response.Deps["rabbitmq"] = "down"
		// No cambiar status a degraded, RabbitMQ es opcional
	} else {
		response.Deps["rabbitmq"] = "up"
	}

	// Set status code - solo 503 si la DB está down
	statusCode := 200
	if response.Status == "degraded" {
		statusCode = 503
	}

	c.JSON(statusCode, response)
}
