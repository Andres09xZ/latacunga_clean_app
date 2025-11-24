// Test script for OSRM integration
// Build: go run scripts/test_osrm.go
//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/osrm"
)

func main() {
	fmt.Println("🧪 Testing OSRM Integration...")
	fmt.Println("=====================================")

	// Configurar cliente OSRM
	client := osrm.NewOSRMClient("http://localhost:5000")

	// Puntos de prueba en Latacunga
	points := []models.Point{
		{Latitude: -0.9346, Longitude: -78.6174}, // Centro de Latacunga
		{Latitude: -0.9250, Longitude: -78.6100}, // Zona Norte
		{Latitude: -0.9400, Longitude: -78.6200}, // Zona Sur
		{Latitude: -0.9280, Longitude: -78.6150}, // Zona Este
	}

	fmt.Printf("\n📍 Puntos de prueba (%d):\n", len(points))
	for i, p := range points {
		fmt.Printf("  %d: (%.4f, %.4f)\n", i, p.Latitude, p.Longitude)
	}

	// Llamar a OSRM
	fmt.Println("\n🚀 Llamando a OSRM /trip...")
	resp, err := client.OptimizeRoute(points)
	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	trip := resp.Trips[0]

	// Mostrar resultados
	fmt.Println("\n✅ Respuesta de OSRM:")
	fmt.Printf("  📏 Distancia: %.2f metros (%.2f km)\n", trip.Distance, trip.Distance/1000)
	fmt.Printf("  ⏱️  Duración: %.2f segundos (%.2f minutos)\n", trip.Duration, trip.Duration/60)
	fmt.Printf("  🗺️  Geometría: %s... (%d caracteres)\n", trip.Geometry[:50], len(trip.Geometry))

	// Orden optimizado
	waypointOrder := osrm.ExtractWaypointOrder(trip)
	fmt.Printf("\n🔄 Orden optimizado: %v\n", waypointOrder)
	fmt.Println("\nRuta sugerida:")
	for i, idx := range waypointOrder {
		p := points[idx]
		fmt.Printf("  %d. Punto %d: (%.4f, %.4f)\n", i+1, idx, p.Latitude, p.Longitude)
	}

	// Serializar respuesta como JSON (simulando lo que se enviaría a RabbitMQ)
	response := models.RouteResponse{
		RequestID:     "test-001",
		ZoneID:        1,
		Distance:      trip.Distance,
		Duration:      trip.Duration,
		Geometry:      trip.Geometry,
		WaypointOrder: waypointOrder,
		OptimizedAt:   "2024-11-22T10:30:00Z",
	}

	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	fmt.Println("\n📤 Payload para RabbitMQ:")
	fmt.Println(string(jsonBytes))

	fmt.Println("\n✅ Test completado exitosamente!")
}
