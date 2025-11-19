package main

import (
	"log"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/server"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Load environment variables
	// Try multiple paths for flexibility
	paths := []string{
		".env",
		"cmd/server/.env",
		"../../.env",
	}

	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("✅ Loaded environment from %s", path)
			break
		}
	}

	// Get configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("❌ JWT_SECRET environment variable is required")
	}

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	// Connect to database
	log.Println("🔌 Connecting to database...")
	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	// Run migrations
	log.Println("🔄 Running migrations...")
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Initialize RabbitMQ
	log.Println("🐰 Initializing RabbitMQ...")
	if err := messaging.InitRabbitMQ(rabbitmqURL); err != nil {
		log.Printf("⚠️  Warning: Failed to initialize RabbitMQ: %v (Service will run without event consumer)", err)
		// Continue anyway - service can still work without RabbitMQ
	} else {
		defer messaging.CloseRabbitMQ()
	}

	// Create server
	srv := server.NewServer(port, db)
	defer srv.Close()

	// Setup Swagger
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Host = "localhost:8083"

	// Setup routes
	if err := srv.Setup(jwtSecret); err != nil {
		log.Fatalf("❌ Failed to setup server: %v", err)
	}

	// Setup Swagger endpoint
	srv.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start event consumer
	srv.StartEventConsumer()

	// Start server
	log.Printf("✅ Schedule service starting on port %s\n", port)
	if err := srv.Start(); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}
