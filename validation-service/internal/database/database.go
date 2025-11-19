package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect establishes database connection
func Connect(dbURL string) error {
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	log.Println("✅ Connected to PostgreSQL")

	// Run migrations
	if err := RunMigrations(db); err != nil {
		log.Printf("⚠️  Warning: Migrations failed: %v", err)
		// Don't return error, continue anyway
	}

	return nil
}

// RunMigrations executes SQL migration file
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Create extensions
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}

	// Create schema
	if err := db.Exec(`CREATE SCHEMA IF NOT EXISTS validacion`).Error; err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Try multiple migration file paths
	migrationPaths := []string{
		"migrations/001_create_validation_schema.sql",
		"./migrations/001_create_validation_schema.sql",
		"../../../migrations/001_create_validation_schema.sql",
		filepath.Join(os.Getenv("PWD"), "migrations/001_create_validation_schema.sql"),
	}

	var migrationSQL []byte
	var lastErr error

	for _, migrationPath := range migrationPaths {
		data, err := os.ReadFile(migrationPath)
		if err == nil {
			migrationSQL = data
			log.Printf("✅ Found migration file: %s", migrationPath)
			break
		}
		lastErr = err
	}

	if migrationSQL == nil {
		return fmt.Errorf("failed to read migration file from any path: %w", lastErr)
	}

	// Execute migration
	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Println("✅ Database migrations completed")
	return nil
}

// Close closes database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
