package main

import (
	"log"

	"task-service/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	server.StartServer()
}
