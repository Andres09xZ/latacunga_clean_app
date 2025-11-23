package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PendingItem struct {
	ID            uint
	Lat           float64
	Lon           float64
	IncidentID    string
	ZoneID        uint
	GravityPoints int
	Status        string
	CreatedAt     string
	UpdatedAt     string
}

func main() {
	godotenv.Load("../.env")
	db, err := gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var items []PendingItem
	db.Table("pending_items").Order("created_at DESC").Limit(10).Find(&items)

	fmt.Println("\n📋 Ultimos incidentes pendientes:")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	if len(items) == 0 {
		fmt.Println("  (No hay incidentes registrados)")
	} else {
		for _, item := range items {
			fmt.Printf("  [%d] Zone:%d | Lat:%.6f Lon:%.6f | Puntos:%d | Status:%s\n",
				item.ID, item.ZoneID, item.Lat, item.Lon, item.GravityPoints, item.Status)
			fmt.Printf("       Incident ID: %s\n", item.IncidentID)
			fmt.Printf("       Creado: %s\n\n", item.CreatedAt)
		}
	}

	var totalPending int64
	db.Table("pending_items").Where("status = ?", "PENDING").Count(&totalPending)
	var totalProcessed int64
	db.Table("pending_items").Where("status = ?", "PROCESSED").Count(&totalProcessed)

	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Printf("📊 Resumen: %d PENDING | %d PROCESSED | %d TOTAL\n\n", totalPending, totalProcessed, len(items))
}
