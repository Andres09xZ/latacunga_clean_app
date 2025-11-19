package events

import (
	"encoding/json"
	"log"
	"os"

	"task-service/internal/handlers"

	"github.com/rabbitmq/amqp091-go"
)

func StartEventConsumer() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		log.Println("Failed to connect to RabbitMQ:", err, "- Event consumer not started")
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open channel:", err)
	}
	defer ch.Close()

	// ============================================================
	// Exchange and queue configuration for novedades events
	// ============================================================
	err = ch.ExchangeDeclare(
		"novedades", // name
		"topic",     // type
		true,        // durable
		false,       // auto-deleted
		false,       // internal
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare novedades exchange:", err)
	}

	// Queue for novedades events
	qNovedades, err := ch.QueueDeclare(
		"task-service-novedades-queue", // name
		true,                           // durable
		false,                          // delete when unused
		false,                          // exclusive
		false,                          // no-wait
		nil,                            // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare novedades queue:", err)
	}

	// Bind to both novedad.creada and novedad.verificada
	err = ch.QueueBind(
		qNovedades.Name,  // queue name
		"novedad.creada", // routing key
		"novedades",      // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind novedad.creada:", err)
	}

	err = ch.QueueBind(
		qNovedades.Name,      // queue name
		"novedad.verificada", // routing key
		"novedades",          // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind novedad.verificada:", err)
	}

	// ============================================================
	// Legacy: reports exchange and queue (for backward compatibility)
	// ============================================================
	err = ch.ExchangeDeclare(
		"reports", // name
		"topic",   // type
		true,      // durable
		false,     // auto-deleted
		false,     // internal
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare reports exchange:", err)
	}

	qReports, err := ch.QueueDeclare(
		"task-service-reports-queue", // name
		true,                         // durable
		false,                        // delete when unused
		false,                        // exclusive
		false,                        // no-wait
		nil,                          // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare reports queue:", err)
	}

	err = ch.QueueBind(
		qReports.Name,     // queue name
		"report.verified", // routing key
		"reports",         // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind report.verified:", err)
	}

	// ============================================================
	// Start consuming from both queues
	// ============================================================
	msgsNovedades, err := ch.Consume(
		qNovedades.Name, // queue
		"",              // consumer
		false,           // auto-ack (manual ack for idempotency)
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		log.Fatal("Failed to register novedades consumer:", err)
	}

	msgsReports, err := ch.Consume(
		qReports.Name, // queue
		"",            // consumer
		false,         // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		log.Fatal("Failed to register reports consumer:", err)
	}

	forever := make(chan bool)

	// Consume novedades events
	go func() {
		for d := range msgsNovedades {
			var event map[string]interface{}
			err := json.Unmarshal(d.Body, &event)
			if err != nil {
				log.Println("Failed to unmarshal event:", err)
				d.Nack(false, true) // Requeue on error
				continue
			}

			eventType := d.RoutingKey
			if err := handlers.HandleNovedadEvent(eventType, event); err != nil {
				log.Println("Failed to handle novedad event:", err)
				d.Nack(false, true) // Requeue on error
				continue
			}

			d.Ack(false) // Acknowledge successful processing
		}
	}()

	// Consume reports events (legacy)
	go func() {
		for d := range msgsReports {
			var event map[string]interface{}
			err := json.Unmarshal(d.Body, &event)
			if err != nil {
				log.Println("Failed to unmarshal event:", err)
				d.Nack(false, true) // Requeue on error
				continue
			}

			if err := handlers.HandleReportVerifiedEvent(event); err != nil {
				log.Println("Failed to handle report event:", err)
				d.Nack(false, true) // Requeue on error
				continue
			}

			d.Ack(false) // Acknowledge successful processing
		}
	}()

	log.Printf(" [*] Task service waiting for messages. To exit press CTRL+C")
	<-forever
}
