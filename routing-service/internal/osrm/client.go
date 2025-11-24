package osrm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/models"
)

// OSRMClient maneja las llamadas a OSRM
type OSRMClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOSRMClient crea una nueva instancia del cliente OSRM
func NewOSRMClient(baseURL string) *OSRMClient {
	return &OSRMClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OSRMResponse representa la respuesta completa de OSRM
type OSRMResponse struct {
	Code      string     `json:"code"`
	Trips     []OSRMTrip `json:"trips"`
	Waypoints []Waypoint `json:"waypoints"` // Los waypoints están en la raíz, no en trips
}

// OSRMTrip representa un viaje optimizado
type OSRMTrip struct {
	Geometry string  `json:"geometry"`
	Distance float64 `json:"distance"` // en metros
	Duration float64 `json:"duration"` // en segundos
}

// Waypoint representa un punto de paso ordenado
type Waypoint struct {
	WaypointIndex int       `json:"waypoint_index"`
	TripsIndex    int       `json:"trips_index"`
	Location      []float64 `json:"location"` // [lon, lat]
}

// OptimizeRoute llama a OSRM /trip para optimizar el orden de los puntos
func (c *OSRMClient) OptimizeRoute(points []models.Point) (*OSRMResponse, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("se requieren al menos 2 puntos para optimizar una ruta")
	}

	// Construir la URL dinámica con coordenadas
	coordinates := make([]string, len(points))
	for i, p := range points {
		// OSRM usa formato: longitude,latitude (¡inverso!)
		coordinates[i] = fmt.Sprintf("%.6f,%.6f", p.Longitude, p.Latitude)
	}

	coordsStr := strings.Join(coordinates, ";")

	// source=first: el primer punto es fijo (EPAGAL), es el inicio obligatorio
	// destination=last: el último punto es fijo (para round trip, OSRM duplica el primero internamente)
	// geometries=polyline: respuesta en formato polyline codificado
	// overview=full: geometría completa de la ruta
	// roundtrip=true: ruta circular que regresa al punto inicial (comportamiento por defecto)
	url := fmt.Sprintf("%s/trip/v1/driving/%s?source=first&geometries=polyline&overview=full",
		c.baseURL, coordsStr)

	// Realizar la petición HTTP
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error al llamar a OSRM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSRM devolvió código %d: %s", resp.StatusCode, string(body))
	}

	// Parsear la respuesta JSON
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error al leer respuesta de OSRM: %w", err)
	}

	var osrmResp OSRMResponse
	if err := json.Unmarshal(body, &osrmResp); err != nil {
		return nil, fmt.Errorf("error al parsear JSON de OSRM: %w", err)
	}

	// Log para debug
	fmt.Printf("🔍 [OSRM DEBUG] Code: %s, Trips: %d, Waypoints: %d\n",
		osrmResp.Code, len(osrmResp.Trips), len(osrmResp.Waypoints))

	if len(osrmResp.Trips) > 0 {
		geom := osrmResp.Trips[0].Geometry
		if len(geom) > 50 {
			geom = geom[:50] + "..."
		}
		fmt.Printf("   Trip[0]: Distance=%.2f, Duration=%.2f, Geometry=%s\n",
			osrmResp.Trips[0].Distance, osrmResp.Trips[0].Duration, geom)
	}

	if len(osrmResp.Waypoints) > 0 {
		fmt.Printf("   First Waypoint: Index=%d, Location=%v\n",
			osrmResp.Waypoints[0].WaypointIndex, osrmResp.Waypoints[0].Location)
	}

	// Validar que la respuesta sea exitosa
	if osrmResp.Code != "Ok" {
		return nil, fmt.Errorf("OSRM devolvió código de error: %s", osrmResp.Code)
	}

	if len(osrmResp.Trips) == 0 {
		return nil, fmt.Errorf("OSRM no devolvió ningún viaje")
	}

	return &osrmResp, nil
}

// ExtractWaypointOrder extrae el orden optimizado de los waypoints
func ExtractWaypointOrder(osrmResp *OSRMResponse) []int {
	if len(osrmResp.Waypoints) == 0 {
		return []int{}
	}

	order := make([]int, len(osrmResp.Waypoints))
	for i, wp := range osrmResp.Waypoints {
		order[i] = wp.WaypointIndex
	}
	return order
}
