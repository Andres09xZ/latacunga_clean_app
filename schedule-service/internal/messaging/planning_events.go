package messaging

import (
	"encoding/json"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// PlanningPublisher publishes planning.resource.requested.v1 events.
type PlanningPublisher struct{ ch *amqp.Channel }

func NewPlanningPublisher(conn *amqp.Connection) (*PlanningPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	// declare exchange (topic) for planning events
	if err := ch.ExchangeDeclare("planning", "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &PlanningPublisher{ch: ch}, nil
}

func (p *PlanningPublisher) PublishResourceRequested(zoneName string, scheduled time.Time, reason string) error {
	if p == nil || p.ch == nil {
		return nil
	} // silently skip if not initialized
	payload := map[string]interface{}{
		"zone":           zoneName,
		"scheduled_time": scheduled.Format(time.RFC3339),
		"reason":         reason,
	}
	body, _ := json.Marshal(payload)
	return p.ch.Publish("planning", "planning.resource.requested.v1", false, false, amqp.Publishing{ContentType: "application/json", Body: body, Timestamp: time.Now()})
}

// PublishRouteRequest publica una solicitud de ruta optimizada al routing-service
func (p *PlanningPublisher) PublishRouteRequest(routeRequest map[string]interface{}) error {
	if p == nil || p.ch == nil {
		return nil
	} // silently skip if not initialized

	// Declarar exchange de planning si no existe
	if err := p.ch.ExchangeDeclare("city.cleaning.planning", "topic", true, false, false, false, nil); err != nil {
		log.Printf("⚠️ Error declarando exchange: %v", err)
		return err
	}

	body, _ := json.Marshal(routeRequest)

	log.Printf("📤 Publicando solicitud de ruta: %s", string(body))

	return p.ch.Publish(
		"city.cleaning.planning",    // exchange
		"planning.run.requested.v1", // routing key (coincide con routing-service)
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
}

// InitPlanningMessaging tries to establish connection; returns publisher or nil.
func InitPlanningMessaging() *PlanningPublisher {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return nil
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("⚠️ RabbitMQ not available: %v", err)
		return nil
	}
	pub, err := NewPlanningPublisher(conn)
	if err != nil {
		log.Printf("⚠️ Publisher init failed: %v", err)
		return nil
	}
	log.Println("✅ Planning events publisher ready")
	return pub
}
