package database

import (
	"log"
	"os"

	"task-service/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate models
	err = DB.AutoMigrate(
		&models.Actor{},
		&models.Task{},
		&models.TaskHistory{},
		&models.ProcessedEvent{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
