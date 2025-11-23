package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test multiple coordinates to find one that works
	testCoords := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"Test 1 (example)", -0.935, -78.618},
		{"Test 2 (center)", -0.935, -78.6175},
		{"Test 3 (corner)", -0.925, -78.625},
		{"Test 4 (other corner)", -0.945, -78.61},
		{"Test 5 (middle)", -0.935, -78.62},
	}

	for _, tc := range testCoords {
		var zoneName string
		var zoneID int
		err := db.QueryRow(`
			SELECT id, zone_name 
			FROM cleaning_zones 
			WHERE ST_Contains(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326))
			LIMIT 1
		`, tc.lon, tc.lat).Scan(&zoneID, &zoneName)

		if err == nil {
			fmt.Printf("✅ %s (lat: %.6f, lon: %.6f) -> Zona: %s (ID: %d)\n", tc.name, tc.lat, tc.lon, zoneName, zoneID)
		} else {
			fmt.Printf("❌ %s (lat: %.6f, lon: %.6f) -> No encontrada\n", tc.name, tc.lat, tc.lon)
		}
	}

	// Show actual zone geometries
	fmt.Println("\n📍 Zonas en la base de datos:")
	rows, err := db.Query(`
		SELECT id, zone_name, 
		       ST_AsText(ST_Envelope(geom)) as bbox
		FROM cleaning_zones
		ORDER BY id
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, bbox string
		rows.Scan(&id, &name, &bbox)
		fmt.Printf("  %d. %s - BBox: %s\n", id, name, bbox)
	}
}
