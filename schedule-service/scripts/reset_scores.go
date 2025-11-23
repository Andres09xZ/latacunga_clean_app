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
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		PrepareStmt: false,
	})
	if err != nil {
		log.Fatalf("❌ Error conectando: %v", err)
	}
	log.Println("✅ Conectado a Neon PostgreSQL")

	// Resetear todos los scores a 0
	log.Println("🔄 Reseteando todos los scores a 0...")
	result := db.Exec(`
		UPDATE zone_metrics 
		SET current_score = 0, 
		    last_trigger = NULL, 
		    updated_at = NOW()
	`)

	if result.Error != nil {
		log.Fatalf("❌ Error reseteando scores: %v", result.Error)
	}

	log.Printf("✅ Scores reseteados: %d zonas actualizadas\n", result.RowsAffected)

	// Mostrar estado actual
	type ZoneMetric struct {
		ZoneID       int    `gorm:"column:zone_id"`
		ZoneName     string `gorm:"column:zone_name"`
		CurrentScore int    `gorm:"column:current_score"`
		Threshold    int    `gorm:"column:threshold"`
	}

	var metrics []ZoneMetric
	db.Raw(`
		SELECT 
			zm.zone_id, 
			cz.zone_name, 
			zm.current_score, 
			zm.threshold
		FROM zone_metrics zm
		JOIN cleaning_zones cz ON zm.zone_id = cz.id
		ORDER BY zm.zone_id
	`).Scan(&metrics)

	log.Println("\n📊 Estado actual de las zonas:")
	for _, m := range metrics {
		log.Printf("  Zona %d (%s): Score=%d/%d", m.ZoneID, m.ZoneName, m.CurrentScore, m.Threshold)
	}

	log.Println("\n✅ ¡Listo para probar el trigger!")
	log.Println("💡 Ahora puedes simular incidentes para alcanzar el umbral (50 puntos)")
}
