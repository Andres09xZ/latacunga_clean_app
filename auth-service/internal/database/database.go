package database

import (
	"log"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection and migrates models
func InitDB() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Error conectando a Neon PostgreSQL:", err)
	}

	log.Println("Conectado a Neon PostgreSQL")

	// Enable required extensions
	if err = DB.Exec("CREATE EXTENSION IF NOT EXISTS citext").Error; err != nil {
		log.Fatal("Failed to create citext extension:", err)
	}

	// Create schema if not exists
	if err = DB.Exec("CREATE SCHEMA IF NOT EXISTS usuario").Error; err != nil {
		log.Fatal("Failed to create usuario schema:", err)
	}

	// Auto-migrate models
	err = DB.AutoMigrate(
		&models.User{},
		&models.OTPCode{},
		&models.OperatorProfile{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database migration completed successfully")
}
