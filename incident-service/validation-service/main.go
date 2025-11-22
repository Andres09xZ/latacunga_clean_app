package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/Andres09xZ/latacunga_clean_app/validation-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Validation Service API
// @version 1.0
// @description API para validación manual de incidentes por administradores (con BD propia)
// @host localhost:8082
// @BasePath /

// IncomingIncident representa un incidente recibido de RabbitMQ
type IncomingIncident struct {
	IncidentID     string `json:"incident_id"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	ReporterID     string `json:"reporter_id"`
	Status         string `json:"status"`
	EventTimestamp string `json:"event_timestamp"`
	IncidentDay    string `json:"incident_day"`
	PhotosCount    int    `json:"photos_count"`
	Address        string `json:"address"`
	Location       struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
}

// ValidationResult es el payload de validación
type ValidationResult struct {
	IncidentID  string `json:"incident_id"`
	Status      string `json:"status"`
	Validator   string `json:"validator"`
	ValidatedAt string `json:"validated_at"`
	Notes       string `json:"notes,omitempty"`
}

// ValidateRequest es el payload para validar un incidente
type ValidateRequest struct {
	Status string `json:"status" binding:"required" example:"incidente_valido"` // incidente_valido o incidente_rechazado
	Notes  string `json:"notes" example:"Incidente verificado por administrador"`
}

var (
	rabbitChannel *amqp.Channel
	exchangeName  = "city.cleaning.incidents"
)

func main() {
	// Cargar .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found, using environment variables")
	}

	// Configuración
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL environment variable is required")
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://tesis:tesis@localhost:5672/"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Println("🚀 Validation Service (Decoupled with own DB)")
	log.Printf("📡 Connecting to database...")

	// Conectar a la base de datos
	if err := database.Connect(dbURL); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Printf("📡 Connecting to RabbitMQ: %s", rabbitURL)

	// Conectar a RabbitMQ
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	rabbitChannel, err = conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer rabbitChannel.Close()

	if err := rabbitChannel.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	// Iniciar consumidor de RabbitMQ en background
	go startRabbitMQConsumer(rabbitChannel)

	// Configurar Gin
	r := gin.Default()

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", healthCheck)

	// API v1
	v1 := r.Group("/api/v1")
	{
		v1.GET("/incidents/pending", getPendingIncidents)
		v1.POST("/incidents/:id/validate", validateIncident)
	}

	log.Printf("✅ Validation service running on port %s", port)
	log.Println("📚 Swagger: http://localhost:" + port + "/swagger/index.html")
	log.Println("🎧 RabbitMQ consumer running - storing incidents in local DB")

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// healthCheck verifica el estado del servicio
// @Summary Health check
// @Description Verifica el estado del servicio y la base de datos
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthCheck(c *gin.Context) {
	sqlDB, err := database.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":   "error",
			"database": "disconnected",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":   "error",
			"database": "unreachable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"service":  "validation-service",
		"database": "connected",
	})
}

// startRabbitMQConsumer escucha eventos y guarda en BD local
func startRabbitMQConsumer(ch *amqp.Channel) {
	q, err := ch.QueueDeclare(
		"validation.service.queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	routingKey := "incidents.submitted.v1"
	if err := ch.QueueBind(q.Name, routingKey, exchangeName, false, nil); err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Println("🎧 Listening for incidents.submitted.v1 messages...")

	for d := range msgs {
		var inc IncomingIncident
		if err := json.Unmarshal(d.Body, &inc); err != nil {
			log.Printf("❌ Failed to unmarshal: %v", err)
			d.Ack(false)
			continue
		}

		// Guardar en base de datos local
		if err := saveIncidentToDB(&inc); err != nil {
			log.Printf("❌ Failed to save incident to DB: %v", err)
			d.Nack(false, true)
			continue
		}

		log.Printf("💾 Incidente guardado en BD: ID=%s Type=%s - Esperando validación manual", inc.IncidentID, inc.Type)
		d.Ack(false)
	}
}

// saveIncidentToDB guarda el incidente en la BD local
func saveIncidentToDB(inc *IncomingIncident) error {
	incidenteID, err := uuid.Parse(inc.IncidentID)
	if err != nil {
		return err
	}

	ciudadanoID, err := uuid.Parse(inc.ReporterID)
	if err != nil {
		return err
	}

	fechaEvento, err := time.Parse(time.RFC3339, inc.EventTimestamp)
	if err != nil {
		return err
	}

	// Intentar parsear incident_day - puede venir como RFC3339 o solo fecha
	var diaIncidente time.Time
	diaIncidente, err = time.Parse("2006-01-02", inc.IncidentDay)
	if err != nil {
		// Si falla, intentar con RFC3339
		diaIncidente, err = time.Parse(time.RFC3339, inc.IncidentDay)
		if err != nil {
			return err
		}
	}

	incidente := models.IncidentePendiente{
		IncidenteID:  incidenteID,
		Tipo:         inc.Type,
		Descripcion:  inc.Description,
		CiudadanoID:  ciudadanoID,
		Estado:       "pendiente_validacion",
		FechaEvento:  fechaEvento,
		DiaIncidente: diaIncidente,
		NumFotos:     inc.PhotosCount,
		Direccion:    inc.Address,
		Latitud:      inc.Location.Latitude,
		Longitud:     inc.Location.Longitude,
	}

	result := database.DB.Create(&incidente)
	return result.Error
}

// getPendingIncidents obtiene incidentes pendientes de la BD local
// @Summary Listar incidentes pendientes
// @Description Obtiene todos los incidentes con estado 'pendiente_validacion' de la BD local
// @Tags Validation
// @Produce json
// @Success 200 {object} map[string]interface{} "Lista de incidentes pendientes"
// @Failure 500 {object} map[string]string "Error al obtener incidentes"
// @Router /api/v1/incidents/pending [get]
func getPendingIncidents(c *gin.Context) {
	var incidentes []models.IncidentePendiente

	result := database.DB.Where("estado = ?", "pendiente_validacion").
		Order("fecha_evento DESC").
		Find(&incidentes)

	if result.Error != nil {
		log.Printf("❌ Database error: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch incidents"})
		return
	}

	// Convertir a respuestas
	var responses []models.IncidentePendienteResponse
	for _, inc := range incidentes {
		responses = append(responses, inc.ToResponse())
	}

	log.Printf("✅ Found %d pending incidents", len(responses))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(responses),
		"data":    responses,
	})
}

// validateIncident valida manualmente un incidente
// @Summary Validar incidente
// @Description Valida o rechaza un incidente manualmente (solo administradores)
// @Tags Validation
// @Accept json
// @Produce json
// @Param id path string true "ID del incidente"
// @Param request body ValidateRequest true "Estado de validación"
// @Success 200 {object} map[string]interface{} "Incidente validado exitosamente"
// @Failure 400 {object} map[string]string "Solicitud inválida"
// @Failure 404 {object} map[string]string "Incidente no encontrado"
// @Failure 500 {object} map[string]string "Error al validar"
// @Router /api/v1/incidents/{id}/validate [post]
func validateIncident(c *gin.Context) {
	incidentID := c.Param("id")

	incidenteID, err := uuid.Parse(incidentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID"})
		return
	}

	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validar que el status sea válido
	if req.Status != "incidente_valido" && req.Status != "incidente_rechazado" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be 'incidente_valido' or 'incidente_rechazado'"})
		return
	}

	// Buscar incidente en BD local
	var incidente models.IncidentePendiente
	result := database.DB.Where("incidente_id = ? AND estado = ?", incidenteID, "pendiente_validacion").First(&incidente)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found or already validated"})
		return
	}

	log.Printf("👨‍💼 Admin validating incident: ID=%s Status=%s", incidentID, req.Status)

	// Actualizar en BD local
	now := time.Now()
	validator := "admin-manual"
	estado := "validado"
	if req.Status == "incidente_rechazado" {
		estado = "rechazado"
	}

	updates := map[string]interface{}{
		"estado":           estado,
		"validado_por":     validator,
		"fecha_validacion": now,
		"notas_validacion": req.Notes,
	}

	if err := database.DB.Model(&incidente).Updates(updates).Error; err != nil {
		log.Printf("❌ Failed to update incident in DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update incident"})
		return
	}

	// Publicar resultado de validación a RabbitMQ
	result2 := ValidationResult{
		IncidentID:  incidentID,
		Status:      req.Status,
		Validator:   validator,
		ValidatedAt: now.UTC().Format(time.RFC3339),
		Notes:       req.Notes,
	}

	if err := publishValidationResult(rabbitChannel, result2); err != nil {
		log.Printf("❌ Failed to publish validation event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish validation event"})
		return
	}

	log.Printf("✅ Validación completada: incident=%s status=%s", incidentID, req.Status)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Incident validated successfully",
		"incident_id":  incidentID,
		"status":       req.Status,
		"validated_at": result2.ValidatedAt,
		"notes":        req.Notes,
	})
}

// publishValidationResult publica el resultado de validación a RabbitMQ
func publishValidationResult(ch *amqp.Channel, result ValidationResult) error {
	payloadBytes, _ := json.Marshal(result)

	log.Printf("📤 Publishing validation result: %s", string(payloadBytes))

	return ch.Publish(
		exchangeName,
		"incidents.validated.v1",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payloadBytes,
			Timestamp:   time.Now(),
		},
	)
}
