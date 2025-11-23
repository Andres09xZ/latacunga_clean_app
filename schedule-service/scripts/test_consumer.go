package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RoutingRequest representa la estructura del mensaje publicado
type RoutingRequest struct {
	RequestID string     `json:"request_id"`
	ZoneID    int        `json:"zone_id"`
	ZoneName  string     `json:"zone_name"`
	Points    []GeoPoint `json:"points"`
	Timestamp string     `json:"timestamp"`
}

// GeoPoint representa un punto geográfico con metadata
type GeoPoint struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	IncidentID    string  `json:"incident_id"`
	GravityPoints int     `json:"gravity_points"`
}

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Warning: .env file not found")
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("❌ RABBITMQ_URL no está configurado")
	}

	// Conectar a RabbitMQ
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declarar el exchange (debe coincidir con el publisher)
	exchangeName := "city.cleaning.planning"
	err = ch.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange: %v", err)
	}

	// Crear una cola temporal exclusiva
	q, err := ch.QueueDeclare(
		"",    // nombre (vacío = nombre auto-generado)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("❌ Failed to declare queue: %v", err)
	}

	// Bindear la cola al exchange con el routing key
	routingKey := "planning.run.requested.v1"
	err = ch.QueueBind(
		q.Name,       // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("❌ Failed to bind queue: %v", err)
	}

	// Consumir mensajes
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("❌ Failed to register consumer: %v", err)
	}

	log.Printf("🎧 Listening for routing requests on exchange '%s' with key '%s'", exchangeName, routingKey)
	log.Println("📬 Waiting for messages... (Press Ctrl+C to exit)")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Println("\n" + string('='))
			log.Printf("📨 New message received!")
			log.Printf("📋 Routing Key: %s", d.RoutingKey)
			log.Printf("📄 Content Type: %s", d.ContentType)
			log.Printf("⏰ Timestamp: %v", d.Timestamp)
			
			// Parsear el JSON
			var req RoutingRequest
			if err := json.Unmarshal(d.Body, &req); err != nil {
				log.Printf("❌ Error parsing JSON: %v", err)
				log.Printf("📜 Raw body: %s", string(d.Body))
			} else {
				log.Printf("\n🗺️  ROUTING REQUEST DETAILS:")
				log.Printf("   Request ID: %s", req.RequestID)
				log.Printf("   Zone ID: %d", req.ZoneID)
				log.Printf("   Zone Name: %s", req.ZoneName)
				log.Printf("   Timestamp: %s", req.Timestamp)
				log.Printf("   Total Points: %d\n", len(req.Points))
				
				// Mostrar los puntos
				for i, point := range req.Points {
					log.Printf("   Point %d:", i+1)
					log.Printf("      Lat: %.6f", point.Lat)
					log.Printf("      Lon: %.6f", point.Lon)
					log.Printf("      Incident ID: %s", point.IncidentID)
					log.Printf("      Gravity Points: %d", point.GravityPoints)
				}
			}
			log.Println(string('=') + "\n")
		}
	}()

	<-forever
}
