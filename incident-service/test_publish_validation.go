package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ValidationResult struct {
	IncidentID  string `json:"incident_id"`
	Status      string `json:"status"`
	Validator   string `json:"validator"`
	ValidatedAt string `json:"validated_at"`
	Notes       string `json:"notes,omitempty"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found")
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://tesis:tesis@localhost:5672/"
	}

	log.Printf("📡 Connecting to RabbitMQ: %s", rabbitURL)

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Publicar evento de validación de prueba
	result := ValidationResult{
		IncidentID:  "a3411ea9-2b39-4fee-8626-ba532d87d130",
		Status:      "incidente_valido",
		Validator:   "test-manual",
		ValidatedAt: time.Now().UTC().Format(time.RFC3339),
		Notes:       "Test validation event",
	}

	payloadBytes, _ := json.Marshal(result)

	log.Printf("📤 Publishing test validation event...")
	log.Printf("   Exchange: city.cleaning.incidents")
	log.Printf("   Routing Key: incidents.validated.v1")
	log.Printf("   Payload: %s", string(payloadBytes))

	err = ch.Publish(
		"city.cleaning.incidents",
		"incidents.validated.v1",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payloadBytes,
			Timestamp:   time.Now(),
		},
	)

	if err != nil {
		log.Fatalf("❌ Failed to publish: %v", err)
	}

	log.Println("✅ Event published successfully!")
	log.Println("📋 Now check the incident-service logs to see if it receives the event")
	log.Println("   Then run: go run check_incident.go")
}
