package repository_test

import (
	"os"
	"testing"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NOTE: requires DB_URL pointing to a test PostGIS database. Skips if absent.
func setupDB(t *testing.T) *gorm.DB {
	url := os.Getenv("DB_URL")
	if url == "" {
		t.Skip("DB_URL not set; skipping repository tests")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	return db
}

func TestIncrementScoreTriggersThreshold(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewZoneRepository(db)
	// Create zone & metrics row manually without geometry (geom can be null for test)
	z := models.CleaningZone{ZoneName: "TEST_ZONE", ScheduleConfig: "NOCTURNO (21:00)", Status: "ACUMULANDO"}
	if err := db.Create(&z).Error; err != nil {
		t.Fatalf("create zone: %v", err)
	}
	m := models.ZoneMetrics{ZoneID: z.ID, CurrentScore: 48, Threshold: 50}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create metrics: %v", err)
	}
	nm, triggered, err := repo.IncrementScore(z.ID, 5)
	if err != nil {
		t.Fatalf("increment: %v", err)
	}
	if nm.CurrentScore != 53 {
		t.Fatalf("score expected 53 got %d", nm.CurrentScore)
	}
	if !triggered {
		t.Fatalf("expected triggered")
	}
}
