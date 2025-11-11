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

	// Drop tables in reverse order to avoid foreign key constraints
	err = DB.Migrator().DropTable(&models.OTPCode{}, &models.OperatorProfile{}, &models.User{})
	if err != nil {
		log.Printf("Warning: Could not drop tables: %v", err)
	}

	// Auto-migrate models
	err = DB.AutoMigrate(&models.User{}, &models.OperatorProfile{}, &models.OTPCode{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
