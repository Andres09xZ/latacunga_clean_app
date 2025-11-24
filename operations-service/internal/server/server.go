package server

import (
	"log"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/messaging"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	files "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Start inicia el servidor HTTP
func Start() {
	// Inicializar base de datos
	database.InitDB()

	// Inicializar RabbitMQ
	if err := messaging.InitRabbitMQ(); err != nil {
		log.Printf("Warning: RabbitMQ initialization failed: %v", err)
	} else {
		// Iniciar el consumidor
		if err := messaging.StartConsumer(); err != nil {
			log.Printf("Warning: Failed to start RabbitMQ consumer: %v", err)
		}
	}

	r := gin.Default()

	// CORS middleware
	r.Use(cors.Default())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "operations-service"})
	})

	// API v1 routes - Driver endpoints
	driverGroup := r.Group("/api/v1/driver")
	{
		driverGroup.GET("/orders/active", handlers.GetActiveOrder)
		driverGroup.POST("/orders/:id/start", handlers.StartOrder)
		driverGroup.POST("/stops/:id/complete", handlers.CompleteStop)
		driverGroup.POST("/orders/:id/finish", handlers.FinishOrder)
	}

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(files.Handler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}
	addr := ":" + port
	log.Printf("Starting operations-service on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
