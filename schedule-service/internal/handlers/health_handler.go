package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type HealthHandler struct {
	rabbitConn *amqp.Connection
}

// NewHealthHandler crea un nuevo handler de salud
func NewHealthHandler(rabbitConn *amqp.Connection) *HealthHandler {
	return &HealthHandler{
		rabbitConn: rabbitConn,
	}
}

// HealthResponse representa la respuesta del health check
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]ServiceInfo `json:"services"`
}

// ServiceInfo contiene información sobre el estado de un servicio
type ServiceInfo struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// RabbitMQHealth godoc
// @Summary Verifica la salud de RabbitMQ
// @Description Verifica la conexión y estado de RabbitMQ
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse "RabbitMQ está saludable"
// @Failure 503 {object} HealthResponse "RabbitMQ no está disponible"
// @Router /health/rabbitmq [get]
func (h *HealthHandler) RabbitMQHealth(c *gin.Context) {
	response := HealthResponse{
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceInfo),
	}

	rabbitStatus := h.checkRabbitMQ()
	response.Services["rabbitmq"] = rabbitStatus

	// Determinar el estado general
	if rabbitStatus.Status == "healthy" {
		response.Status = "healthy"
		c.JSON(http.StatusOK, response)
	} else {
		response.Status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// FullHealth godoc
// @Summary Verifica la salud de todos los servicios
// @Description Verifica la salud de RabbitMQ y otros componentes del sistema
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse "Todos los servicios están saludables"
// @Failure 503 {object} HealthResponse "Uno o más servicios no están disponibles"
// @Router /health [get]
func (h *HealthHandler) FullHealth(c *gin.Context) {
	response := HealthResponse{
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceInfo),
	}

	// Check RabbitMQ
	rabbitStatus := h.checkRabbitMQ()
	response.Services["rabbitmq"] = rabbitStatus

	// Check Application
	response.Services["application"] = ServiceInfo{
		Status:  "healthy",
		Message: "Application is running",
	}

	// Determinar el estado general
	allHealthy := true
	for _, service := range response.Services {
		if service.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		response.Status = "healthy"
		c.JSON(http.StatusOK, response)
	} else {
		response.Status = "degraded"
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// checkRabbitMQ verifica el estado de la conexión de RabbitMQ
func (h *HealthHandler) checkRabbitMQ() ServiceInfo {
	// Si no hay conexión configurada
	if h.rabbitConn == nil {
		rabbitURL := os.Getenv("RABBITMQ_URL")
		if rabbitURL == "" {
			return ServiceInfo{
				Status:  "not_configured",
				Message: "RabbitMQ URL not configured (RABBITMQ_URL env var is empty)",
			}
		}
		return ServiceInfo{
			Status:  "disconnected",
			Message: "RabbitMQ connection not initialized",
		}
	}

	// Verificar si la conexión está cerrada
	if h.rabbitConn.IsClosed() {
		return ServiceInfo{
			Status:  "unhealthy",
			Message: "RabbitMQ connection is closed",
		}
	}

	// Intentar abrir un canal para verificar que la conexión funciona
	ch, err := h.rabbitConn.Channel()
	if err != nil {
		return ServiceInfo{
			Status:  "unhealthy",
			Message: "Failed to open channel: " + err.Error(),
		}
	}
	defer ch.Close()

	// Si llegamos aquí, la conexión está saludable
	return ServiceInfo{
		Status:  "healthy",
		Message: "RabbitMQ is connected and operational",
	}
}
