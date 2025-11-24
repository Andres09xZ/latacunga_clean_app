package database

import (
	"fmt"
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establece conexión con PostgreSQL
func Connect(dbURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("error al conectar con la base de datos: %w", err)
	}

	log.Println("✅ Conexión a PostgreSQL establecida")
	return db, nil
}

// RunMigrations ejecuta las migraciones automáticas de GORM
func RunMigrations(db *gorm.DB) error {
	log.Println("🔄 Ejecutando migraciones automáticas...")

	// Auto-migrar el modelo RoutePlan
	if err := db.AutoMigrate(&models.RoutePlan{}); err != nil {
		return fmt.Errorf("error al ejecutar migraciones: %w", err)
	}

	log.Println("✅ Migraciones completadas")
	return nil
}
