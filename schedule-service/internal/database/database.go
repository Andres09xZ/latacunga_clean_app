package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// RunMigrations runs all pending migrations
func Connect(dbURL string) (*gorm.DB, error) {
	conn, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL")

	// Run migrations
	if err := RunMigrations(conn); err != nil {
		log.Printf("⚠️  Warning: Migrations failed: %v", err)
	}

	return conn, nil
}

// RunMigrations runs all pending migrations
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Try multiple migration file paths
	migrationPaths := []string{
		"migrations/001_create_schedule_schema.sql",
		"./migrations/001_create_schedule_schema.sql",
		"../../../migrations/001_create_schedule_schema.sql",
		filepath.Join(os.Getenv("PWD"), "migrations/001_create_schedule_schema.sql"),
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

	if err := db.Exec(string(migrationSQL)).Error; err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Println("✅ Database migrations completed")
	return nil
}
