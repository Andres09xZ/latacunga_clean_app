package server

import (
	"fmt"
	"log"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/validation-service/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Import docs for swagger
	_ "github.com/Andres09xZ/latacunga_clean_app/validation-service/docs"
)

// Start starts the validation service
func Start() error {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return fmt.Errorf("DB_URL environment variable not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "tu_secreto_muy_largo_y_seguro_123456789"
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}

	incidentServiceURL := os.Getenv("INCIDENT_SERVICE_URL")
	if incidentServiceURL == "" {
		incidentServiceURL = "http://localhost:8081"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	// Connect to database
	if err := database.Connect(dbURL); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Initialize RabbitMQ
	if err := messaging.InitRabbitMQ(rabbitMQURL); err != nil {
		log.Printf("⚠️  Warning: RabbitMQ initialization failed: %v", err)
	}
	defer messaging.CloseRabbitMQ()

	// Start consuming incident creation events
	validationHandler := handlers.NewValidationHandler(incidentServiceURL)
	go func() {
		if err := messaging.ConsumeIncidentCreatedEvents(func(incident *messaging.IncidentCreatedEvent) error {
			// Automatic validation logic for MVP
			// Rules: Auto-validate certain incident types, auto-reject others
			log.Printf("📥 Received incident pending event: %s (type: %s, status: %s)", incident.ID, incident.Type, incident.Status)

			// Auto-validate "punto_acopio" and "zona_reciclaje" types
			autoValidateTypes := map[string]bool{
				"punto_acopio":   true,
				"zona_reciclaje": true,
			}

			// For now, just log - manual validation will be triggered via API
			// In future, this could trigger automatic validation
			if _, shouldAutoValidate := autoValidateTypes[incident.Type]; shouldAutoValidate {
				log.Printf("✅ Incident %s could be auto-validated (type: %s)", incident.ID, incident.Type)
			} else {
				log.Printf("ℹ️  Incident %s requires manual validation (type: %s)", incident.ID, incident.Type)
			}

			return nil
		}); err != nil {
			log.Printf("⚠️  Failed to start event consumer: %v", err)
		}
	}()

	// Setup Gin router
	router := gin.Default()

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "validation-service"})
	})

	// Validation endpoints with JWT authentication
	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuth())
	{
		validations := api.Group("/validations")
		validations.Use(middleware.RequireValidatorRole())
		{
			// Mark incident as valid
			validations.POST("/:incident_id/mark-valid", validationHandler.MarkIncidentAsValid)

			// Reject incident
			validations.POST("/:incident_id/reject", validationHandler.RejectIncident)

			// Get validation history
			validations.GET("/:incident_id", validationHandler.GetValidation)
		}
	}

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("🚀 Validation Service starting on %s", addr)
	log.Printf("📖 Swagger docs available at http://localhost:%s/swagger/index.html", port)

	if err := router.Run(addr); err != nil {
		return fmt.Errorf("server startup failed: %w", err)
	}

	return nil
}
