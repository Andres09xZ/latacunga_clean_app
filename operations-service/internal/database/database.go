package database

import (
	"log"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/operations-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB inicializa la conexión a la base de datos
func InitDB() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Connected to PostgreSQL")

	// Auto-migrate models
	err = DB.AutoMigrate(
		&models.WorkOrder{},
		&models.WorkOrderStop{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database migration completed successfully")
}

// GetDB retorna la instancia de la base de datos
func GetDB() *gorm.DB {
	return DB
}
