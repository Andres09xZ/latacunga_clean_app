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

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM cleaning_zones").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Total zones in DB: %d\n", count)

	if count > 0 {
		rows, err := db.Query("SELECT id, zone_name, route_name, schedule_day, schedule_config, status FROM cleaning_zones LIMIT 5")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		fmt.Println("\n📋 Sample zones:")
		for rows.Next() {
			var id int
			var name, route, config, status string
			var day int
			rows.Scan(&id, &name, &route, &day, &config, &status)
			fmt.Printf("  ID=%d, Name=%s, Route=%s, Day=%d, Config=%s, Status=%s\n", id, name, route, day, config, status)
		}
	} else {
		fmt.Println("⚠️  No zones found in database")
	}
}
