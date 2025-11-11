package main

//	@title		Auth Service API
//	@version	1.0
//	@description	Servicio de autenticación para registro y login de usuarios
//	@host		localhost:8080
//	@BasePath	/api/v1

import (
	"log"

	_ "github.com/Andres09xZ/latacunga_clean_app/auth-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file - try multiple possible locations
	envPaths := []string{".env", "../.env", "../../auth-service/.env"}
	envLoaded := false

	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Println("No .env file found in any expected location, using system environment variables")
	}

	server.Start()
}
