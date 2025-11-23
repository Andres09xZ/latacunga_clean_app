package schedule

import (
	"regexp"
	"strings"
	"time"
)

// ParseSchedule interpreta el campo ScheduleConfig y retorna el próximo momento de recolección y etiqueta descriptiva.
// Reglas:
//
//	NOCTURNO (HH:MM) => hoy a HH:MM si falta, sino mañana.
//	DIURNO (Dia-Dia-...) => próximo día de la lista a las 08:00.
//	RURAL (Rutas x,y,z) => mañana 09:00 etiqueta con lista de rutas.
func ParseSchedule(config string, now time.Time) (time.Time, string) {
	config = strings.TrimSpace(config)
	upper := strings.ToUpper(config)

	// NOCTURNO (21:00)
	if strings.HasPrefix(upper, "NOCTURNO") {
		re := regexp.MustCompile(`\((\d{2}:\d{2})\)`) // extrae la hora
		m := re.FindStringSubmatch(config)
		hourStr := "21:00"
		if len(m) == 2 {
			hourStr = m[1]
		}
		parts := strings.Split(hourStr, ":")
		h := 21
		min := 0
		if len(parts) == 2 {
			if ph, err := time.Parse("15", parts[0]); err == nil {
				h = ph.Hour()
			}
			if pm, err := time.Parse("04", parts[1]); err == nil {
				min = pm.Minute()
			}
		}
		candidate := time.Date(now.Year(), now.Month(), now.Day(), h, min, 0, 0, now.Location())
		if candidate.Before(now) {
			candidate = candidate.Add(24 * time.Hour)
		}
		return candidate, candidate.Format("HOY a 15:04")
	}

	// DIURNO (Mar-Jue-Sab)
	if strings.HasPrefix(upper, "DIURNO") {
		re := regexp.MustCompile(`\(([^)]+)\)`) // días dentro de paréntesis
		m := re.FindStringSubmatch(config)
		var days []string
		if len(m) == 2 {
			days = strings.Split(m[1], "-")
		}
		// Mapa días abreviatura -> índice weekday
		idx := map[string]time.Weekday{"DOM": time.Sunday, "LUN": time.Monday, "MAR": time.Tuesday, "MIE": time.Wednesday, "MIÉ": time.Wednesday, "JUE": time.Thursday, "VIE": time.Friday, "SAB": time.Saturday, "SÁB": time.Saturday}
		// Buscar próximo
		for add := 0; add < 14; add++ { // dos semanas máximo
			d := now.Add(time.Duration(add) * 24 * time.Hour)
			for _, target := range days {
				t := strings.ToUpper(strings.TrimSpace(target))
				if w, ok := idx[t]; ok && w == d.Weekday() {
					scheduled := time.Date(d.Year(), d.Month(), d.Day(), 8, 0, 0, 0, d.Location())
					label := d.Format("02/01 08:00")
					return scheduled, label
				}
			}
		}
		// Fallback mañana 08:00
		fallback := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location()).Add(24 * time.Hour)
		return fallback, fallback.Format("02/01 08:00")
	}

	// RURAL (Rutas 2,3,4)
	if strings.HasPrefix(upper, "RURAL") {
		// Simple: ejecutar mañana 09:00
		scheduled := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()).Add(24 * time.Hour)
		return scheduled, scheduled.Format("MAÑANA 09:00")
	}

	// Default: próximo día 08:00
	def := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location()).Add(24 * time.Hour)
	return def, def.Format("02/01 08:00")
}
