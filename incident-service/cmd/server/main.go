package main

import (
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/server"
	"github.com/joho/godotenv"
)

// @title Incident Service API
// @version 1.0
// @description API para el servicio de gestión de incidentes de limpieza de la ciudad
// @termsOfService http://swagger.io/terms/

// @contact.name Soporte API
// @contact.email soporte@latacunga.gob.ec

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8081
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Ingrese el token JWT con el formato: Bearer {token}

func main() {

	//Cargar .env desde la raiz del proyecto
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: No .env file found")
	}

	// Start the server
	server.Start()
}
