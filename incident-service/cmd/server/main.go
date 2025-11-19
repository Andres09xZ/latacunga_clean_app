package main

import (
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	// Start the server
	server.Start()
}
