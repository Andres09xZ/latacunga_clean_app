package database

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establece la conexión con PostgreSQL
func Connect() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to PostgreSQL")

	// Ejecutar migraciones
	if err := RunMigrations(); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
	}
}

// RunMigrations ejecuta los archivos SQL de migración
func RunMigrations() error {
	// Primero crear el schema si no existe
	if err := DB.Exec(`CREATE SCHEMA IF NOT EXISTS incidentes`).Error; err != nil {
		log.Printf("Warning: Failed to create schema: %v", err)
		return err
	}

	// Crear extensiones
	if err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Printf("Warning: Failed to create uuid extension: %v", err)
	}

	if err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS postgis`).Error; err != nil {
		log.Printf("Warning: Failed to create postgis extension: %v", err)
	}

	// Construir ruta a las migraciones
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: Could not get working directory: %v", err)
		return nil
	}

	migrationPath := filepath.Join(wd, "migrations", "001_create_incidents_schema.sql")

	// Si no existe en esa ruta, buscar en rutas alternativas
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		// Intentar ruta relativa desde el binario
		migrationPath = "migrations/001_create_incidents_schema.sql"
	}

	migrationSQL, err := ioutil.ReadFile(migrationPath)
	if err != nil {
		log.Printf("Warning: Could not read migration file from %s: %v", migrationPath, err)
		return nil // No fallar si no existe el archivo
	}

	if err := DB.Exec(string(migrationSQL)).Error; err != nil {
		log.Printf("Warning: Failed to execute migration: %v", err)
		return err
	}

	log.Println("Migrations executed successfully")
	return nil
}
