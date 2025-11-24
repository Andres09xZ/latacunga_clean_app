package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	conn    *amqp.Connection
	channel *amqp.Channel
)

// InitRabbitMQ inicializa la conexión a RabbitMQ
func InitRabbitMQ(url string) error {
	var err error
	conn, err = amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("error al conectar con RabbitMQ: %w", err)
	}

	channel, err = conn.Channel()
	if err != nil {
		return fmt.Errorf("error al abrir canal de RabbitMQ: %w", err)
	}

	// Declarar exchange para input (city.cleaning.planning)
	if err := channel.ExchangeDeclare(
		"city.cleaning.planning", // name
		"topic",                  // type
		true,                     // durable
		false,                    // auto-deleted
		false,                    // internal
		false,                    // no-wait
		nil,                      // arguments
	); err != nil {
		return fmt.Errorf("error al declarar exchange de entrada: %w", err)
	}

	// Declarar exchange para output (city.cleaning.routes)
	if err := channel.ExchangeDeclare(
		"city.cleaning.routes", // name
		"topic",                // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	); err != nil {
		return fmt.Errorf("error al declarar exchange de salida: %w", err)
	}

	// Declarar cola para consumir
	queue, err := channel.QueueDeclare(
		"q.routing.plan-requests", // name
		true,                      // durable
		false,                     // delete when unused
		false,                     // exclusive
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		return fmt.Errorf("error al declarar cola: %w", err)
	}

	// Bind cola al exchange con routing key
	if err := channel.QueueBind(
		queue.Name,                  // queue name
		"planning.run.requested.v1", // routing key
		"city.cleaning.planning",    // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("error al hacer bind de cola: %w", err)
	}

	log.Println("✅ RabbitMQ inicializado correctamente")
	return nil
}

// CloseRabbitMQ cierra la conexión
func CloseRabbitMQ() {
	if channel != nil {
		channel.Close()
	}
	if conn != nil {
		conn.Close()
	}
}

// ConsumeRoutingRequests escucha solicitudes de optimización de rutas
func ConsumeRoutingRequests(routeService *service.RouteService) error {
	msgs, err := channel.Consume(
		"q.routing.plan-requests",  // queue
		"routing-service-consumer", // consumer tag
		false,                      // auto-ack (false para ACK manual)
		false,                      // exclusive
		false,                      // no-local
		false,                      // no-wait
		nil,                        // args
	)
	if err != nil {
		return fmt.Errorf("error al iniciar consumidor: %w", err)
	}

	log.Println("👂 Consumidor de RabbitMQ iniciado, esperando mensajes...")

	// Procesar mensajes en goroutine
	go func() {
		for msg := range msgs {
			log.Printf("[RECEIVED] Mensaje recibido: %s", string(msg.Body))

			// Parsear el mensaje
			var req models.RouteRequest
			if err := json.Unmarshal(msg.Body, &req); err != nil {
				log.Printf("❌ Error al parsear mensaje: %v", err)
				msg.Nack(false, false) // Rechazar mensaje sin reencolar
				continue
			}

			// Validar que haya suficientes puntos
			if len(req.Points) < 2 {
				log.Printf("⚠️  Mensaje con puntos insuficientes (%d puntos): RequestID=%s ZoneID=%d - Descartando mensaje antiguo",
					len(req.Points), req.RequestID, req.ZoneID)
				msg.Ack(false) // ACK para remover de la cola (mensaje inválido/antiguo)
				continue
			}

			// Procesar la solicitud
			response, err := routeService.OptimizeAndSave(req)
			if err != nil {
				log.Printf("❌ Error al optimizar ruta: %v", err)
				// Reencolar para reintentar (error recuperable, ej: OSRM configuración incorrecta)
				msg.Nack(false, true)
				continue
			}

			// Publicar el resultado
			if err := PublishRoutePlanCreated(response); err != nil {
				log.Printf("⚠️  Ruta optimizada pero error al publicar: %v", err)
				msg.Nack(false, true)
				continue
			}

			// ACK del mensaje
			msg.Ack(false)
			log.Printf("✅ Ruta procesada y publicada: RequestID=%s", req.RequestID)
		}
	}()

	return nil
}

// PublishRoutePlanCreated publica un evento de ruta creada
func PublishRoutePlanCreated(response *models.RouteResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error al serializar respuesta: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = channel.PublishWithContext(
		ctx,
		"city.cleaning.routes",   // exchange
		"routes.plan.created.v1", // routing key
		false,                    // mandatory
		false,                    // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("error al publicar mensaje: %w", err)
	}

	log.Printf("[PUBLISHED] Evento routes.plan.created.v1 publicado: RequestID=%s", response.RequestID)
	return nil
}

