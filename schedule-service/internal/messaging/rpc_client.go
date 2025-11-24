package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/models"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	RPCTimeout = 10 * time.Second
	ReplyQueue = "q.scheduler.replies"
)

// RPCClient maneja comunicación RPC Request-Reply sobre RabbitMQ
type RPCClient struct {
	conn            *amqp.Connection
	channel         *amqp.Channel
	replyQueue      amqp.Queue
	pendingRequests map[string]chan []byte // correlationID -> response channel
	mutex           sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewRPCClient crea un nuevo cliente RPC
func NewRPCClient(rabbitURL string) (*RPCClient, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("error al conectar a RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error al abrir canal: %w", err)
	}

	// Declarar cola de respuestas exclusiva
	replyQueue, err := ch.QueueDeclare(
		ReplyQueue, // name
		false,      // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("error al declarar cola de respuestas: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &RPCClient{
		conn:            conn,
		channel:         ch,
		replyQueue:      replyQueue,
		pendingRequests: make(map[string]chan []byte),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Iniciar consumidor de respuestas
	go client.consumeReplies()

	log.Printf("✅ RPCClient inicializado con cola de respuestas: %s", ReplyQueue)
	return client, nil
}

// consumeReplies escucha la cola de respuestas y despacha a los canales correspondientes
func (c *RPCClient) consumeReplies() {
	msgs, err := c.channel.Consume(
		c.replyQueue.Name, // queue
		"",                // consumer
		true,              // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		log.Printf("❌ Error al consumir respuestas RPC: %v", err)
		return
	}

	log.Printf("🔄 Consumidor de respuestas RPC iniciado en cola: %s", c.replyQueue.Name)

	for {
		select {
		case <-c.ctx.Done():
			log.Println("🛑 Consumidor de respuestas RPC detenido")
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("⚠️  Canal de mensajes RPC cerrado")
				return
			}

			correlationID := msg.CorrelationId
			if correlationID == "" {
				log.Println("⚠️  Mensaje sin CorrelationId recibido")
				continue
			}

			c.mutex.RLock()
			responseChan, exists := c.pendingRequests[correlationID]
			c.mutex.RUnlock()

			if exists {
				log.Printf("📨 Respuesta RPC recibida para CorrelationId: %s", correlationID)
				select {
				case responseChan <- msg.Body:
					// Respuesta enviada exitosamente
				case <-time.After(1 * time.Second):
					log.Printf("⚠️  Timeout al enviar respuesta a canal para CorrelationId: %s", correlationID)
				}
			} else {
				log.Printf("⚠️  No hay solicitud pendiente para CorrelationId: %s", correlationID)
			}
		}
	}
}

// RequestFleet solicita recursos (chofer + camión) al Fleet Service
func (c *RPCClient) RequestFleet(req models.FleetRequest) (*models.FleetResponse, error) {
	correlationID := uuid.New().String()
	responseChan := make(chan []byte, 1)

	// Registrar solicitud pendiente
	c.mutex.Lock()
	c.pendingRequests[correlationID] = responseChan
	c.mutex.Unlock()

	// Limpiar al finalizar
	defer func() {
		c.mutex.Lock()
		delete(c.pendingRequests, correlationID)
		c.mutex.Unlock()
		close(responseChan)
	}()

	// Serializar request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error al serializar FleetRequest: %w", err)
	}

	// Publicar solicitud
	err = c.channel.PublishWithContext(
		c.ctx,
		"city.cleaning.planning", // exchange
		"fleet.resource.request", // routing key
		false,                    // mandatory
		false,                    // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       c.replyQueue.Name,
			Body:          body,
			Timestamp:     time.Now(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error al publicar FleetRequest: %w", err)
	}

	log.Printf("📤 FleetRequest enviado: ZoneID=%d, CorrelationId=%s", req.ZoneID, correlationID)

	// Esperar respuesta con timeout
	select {
	case responseBody := <-responseChan:
		var fleetResp models.FleetResponse
		if err := json.Unmarshal(responseBody, &fleetResp); err != nil {
			return nil, fmt.Errorf("error al parsear FleetResponse: %w", err)
		}
		log.Printf("✅ FleetResponse recibido: Driver=%s, Truck=%s", fleetResp.DriverID, fleetResp.TruckPlate)
		return &fleetResp, nil

	case <-time.After(RPCTimeout):
		return nil, fmt.Errorf("timeout esperando respuesta de Fleet Service (CorrelationId: %s)", correlationID)

	case <-c.ctx.Done():
		return nil, fmt.Errorf("RPCClient cerrado durante solicitud")
	}
}

// RequestRouting solicita optimización de ruta al Routing Service
func (c *RPCClient) RequestRouting(req models.RoutingRequest) (*models.RoutingResponse, error) {
	correlationID := uuid.New().String()
	responseChan := make(chan []byte, 1)

	// Registrar solicitud pendiente
	c.mutex.Lock()
	c.pendingRequests[correlationID] = responseChan
	c.mutex.Unlock()

	// Limpiar al finalizar
	defer func() {
		c.mutex.Lock()
		delete(c.pendingRequests, correlationID)
		c.mutex.Unlock()
		close(responseChan)
	}()

	// Serializar request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error al serializar RoutingRequest: %w", err)
	}

	// Publicar solicitud
	err = c.channel.PublishWithContext(
		c.ctx,
		"city.cleaning.planning", // exchange
		"routing.optimize",       // routing key
		false,                    // mandatory
		false,                    // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       c.replyQueue.Name,
			Body:          body,
			Timestamp:     time.Now(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error al publicar RoutingRequest: %w", err)
	}

	log.Printf("📤 RoutingRequest enviado: ZoneID=%d, Points=%d, CorrelationId=%s",
		req.ZoneID, len(req.Points), correlationID)

	// Esperar respuesta con timeout
	select {
	case responseBody := <-responseChan:
		var routingResp models.RoutingResponse
		if err := json.Unmarshal(responseBody, &routingResp); err != nil {
			return nil, fmt.Errorf("error al parsear RoutingResponse: %w", err)
		}
		log.Printf("✅ RoutingResponse recibido: Distance=%.2fkm, Duration=%dmin",
			routingResp.DistanceKm, routingResp.DurationMin)
		return &routingResp, nil

	case <-time.After(RPCTimeout):
		return nil, fmt.Errorf("timeout esperando respuesta de Routing Service (CorrelationId: %s)", correlationID)

	case <-c.ctx.Done():
		return nil, fmt.Errorf("RPCClient cerrado durante solicitud")
	}
}

// ReleaseFleet envía compensación para liberar recursos (rollback)
func (c *RPCClient) ReleaseFleet(req models.FleetReleaseRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error al serializar FleetReleaseRequest: %w", err)
	}

	err = c.channel.PublishWithContext(
		c.ctx,
		"city.cleaning.planning", // exchange
		"fleet.resource.release", // routing key
		false,                    // mandatory
		false,                    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("error al publicar FleetReleaseRequest: %w", err)
	}

	log.Printf("🔄 Compensación enviada: Driver=%s, Truck=%s, Reason=%s",
		req.DriverID, req.TruckPlate, req.Reason)
	return nil
}

// Close cierra el cliente RPC
func (c *RPCClient) Close() error {
	c.cancel()
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
