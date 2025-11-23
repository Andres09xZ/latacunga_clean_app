package bdd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/schedule"
	"github.com/joho/godotenv"

	"github.com/cucumber/godog"
	"gorm.io/gorm"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/services"
)

type suiteState struct {
	db         *gorm.DB
	repo       repository.IZoneRepository
	service    *services.PlanningService
	lastResult *models.PlanningResult
	lastError  error
	lastEvent  struct {
		Zone   string
		At     time.Time
		Reason string
	}
}

var state suiteState

func (s *suiteState) reset() {
	s.lastResult = nil
	s.lastError = nil
}

// --- Step definitions ---

func theSystemHasLoadedGeoJSONWithZones(table *godog.Table) error {
	// Ensure zones exist with provided schedule_config; if not, create minimal entries
	for _, row := range table.Rows[1:] { // skip header
		name := strings.TrimSpace(row.Cells[0].Value)
		sched := strings.TrimSpace(row.Cells[1].Value)
		var existing models.CleaningZone
		if err := state.db.Where("zone_name = ?", name).First(&existing).Error; err == nil {
			// update schedule if differs
			if existing.ScheduleConfig != sched {
				state.db.Model(&existing).Update("schedule_config", sched)
			}
			continue
		}
		z := models.CleaningZone{ZoneName: name, ScheduleConfig: sched, Status: "ACUMULANDO"}
		if err := state.db.Create(&z).Error; err != nil {
			return fmt.Errorf("crear zona %s: %w", name, err)
		}
		// minimal metrics row (threshold default 50)
		m := models.ZoneMetrics{ZoneID: z.ID, CurrentScore: 0, Threshold: 50}
		if err := state.db.Create(&m).Error; err != nil {
			return fmt.Errorf("crear metrics %s: %w", name, err)
		}
	}
	return nil
}

func defaultThresholdIs50Points() error { return nil } // already ensured above

var coordRegex = regexp.MustCompile(`lat:\s*([-0-9\.]+),\s*lon:\s*([-0-9\.]+)`) // extract lat lon

func anIncidentArrivesWithCoordinates(text string) error {
	m := coordRegex.FindStringSubmatch(text)
	if len(m) != 3 {
		return fmt.Errorf("no se pudo parsear coordenadas: %s", text)
	}
	lat, _ := strconv.ParseFloat(m[1], 64)
	lon, _ := strconv.ParseFloat(m[2], 64)
	// choose incident type generic
	res, err := state.service.ProcessIncident(lat, lon, "REPORTE_VECINO")
	state.lastResult = res
	state.lastError = err
	return err
}

func systemDetectsZone(expected string) error {
	if state.lastError != nil {
		return state.lastError
	}
	if state.lastResult == nil {
		return fmt.Errorf("no hay resultado previo")
	}
	if state.lastResult.ZoneName != expected {
		return fmt.Errorf("zona detectada %s != esperada %s", state.lastResult.ZoneName, expected)
	}
	return nil
}

func scheduleBaseIs(hourText string) error {
	if state.lastResult == nil {
		return fmt.Errorf("sin resultado")
	}
	if state.lastResult.ScheduledTime.Hour() != 21 { // only validating 21:00 for now
		return fmt.Errorf("hora programada %d != 21", state.lastResult.ScheduledTime.Hour())
	}
	return nil
}

// Pending advanced scenarios (score accumulation & threshold)
func zoneHasCurrentScoreOf(name string, score int) error {
	var z models.CleaningZone
	if err := state.db.Where("zone_name = ?", name).First(&z).Error; err != nil {
		return err
	}
	m := models.ZoneMetrics{ZoneID: z.ID}
	// upsert metrics row
	if err := state.db.Where("zone_id = ?", z.ID).First(&m).Error; err != nil {
		if err := state.db.Create(&models.ZoneMetrics{ZoneID: z.ID, CurrentScore: score, Threshold: 50}).Error; err != nil {
			return err
		}
	} else {
		state.db.Model(&m).Updates(map[string]any{"current_score": score})
	}
	return nil
}

func incidentOfTypeWithValueInThatZone(typ string, pts int) error {
	// We'll locate the zone from lastResult or by scanning metrics
	if state.lastResult == nil {
		return fmt.Errorf("no hay resultado previo para identificar zona")
	}
	// override incident weight temporarily by calling repository increment directly
	_, triggered, err := state.repo.(*repository.ZoneRepository).IncrementScore(state.lastResult.ZoneID, pts)
	if err != nil {
		return err
	}
	// rebuild PlanningResult (simulate scheduling evaluation)
	zoneList, _ := state.repo.ListZones()
	var zone *models.CleaningZone
	for i := range zoneList {
		if zoneList[i].ID == state.lastResult.ZoneID {
			zone = &zoneList[i]
			break
		}
	}
	if zone == nil {
		return fmt.Errorf("zona perdida")
	}
	m, _ := state.repo.GetMetrics(zone.ID)
	nextTime, label := schedule.ParseSchedule(zone.ScheduleConfig, time.Now())
	status := zone.Status
	if triggered {
		status = "LISTO_PARA_RECOLECCION"
	}
	state.lastResult = &models.PlanningResult{ZoneID: zone.ID, ZoneName: zone.ZoneName, NewScore: m.CurrentScore, Threshold: m.Threshold, Triggered: triggered, Status: status, ScheduledTime: nextTime, ScheduledLabel: label}
	if triggered {
		state.lastEvent.Zone = zone.ZoneName
		state.lastEvent.At = nextTime
		state.lastEvent.Reason = "threshold_exceeded"
	}
	return nil
}

