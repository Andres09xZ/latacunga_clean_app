package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// RunMigrations runs all pending migrations
func Connect(dbURL string) (*gorm.DB, error) {
	// Configuración para evitar el error de cached plan en Neon PostgreSQL
	config := &gorm.Config{
		PrepareStmt: false, // Deshabilitar prepared statements para evitar cache issues
	}

	conn, err := gorm.Open(postgres.Open(dbURL), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL")

	if os.Getenv("SKIP_MIGRATIONS") == "1" {
		log.Println("⏭️  SKIP_MIGRATIONS=1 -> no se ejecutan migraciones")
		return conn, nil
	}
	// Run migrations
	if err := RunMigrations(conn); err != nil {
		log.Printf("⚠️  Warning: Migrations failed: %v", err)
	}

	return conn, nil
}

// RunMigrations runs all pending migrations
func RunMigrations(db *gorm.DB) error {
	log.Println("🔄 Running SQL migrations (all *.sql in migrations folder)...")
	base := "migrations"
	entries, err := os.ReadDir(base)
	if err != nil {
		return fmt.Errorf("cannot read migrations dir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(base, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			continue
		}
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("exec %s failed: %w", f, err)
		}
		log.Printf("✅ Applied %s", filepath.Base(f))
	}
	// mark end
	log.Println("✅ All migrations applied")
	return nil
}
