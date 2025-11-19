package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"

	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/cache"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/optimizer"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/server"
)

// @title Route Service API
// @version 1.0
// @description API for route optimization and management
// @host localhost:8084
// @basePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load .env file
	envPaths := []string{
		".env",
		"cmd/server/.env",
		filepath.Join(os.Getenv("HOME"), ".env"),
	}

	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			err := godotenv.Load(path)
			if err != nil {
				log.Printf("⚠️  Failed to load .env from %s: %v", path, err)
			} else {
				log.Printf("✅ Loaded .env from %s", path)
				break
			}
		}
	}

	// Get environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost user=postgres password=postgres dbname=rutas port=5432 sslmode=disable"
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key"
	}

	solverTimeoutStr := os.Getenv("VRP_SOLVER_TIMEOUT_SECONDS")
	solverTimeout := 30
	if solverTimeoutStr != "" {
		if timeout, err := strconv.Atoi(solverTimeoutStr); err == nil {
			solverTimeout = timeout
		}
	}

	cacheTTLStr := os.Getenv("DISTANCE_MATRIX_CACHE_TTL_MINUTES")
	cacheTTL := 60
	if cacheTTLStr != "" {
		if ttl, err := strconv.Atoi(cacheTTLStr); err == nil {
			cacheTTL = ttl
		}
	}

	log.Printf("🔧 Configuration loaded:")
	log.Printf("   PORT: %s", port)
	log.Printf("   DATABASE_URL: %s", dbURL)
	log.Printf("   RABBITMQ_URL: %s", rabbitURL)
	log.Printf("   VRP_SOLVER_TIMEOUT: %d seconds", solverTimeout)
	log.Printf("   DISTANCE_MATRIX_CACHE_TTL: %d minutes", cacheTTL)

	// Connect to database
	log.Printf("🗄️  Connecting to database...")
	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	// Run migrations
	log.Printf("📝 Running migrations...")
	err = database.RunMigrations(db)
	if err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Connect to RabbitMQ
	log.Printf("🐰 Connecting to RabbitMQ...")
	rabbitmq, err := messaging.NewRabbitMQClient(rabbitURL, "incidentes")
	if err != nil {
		log.Fatalf("❌ Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	// Initialize cache
	log.Printf("💾 Initializing distance matrix cache...")
	distanceCache := cache.NewDistanceMatrixCache(db, cacheTTL)

	// Initialize VRP optimizer
	log.Printf("🧮 Initializing VRP optimizer...")
	vrpOptimizer := optimizer.NewRouteOptimizer(solverTimeout)

	// Create server
	log.Printf("🚀 Creating server...")
	srv := server.NewServer(db, rabbitmq, distanceCache, vrpOptimizer, port, jwtSecret)

	// Start server
	log.Printf("🌟 Starting route service on port %s", port)
	if err := srv.Start(); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}
