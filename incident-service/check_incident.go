package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Incident struct {
	ID     string `gorm:"column:id"`
	Status string `gorm:"column:status"`
}

func (Incident) TableName() string {
	return "incidentes.incidents"
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL not set")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}

	// Buscar el incidente que acabamos de validar
	incidentID := "a3411ea9-2b39-4fee-8626-ba532d87d130"

	var incident Incident
	if err := db.Where("id = ?", incidentID).First(&incident).Error; err != nil {
		log.Fatalf("❌ Incident not found: %v", err)
	}

	fmt.Printf("\n📊 Incident Status Check:\n")
	fmt.Printf("   ID: %s\n", incident.ID)
	fmt.Printf("   Status: %s\n", incident.Status)
	fmt.Println()

	if incident.Status == "incidente_valido" {
		fmt.Println("✅ SUCCESS - Incident was updated by validation consumer!")
	} else if incident.Status == "incidente_no_validado" {
		fmt.Println("⚠️ PENDING - Incident is still 'incidente_no_validado'")
		fmt.Println("   The validation consumer might not be running or didn't receive the event")
	} else {
		fmt.Printf("ℹ️ INFO - Incident has status: %s\n", incident.Status)
	}
}
