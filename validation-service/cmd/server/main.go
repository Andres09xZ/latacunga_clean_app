package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/Andres09xZ/latacunga_clean_app/validation-service/internal/server"
)

// @title Validation Service API
// @version 1.0
// @description Validation service for incident management with manual and automatic validation
// @host localhost:8082
// @basePath /
// @schemes http
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization

func main() {
	// Load .env file - try multiple paths
	envPaths := []string{
		".env",
		"cmd/server/.env",
		"../../cmd/server/.env",
		filepath.Join(os.Getenv("PWD"), ".env"),
	}

	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("✅ Loaded .env from: %s", envPath)
				break
			}
		}
	}

	// Start server
	if err := server.Start(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
