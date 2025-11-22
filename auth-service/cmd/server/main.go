package main

import (
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Conectar a la base de datos y migrar esquemas
	database.InitDB()

	// Iniciar servidor HTTP (bloquea aquí)
	server.Start()
}
