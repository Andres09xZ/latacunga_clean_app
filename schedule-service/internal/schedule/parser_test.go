package schedule_test

import (
	"testing"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/schedule"
)

func TestParseSchedule_NocturnoTodayOrTomorrow(t *testing.T) {
	now := time.Date(2025, 11, 22, 20, 30, 0, 0, time.UTC)
	dt, label := schedule.ParseSchedule("NOCTURNO (21:00)", now)
	if dt.Day() != 22 || dt.Hour() != 21 {
		t.Fatalf("expected today 21:00 got %v", dt)
	}
	if label == "" {
		t.Fatalf("empty label")
	}
	// after hour should roll to tomorrow
	nowLate := time.Date(2025, 11, 22, 22, 0, 0, 0, time.UTC)
	dt2, _ := schedule.ParseSchedule("NOCTURNO (21:00)", nowLate)
	if dt2.Day() != 23 {
		t.Fatalf("expected tomorrow got %v", dt2)
	}
}

func TestParseSchedule_DiurnoFindNextDay(t *testing.T) {
	// Friday, next listed day is Saturday
	now := time.Date(2025, 11, 21, 10, 0, 0, 0, time.UTC) // Friday
	dt, _ := schedule.ParseSchedule("DIURNO (Mar-Jue-Sab)", now)
	if dt.Weekday() != time.Saturday || dt.Hour() != 8 {
		t.Fatalf("expected Sat 08:00 got %v", dt)
	}
}

func TestParseSchedule_RuralTomorrow(t *testing.T) {
	now := time.Date(2025, 11, 22, 12, 0, 0, 0, time.UTC)
	dt, label := schedule.ParseSchedule("RURAL (Rutas 2,3,4)", now)
	if dt.Day() != 23 || dt.Hour() != 9 {
		t.Fatalf("expected tomorrow 09:00 got %v", dt)
	}
	if label == "" {
		t.Fatalf("empty label")
	}
}
