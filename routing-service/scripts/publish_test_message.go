// Test script to publish message to RabbitMQ
// Build: go run scripts/publish_test_message.go
//go:build ignore
// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("📨 Publicando mensaje de prueba a RabbitMQ...")

	// Conectar a RabbitMQ
	rabbitmqURL := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		log.Fatalf("❌ Error al conectar con RabbitMQ: %v", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ Error al abrir canal: %v", err)
	}
	defer channel.Close()

	// Declarar exchange
	if err := channel.ExchangeDeclare(
		"city.cleaning.planning",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("❌ Error al declarar exchange: %v", err)
	}

	// Crear mensaje de prueba
	request := models.RouteRequest{
		RequestID: "test-routing-001",
		ZoneID:    3,
		Points: []models.Point{
			{Latitude: -0.9346, Longitude: -78.6174}, // Centro Latacunga (garaje)
			{Latitude: -0.9250, Longitude: -78.6100}, // Zona Norte
			{Latitude: -0.9400, Longitude: -78.6200}, // Zona Sur
			{Latitude: -0.9280, Longitude: -78.6150}, // Zona Este
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("❌ Error al serializar: %v", err)
	}

	fmt.Println("\n📋 Payload:")
	fmt.Println(string(body))

	// Publicar mensaje
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = channel.PublishWithContext(
		ctx,
		"city.cleaning.planning",    // exchange
		"planning.run.requested.v1", // routing key
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		log.Fatalf("❌ Error al publicar: %v", err)
	}

	fmt.Println("\n✅ Mensaje publicado exitosamente!")
	fmt.Println("🔍 Verifica los logs del Routing Service para ver el procesamiento.")
}
