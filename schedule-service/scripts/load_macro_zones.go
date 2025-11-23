package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	ID         int                    `json:"id"`
}

type GeoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type CleaningZone struct {
	ID             uint    `gorm:"primaryKey"`
	ZoneName       string  `gorm:"size:100;not null"`
	ScheduleConfig string  `gorm:"size:100"`
	Color          string  `gorm:"size:20"`
	Geom           string  `gorm:"type:geometry(Polygon,4326)"`
	Status         string  `gorm:"size:50;default:'NORMAL'"`
	AreaKm2        float64 `gorm:"type:decimal(10,4)"`
}

func (CleaningZone) TableName() string {
	return "cleaning_zones"
}

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No se encontró archivo .env, usando variables de entorno del sistema")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL no está configurado")
	}

	// Conectar a la base de datos
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Error conectando a la base de datos: %v", err)
	}
	log.Println("✅ Conectado a la base de datos")

	// Verificar si ya hay zonas cargadas
	var existingCount int64
	db.Model(&CleaningZone{}).Count(&existingCount)
	if existingCount > 0 {
		log.Printf("⚠️  Ya existen %d zonas en la base de datos", existingCount)
		log.Println("🗑️  Limpiando tabla cleaning_zones...")
		if err := db.Exec("TRUNCATE TABLE cleaning_zones RESTART IDENTITY CASCADE").Error; err != nil {
			log.Fatalf("❌ Error limpiando tabla: %v", err)
		}
		log.Println("✅ Tabla limpiada")
	}

	// Leer archivo GeoJSON
	log.Println("📖 Leyendo zonas_macro.geojson...")
	data, err := os.ReadFile("migrations/zonas_macro.geojson")
	if err != nil {
		log.Fatalf("❌ Error leyendo archivo: %v", err)
	}

	var collection GeoJSONFeatureCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		log.Fatalf("❌ Error parseando GeoJSON: %v", err)
	}

	log.Printf("📦 Encontradas %d zonas en el GeoJSON\n", len(collection.Features))

	// Insertar cada zona en una transacción
	tx := db.Begin()
	insertedCount := 0

	for _, feature := range collection.Features {
		zoneName := feature.Properties["zone_name"].(string)
		schedule := feature.Properties["schedule"].(string)

		// Construir geometría GeoJSON compatible con PostGIS
		// Convertir Polygon a MultiPolygon
		geomJSON := map[string]interface{}{
			"type":        "MultiPolygon",
			"coordinates": []json.RawMessage{feature.Geometry.Coordinates},
		}
		geomBytes, _ := json.Marshal(geomJSON)

		// Insertar en la base de datos usando PostGIS con schedule_config
		query := `
			INSERT INTO cleaning_zones (zone_name, route_name, schedule_day, schedule_config, geom, status)
			VALUES ($1, $2, 0, $3, ST_GeomFromGeoJSON($4), 'ACUMULANDO')
		`
		if err := tx.Exec(query, zoneName, schedule, schedule, string(geomBytes)).Error; err != nil {
			log.Printf("❌ Error insertando zona %s: %v", zoneName, err)
			tx.Rollback()
			log.Fatal("❌ Transacción cancelada debido a errores")
		}

		insertedCount++
		log.Printf("✅ Cargada zona: %s (%s)", zoneName, schedule)
	}

	// Confirmar la transacción
	if err := tx.Commit().Error; err != nil {
		log.Fatalf("❌ Error confirmando transacción: %v", err)
	}
	log.Printf("\n✅ Transacción completada: %d zonas insertadas\n", insertedCount)

	// Verificar cuántas zonas se cargaron
	var count int64
	db.Model(&CleaningZone{}).Count(&count)
	log.Printf("\n✅ Total de zonas en la base de datos: %d\n", count)

	// Mostrar resumen
	var zones []CleaningZone
	db.Find(&zones)
	log.Println("\n📋 Resumen de zonas cargadas:")
	for _, zone := range zones {
		log.Printf("  - %s (%s)", zone.ZoneName, zone.ScheduleConfig)
	}

	log.Println("\n✅ ¡Proceso completado exitosamente!")
}
