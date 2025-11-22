package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Cargar .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL environment variable is required")
	}

	log.Println("🔌 Connecting to database...")
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}

	log.Println("📝 Running migration...")

	// Crear esquema
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS validacion").Error; err != nil {
		log.Fatalf("❌ Failed to create schema: %v", err)
	}

	// Crear tabla
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS validacion.incidentes_pendientes (
		incidente_id UUID PRIMARY KEY,
		tipo VARCHAR(50) NOT NULL,
		descripcion TEXT NOT NULL,
		ciudadano_id UUID NOT NULL,
		estado VARCHAR(50) NOT NULL DEFAULT 'pendiente_validacion',
		fecha_evento TIMESTAMP WITH TIME ZONE NOT NULL,
		dia_incidente DATE NOT NULL,
		num_fotos INTEGER DEFAULT 0,
		direccion TEXT,
		latitud DECIMAL(10, 8) NOT NULL,
		longitud DECIMAL(11, 8) NOT NULL,
		geometria GEOMETRY(Point, 4326),
		
		validado_por VARCHAR(100),
		fecha_validacion TIMESTAMP WITH TIME ZONE,
		notas_validacion TEXT,
		
		recibido_en TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		actualizado_en TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`

	if err := db.Exec(createTableSQL).Error; err != nil {
		log.Fatalf("❌ Failed to create table: %v", err)
	}

	// Crear índices
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_incidentes_pendientes_estado ON validacion.incidentes_pendientes(estado)",
		"CREATE INDEX IF NOT EXISTS idx_incidentes_pendientes_fecha ON validacion.incidentes_pendientes(fecha_evento DESC)",
		"CREATE INDEX IF NOT EXISTS idx_incidentes_pendientes_tipo ON validacion.incidentes_pendientes(tipo)",
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Printf("⚠️ Warning creating index: %v", err)
		}
	}

	// Crear función y trigger
	triggerSQL := `
	CREATE OR REPLACE FUNCTION validacion.actualizar_timestamp()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.actualizado_en = NOW();
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS trigger_actualizar_timestamp ON validacion.incidentes_pendientes;

	CREATE TRIGGER trigger_actualizar_timestamp
		BEFORE UPDATE ON validacion.incidentes_pendientes
		FOR EACH ROW
		EXECUTE FUNCTION validacion.actualizar_timestamp();
	`

	if err := db.Exec(triggerSQL).Error; err != nil {
		log.Printf("⚠️ Warning creating trigger: %v", err)
	}

	log.Println("✅ Migration completed successfully!")
	fmt.Println("\n📊 Database structure:")
	fmt.Println("  - Schema: validacion")
	fmt.Println("  - Table: validacion.incidentes_pendientes")
	fmt.Println("  - Indexes: estado, fecha_evento, tipo")
	fmt.Println("  - Trigger: actualizar_timestamp")
}
