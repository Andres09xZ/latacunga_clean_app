package main

import (
	"encoding/json"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// Conectar a RabbitMQ
	conn, err := amqp.Dial("amqp://tesis:tesis@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declarar el exchange
	err = ch.ExchangeDeclare(
		"city.cleaning.incidents", // name
		"topic",                   // type
		true,                      // durable
		false,                     // auto-deleted
		false,                     // internal
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	// Crear una cola temporal para consumir
	q, err := ch.QueueDeclare(
		"",    // nombre vacío = cola temporal
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Binding para escuchar todos los eventos de incidentes
	routingKey := "incidents.#" // Escuchar todos los eventos que empiecen con "incidents."
	if len(os.Args) > 1 {
		routingKey = os.Args[1]
	}

	err = ch.QueueBind(
		q.Name,                    // queue name
		routingKey,                // routing key
		"city.cleaning.incidents", // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

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
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	log.Printf("🎧 Escuchando mensajes en exchange 'city.cleaning.incidents' con routing key '%s'", routingKey)
	log.Println("Presiona CTRL+C para salir")
	log.Println("=" + string(make([]byte, 60)))

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("\n📨 Nuevo mensaje recibido:")
			log.Printf("   Routing Key: %s", d.RoutingKey)
			log.Printf("   Content-Type: %s", d.ContentType)
			log.Printf("   Timestamp: %s", d.Timestamp)

			// Intentar formatear como JSON
			var prettyJSON map[string]interface{}
			if err := json.Unmarshal(d.Body, &prettyJSON); err == nil {
				formatted, _ := json.MarshalIndent(prettyJSON, "   ", "  ")
				log.Printf("   Payload:\n   %s", string(formatted))
			} else {
				log.Printf("   Payload: %s", string(d.Body))
			}
			log.Println("   " + string(make([]byte, 60)))
		}
	}()

	<-forever
}
