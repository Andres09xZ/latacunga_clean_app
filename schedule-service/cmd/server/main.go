// @title Schedule Service - Planning Core API
// @version 1.0
// @description API de planificación para recolección de residuos de Latacunga. Geolocaliza incidentes, acumula puntaje y dispara eventos de recolección.
// @termsOfService http://latacunga.gob.ec/terms/

// @contact.name API Support
// @contact.email soporte@latacunga.gob.ec

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8083
// @BasePath /
// @schemes http https

package main

import (
	"log"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/server"
	"github.com/joho/godotenv"

	_ "github.com/Andres09xZ/latacunga_clean_app/schedule-service/docs"
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

	// Create server
	srv := server.NewServer(port, db)
	defer srv.Close()
	if err := srv.Setup(); err != nil {
		log.Fatalf("❌ Failed to setup server: %v", err)
	}

	// Start server
	log.Printf("✅ Planning Core service starting on port %s\n", port)
	if err := srv.Start(); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}
