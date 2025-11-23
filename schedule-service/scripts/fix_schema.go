package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No se encontró archivo .env")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL no está configurado")
	}

	// Conectar a la base de datos
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Error conectando: %v", err)
	}
	log.Println("✅ Conectado a Neon PostgreSQL")

	// Agregar columna schedule_config si no existe
	log.Println("🔧 Agregando columna schedule_config...")
	err = db.Exec(`
		DO $$ BEGIN
			ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS schedule_config TEXT;
		EXCEPTION WHEN duplicate_column THEN
			NULL;
		END $$;
	`).Error

	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	log.Println("✅ Columna schedule_config agregada")

	// Agregar columna status si no existe
	log.Println("🔧 Agregando columna status...")
	err = db.Exec(`
		DO $$ BEGIN
			ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS status VARCHAR(40) DEFAULT 'ACUMULANDO';
		EXCEPTION WHEN duplicate_column THEN
			NULL;
		END $$;
	`).Error

	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	log.Println("✅ Columna status agregada")

	// Crear tabla zone_metrics si no existe
	log.Println("🔧 Creando tabla zone_metrics...")
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS zone_metrics (
			zone_id INT PRIMARY KEY REFERENCES cleaning_zones(id) ON DELETE CASCADE,
			current_score INT NOT NULL DEFAULT 0,
			threshold INT NOT NULL DEFAULT 50,
			last_trigger TIMESTAMPTZ NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`).Error

	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	log.Println("✅ Tabla zone_metrics creada")

	log.Println("\n✅ ¡Schema actualizado correctamente!")
}
