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

	// Inicializar zone_metrics para cada zona
	log.Println("🔧 Inicializando zone_metrics...")
	err = db.Exec(`
		INSERT INTO zone_metrics (zone_id, current_score, threshold)
		SELECT id, 0, 50 FROM cleaning_zones cz
		WHERE NOT EXISTS (SELECT 1 FROM zone_metrics zm WHERE zm.zone_id = cz.id)
	`).Error

	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	// Verificar cuántas métricas se crearon
	var count int64
	db.Table("zone_metrics").Count(&count)
	log.Printf("✅ Zone_metrics inicializados: %d registros\n", count)

	log.Println("\n✅ ¡Inicialización completada!")
}
