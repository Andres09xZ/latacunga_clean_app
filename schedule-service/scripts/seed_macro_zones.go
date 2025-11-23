package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ZoneData struct {
	Name           string
	Route          string
	Day            int
	ScheduleConfig string
	Geom           string
}

func main() {
	// Load .env from parent directory
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env not found, using environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable not set")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	zones := []ZoneData{
		{
			Name:           "URBANO_CENTRAL",
			Route:          "URBANO_CENTRAL",
			Day:            0,
			ScheduleConfig: "NOCTURNO (21:00)",
			Geom:           `{"type":"MultiPolygon","coordinates":[[[[-78.625,-0.925],[-78.61,-0.925],[-78.61,-0.945],[-78.625,-0.945],[-78.625,-0.925]]]]}`,
		},
		{
			Name:           "URBANO_NORTE",
			Route:          "URBANO_NORTE",
			Day:            2,
			ScheduleConfig: "DIURNO (Mar-Jue-Sab)",
			Geom:           `{"type":"MultiPolygon","coordinates":[[[[-78.625,-0.925],[-78.61,-0.925],[-78.6,-0.89],[-78.63,-0.89],[-78.625,-0.925]]]]}`,
		},
		{
			Name:           "URBANO_SUR",
			Route:          "URBANO_SUR",
			Day:            1,
			ScheduleConfig: "DIURNO (Lun-Mie-Vie)",
			Geom:           `{"type":"MultiPolygon","coordinates":[[[[-78.625,-0.945],[-78.61,-0.945],[-78.6,-0.98],[-78.64,-0.98],[-78.625,-0.945]]]]}`,
		},
		{
			Name:           "RURAL_NORTE",
			Route:          "RURAL_NORTE",
			Day:            3,
			ScheduleConfig: "RURAL (Rutas 2,3,4)",
			Geom:           `{"type":"MultiPolygon","coordinates":[[[[-78.63,-0.89],[-78.6,-0.89],[-78.55,-0.8],[-78.7,-0.8],[-78.63,-0.89]]]]}`,
		},
		{
			Name:           "RURAL_SUR",
			Route:          "RURAL_SUR",
			Day:            5,
			ScheduleConfig: "RURAL (Rutas 1,5)",
			Geom:           `{"type":"MultiPolygon","coordinates":[[[[-78.64,-0.98],[-78.6,-0.98],[-78.55,-1.05],[-78.7,-1.05],[-78.64,-0.98]]]]}`,
		},
	}

	// Verificar si ya existen zonas
	var count int64
	db.Raw("SELECT COUNT(*) FROM cleaning_zones").Scan(&count)

	if count > 0 {
		fmt.Printf("✅ Ya existen %d zonas en la base de datos. No se insertarán duplicados.\n", count)
		fmt.Println("💡 Si deseas recargar las zonas, ejecuta primero: DELETE FROM cleaning_zones;")
		return
	}

	fmt.Printf("📝 Insertando %d macro zones...\n", len(zones))
	for _, zone := range zones {
		err := db.Exec(`
			INSERT INTO cleaning_zones(zone_name, route_name, schedule_day, schedule_config, points_count, geom, status)
			VALUES ($1, $2, $3, $4, 4, ST_GeomFromGeoJSON($5), 'ACUMULANDO')
			ON CONFLICT (route_name, schedule_day) DO NOTHING
		`, zone.Name, zone.Route, zone.Day, zone.ScheduleConfig, zone.Geom).Error
		if err != nil {
			log.Printf("❌ Failed to insert %s: %v", zone.Name, err)
		} else {
			fmt.Printf("  ✅ %s\n", zone.Name)
		}
	}

	db.Raw("SELECT COUNT(*) FROM cleaning_zones").Scan(&count)
	fmt.Printf("\n✅ Total de zonas en la base de datos: %d\n", count)
}
