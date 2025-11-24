package main

import (
	"log"

	_ "github.com/Andres09xZ/latacunga_clean_app/operations-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/server"
	"github.com/joho/godotenv"
)

// @title Operations Service API
// @version 1.0
// @description API para gestionar órdenes de trabajo y paradas de recolección
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@latacunga.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8085
// @BasePath /

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Iniciar el servidor
	server.Start()
}
