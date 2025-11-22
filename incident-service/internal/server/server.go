package server

import (
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/incident-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Start arranca el servidor Gin para incident-service
func Start() {
	database.Connect()

	// Inicializar RabbitMQ
	if err := messaging.InitRabbitMQ(); err != nil {
		log.Printf("Warning: RabbitMQ initialization failed: %v", err)
	}
	defer messaging.CloseRabbitMQ()

	// Iniciar consumidor de resultados de validación
	messaging.StartValidationConsumer()

	// AutoMigrate no es necesario ya que usamos migraciones SQL
	// pero lo dejamos para sincronizar el schema de GORM
	database.DB.AutoMigrate(
		&models.Incident{},
		&models.IncidentAttachment{},
		&models.IncidentEvent{},
		&models.IdempotencyKey{},
		&models.OutboxEvent{},
	)

	r := gin.Default()

	// Swagger documentation
	docs.SwaggerInfo.BasePath = ""
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check con verificación de dependencias
	r.GET("/health", handlers.CheckHealth)

	// Incident routes (offline-first, ciudadanos)
	// CreateIncident: Requiere JWT (extrae reporter_kind y reporter_id del token, solo ciudadanos)
	r.POST("/api/v1/incidents", middleware.JWTAuth(), handlers.CreateIncident)
	r.GET("/api/v1/incidents", handlers.ListIncidents)
	r.GET("/api/v1/incidents/:id", handlers.GetIncident)
	r.PUT("/api/v1/incidents/:id/status", middleware.JWTAuth(), middleware.RequireRole("operador", "admin"), handlers.UpdateIncidentStatus)
	r.POST("/api/v1/incidents/:id/attachments", middleware.JWTAuth(), handlers.AddIncidentAttachment)

	addr := ":8081"
	log.Printf("Starting incident service on %s", addr)
	log.Printf("Swagger documentation available at http://localhost:8081/swagger/index.html")
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