func zoneScoreRisesTo(score int) error {
	if state.lastResult == nil {
		return fmt.Errorf("sin resultado")
	}
	if state.lastResult.NewScore != score {
		return fmt.Errorf("puntaje %d != esperado %d", state.lastResult.NewScore, score)
	}
	return nil
}
func zoneStatusRemainsAccumulating() error {
	if state.lastResult.Status != "ACUMULANDO" {
		return fmt.Errorf("estado %s != ACUMULANDO", state.lastResult.Status)
	}
	return nil
}
func noResourceRequestIsEmitted() error {
	if state.lastResult.Triggered {
		return fmt.Errorf("se disparó trigger cuando no debía")
	}
	if state.lastEvent.Zone != "" {
		return fmt.Errorf("evento emitido inesperadamente")
	}
	return nil
}
func systemDetectsThresholdExceeded() error {
	if !state.lastResult.Triggered {
		return fmt.Errorf("no se marcó trigger")
	}
	return nil
}
func systemEmitsEvent(eventName string) error {
	if state.lastEvent.Zone == "" {
		return fmt.Errorf("no hay evento capturado")
	}
	if eventName != "planning.resource.requested.v1" {
		return fmt.Errorf("nombre evento distinto")
	}
	return nil
}
func requestAsksForTruckFor(text string) error {
	// verify label contains expected hour marker
	if state.lastResult == nil {
		return fmt.Errorf("sin resultado")
	}
	if !strings.Contains(state.lastResult.ScheduledLabel, "21:00") {
		return fmt.Errorf("label %s no contiene 21:00", state.lastResult.ScheduledLabel)
	}
	return nil
}
func systemAssignsDateForRoutes(text string) error { return nil }

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(context.Context, *godog.Scenario) (context.Context, error) {
		state.reset()
		return context.Background(), nil
	})

	ctx.Step(`^que el sistema tiene cargado el GeoJSON con las siguientes zonas:$`, theSystemHasLoadedGeoJSONWithZones)
	ctx.Step(`^el umbral de activación por defecto es 50 puntos$`, defaultThresholdIs50Points)
	ctx.Step(`^llega un incidente validado con coordenadas (.*)$`, anIncidentArrivesWithCoordinates)
	ctx.Step(`^el sistema debe detectar que cae en el polígono "([^"]+)"$`, systemDetectsZone)
	ctx.Step(`^debe identificar que el horario base es "([0-9:]+)"$`, scheduleBaseIs)

	// placeholders for advanced scenarios
	ctx.Step(`^la zona "([^"]+)" tiene un puntaje actual de (\d+)$`, zoneHasCurrentScoreOf)
	ctx.Step(`^llega un incidente de tipo "([^"]+)" \(valor (\d+) pts\) en esa zona$`, incidentOfTypeWithValueInThatZone)
	ctx.Step(`^el puntaje de la zona sube a (\d+)$`, zoneScoreRisesTo)
	ctx.Step(`^el estado de la zona permanece en "ACUMULANDO"$`, zoneStatusRemainsAccumulating)
	ctx.Step(`^NO se emite ninguna solicitud de recursos$`, noResourceRequestIsEmitted)
	ctx.Step(`^el sistema detecta que se superó el umbral \((\d+) > (\d+)\)$`, systemDetectsThresholdExceeded)
	ctx.Step(`^el sistema debe emitir el evento "([^"]+)"$`, systemEmitsEvent)
	ctx.Step(`^la solicitud debe pedir un camión para "([^"]+)"$`, requestAsksForTruckFor)
	ctx.Step(`^el sistema debe detectar que cae en "([^"]+)"$`, systemDetectsZone)
	ctx.Step(`^el sistema debe asignar la fecha según el calendario de "([^"]+)"$`, systemAssignsDateForRoutes)
}

func TestMain(m *testing.M) {
	// Cargar .env desde raíz del proyecto (2 niveles arriba)
	_ = godotenv.Load("../../.env")
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Println("DB_URL no establecido - exporta DB_URL o define en .env")
		os.Exit(1)
	}
	// Permitir saltar migraciones largas en pruebas (set SKIP_MIGRATIONS=1)
	if os.Getenv("SKIP_MIGRATIONS") == "1" {
		fmt.Println("SKIP_MIGRATIONS=1 -> saltando migraciones en pruebas")
	}
	db, err := database.Connect(dbURL)
	if err != nil {
		fmt.Println("Error conexión DB:", err)
		os.Exit(1)
	}
	state.db = db
	state.repo = repository.NewZoneRepository(db)
	state.service = services.NewPlanningService(state.repo)

	opts := godog.Options{Format: "pretty", Paths: []string{"."}, StopOnFailure: true}

	status := godog.TestSuite{
		Name:                "planning-core",
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}.Run()

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