// ConsumeRPCRequests escucha solicitudes RPC de routing con patrón Request-Reply
func ConsumeRPCRequests(routeService *service.RouteService) error {
	// Declarar cola para RPC
	queue, err := channel.QueueDeclare(
		"q.routing.rpc", // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("error al declarar cola RPC: %w", err)
	}

	// Bind a routing key de RPC
	if err := channel.QueueBind(
		queue.Name,               // queue name
		"routing.optimize",       // routing key (del RPCClient)
		"city.cleaning.planning", // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("error al hacer bind de cola RPC: %w", err)
	}

	// Configurar QoS para procesar un mensaje a la vez
	if err := channel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("error al configurar QoS: %w", err)
	}

	msgs, err := channel.Consume(
		queue.Name,           // queue
		"routing-rpc-worker", // consumer tag
		false,                // auto-ack
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
	if err != nil {
		return fmt.Errorf("error al iniciar consumidor RPC: %w", err)
	}

	log.Println("🔄 Consumidor RPC iniciado en cola: q.routing.rpc")

	// Procesar mensajes RPC
	go func() {
		for msg := range msgs {
			correlationID := msg.CorrelationId
			replyTo := msg.ReplyTo

			log.Printf("📨 [RPC REQUEST] CorrelationId=%s, ReplyTo=%s", correlationID, replyTo)

			// Parsear solicitud
			var req models.RouteRequest
			if err := json.Unmarshal(msg.Body, &req); err != nil {
				log.Printf("❌ Error parseando RPC request: %v", err)
				msg.Nack(false, false)
				continue
			}

			// Validar puntos mínimos
			if len(req.Points) < 2 {
				log.Printf("⚠️  RPC request con puntos insuficientes: %d puntos", len(req.Points))
				// Enviar respuesta de error
				errorResp := map[string]interface{}{
					"error": "se requieren al menos 2 puntos para optimizar ruta",
				}
				errorBody, _ := json.Marshal(errorResp)
				channel.Publish(
					"",      // exchange
					replyTo, // routing key (cola de respuesta)
					false,   // mandatory
					false,   // immediate
					amqp.Publishing{
						ContentType:   "application/json",
						CorrelationId: correlationID,
						Body:          errorBody,
					},
				)
				msg.Ack(false)
				continue
			}

			// Optimizar ruta
			response, err := routeService.OptimizeAndSave(req)
			if err != nil {
				log.Printf("❌ Error optimizando ruta RPC: %v", err)
				errorResp := map[string]interface{}{
					"error": err.Error(),
				}
				errorBody, _ := json.Marshal(errorResp)
				channel.Publish(
					"",
					replyTo,
					false,
					false,
					amqp.Publishing{
						ContentType:   "application/json",
						CorrelationId: correlationID,
						Body:          errorBody,
					},
				)
				msg.Ack(false)
				continue
			}

			// Construir respuesta RPC en el formato esperado por schedule-service
			waypointOrderJSON, _ := json.Marshal(response.WaypointOrder)
			rpcResponse := map[string]interface{}{
				"request_id":      response.RequestID,
				"polyline":        response.Geometry,
				"distance_km":     response.Distance / 1000.0,    // metros a kilómetros
				"duration_min":    int(response.Duration / 60.0), // segundos a minutos
				"optimized_order": response.WaypointOrder,
				"waypoint_order":  string(waypointOrderJSON),
			}

			responseBody, err := json.Marshal(rpcResponse)
			if err != nil {
				log.Printf("❌ Error serializando respuesta RPC: %v", err)
				msg.Nack(false, true)
				continue
			}

			// Enviar respuesta al ReplyTo
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = channel.PublishWithContext(
				ctx,
				"",      // exchange (default)
				replyTo, // routing key
				false,   // mandatory
				false,   // immediate
				amqp.Publishing{
					ContentType:   "application/json",
					CorrelationId: correlationID,
					Body:          responseBody,
					Timestamp:     time.Now(),
				},
			)
			cancel()

			if err != nil {
				log.Printf("❌ Error enviando respuesta RPC: %v", err)
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
			log.Printf("✅ [RPC RESPONSE] Enviado a %s, CorrelationId=%s, Distance=%.2fkm, Duration=%dmin",
				replyTo, correlationID, response.Distance/1000.0, int(response.Duration/60.0))
		}
	}()

	return nil
}
