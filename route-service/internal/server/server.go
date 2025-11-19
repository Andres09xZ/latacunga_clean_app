package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"github.com/Andres09xZ/latacunga_clean_app/route-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/cache"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/optimizer"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/middleware"
)

// Server wraps the HTTP server and services
type Server struct {
	router        *gin.Engine
	db            *gorm.DB
	rabbitmq      *messaging.RabbitMQClient
	distanceCache *cache.DistanceMatrixCache
	vrpOptimizer  *optimizer.RouteOptimizer
	port          string
	jwtSecret     string
}

// NewServer creates a new server instance
func NewServer(
	db *gorm.DB,
	rabbitmq *messaging.RabbitMQClient,
	distanceCache *cache.DistanceMatrixCache,
	vrpOptimizer *optimizer.RouteOptimizer,
	port string,
	jwtSecret string,
) *Server {
	router := gin.Default()
	server := &Server{
		router:        router,
		db:            db,
		rabbitmq:      rabbitmq,
		distanceCache: distanceCache,
		vrpOptimizer:  vrpOptimizer,
		port:          port,
		jwtSecret:     jwtSecret,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Health check endpoint (public)
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "route-service"})
	})

	// Swagger documentation
	docs.SwaggerInfo.Title = "Route Service API"
	docs.SwaggerInfo.Description = "API for route optimization"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8084"
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http"}

	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Protected routes with JWT middleware
	protected := s.router.Group("")
	protected.Use(middleware.JWTAuth(s.jwtSecret))
	protected.Use(middleware.RequireOperatorRole())

	routeHandler := handlers.NewRouteHandler(s.db, s.rabbitmq, s.distanceCache, s.vrpOptimizer)

	// Route endpoints
	protected.POST("/api/v1/routes/generate", routeHandler.CreateRoute)
	protected.GET("/api/v1/routes/:id", func(c *gin.Context) {
		routeHandler.GetRoute(c, c.Param("id"))
	})
	protected.GET("/api/v1/routes/:operator_id/:date/steps", routeHandler.GetRouteSteps)

	log.Printf("✅ Routes configured")
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("🚀 Starting route service on port %s", s.port)

	// Initialize event consumers
	err := s.initializeEventConsumers()
	if err != nil {
		return fmt.Errorf("failed to initialize event consumers: %w", err)
	}

	return s.router.Run(":" + s.port)
}

// initializeEventConsumers sets up RabbitMQ event listeners
func (s *Server) initializeEventConsumers() error {
	ctx := context.Background()

	// Handler for tarea_creada events
	tareaCreada := func(ctx context.Context, eventData []byte) error {
		var event messaging.TareaEvent
		if err := json.Unmarshal(eventData, &event); err != nil {
			return fmt.Errorf("failed to unmarshal tarea_creada event: %w", err)
		}

		log.Printf("📌 Received tarea_creada event: %s for operator: %s", event.TareaID, event.OperadorID)

		// Create task in database
		task := models.Task{
			ID:          event.TareaID,
			OperatorID:  event.OperadorID,
			Latitude:    event.Latitud,
			Longitude:   event.Longitud,
			Duration:    event.Duracion,
			Priority:    event.Prioridad,
			Description: event.Descripcion,
		}

		if event.WindowStart != nil {
			windowStart, _ := time.Parse(time.RFC3339, *event.WindowStart)
			task.WindowStart = &windowStart
		}
		if event.WindowEnd != nil {
			windowEnd, _ := time.Parse(time.RFC3339, *event.WindowEnd)
			task.WindowEnd = &windowEnd
		}

		if err := s.db.Create(&task).Error; err != nil {
			log.Printf("⚠️  Task already exists or error: %v", err)
		}

		log.Printf("✅ Task stored: %s", event.TareaID)
		return nil
	}

	// Handler for tarea_asignada events
	tareaAsignada := func(ctx context.Context, eventData []byte) error {
		var event messaging.TareaEvent
		if err := json.Unmarshal(eventData, &event); err != nil {
			return fmt.Errorf("failed to unmarshal tarea_asignada event: %w", err)
		}

		log.Printf("📌 Received tarea_asignada event: %s", event.TareaID)
		return nil
	}

	// Handler for tarea_actualizada events
	tareaActualizada := func(ctx context.Context, eventData []byte) error {
		var event messaging.TareaEvent
		if err := json.Unmarshal(eventData, &event); err != nil {
			return fmt.Errorf("failed to unmarshal tarea_actualizada event: %w", err)
		}

		log.Printf("📌 Received tarea_actualizada event: %s", event.TareaID)

		// Update task in database
		updates := map[string]interface{}{
			"latitude":    event.Latitud,
			"longitude":   event.Longitud,
			"duration":    event.Duracion,
			"priority":    event.Prioridad,
			"description": event.Descripcion,
		}

		if event.WindowStart != nil {
			windowStart, _ := time.Parse(time.RFC3339, *event.WindowStart)
			updates["window_start"] = windowStart
		}
		if event.WindowEnd != nil {
			windowEnd, _ := time.Parse(time.RFC3339, *event.WindowEnd)
			updates["window_end"] = windowEnd
		}

		if err := s.db.Model(&models.Task{}).Where("id = ?", event.TareaID).Updates(updates).Error; err != nil {
			log.Printf("❌ Failed to update task: %v", err)
		}

		return nil
	}

	// Handler for schedule_actualizado events
	scheduleActualizado := func(ctx context.Context, eventData []byte) error {
		var event messaging.ScheduleEvent
		if err := json.Unmarshal(eventData, &event); err != nil {
			return fmt.Errorf("failed to unmarshal schedule_actualizado event: %w", err)
		}

		log.Printf("📌 Received schedule_actualizado event for operator: %s on %s", event.OperadorID, event.FechaLaboral)
		// Trigger route recalculation if needed
		return nil
	}

	// Register handlers
	s.rabbitmq.RegisterEventHandler("tarea_creada", tareaCreada)
	s.rabbitmq.RegisterEventHandler("tarea_asignada", tareaAsignada)
	s.rabbitmq.RegisterEventHandler("tarea_actualizada", tareaActualizada)
	s.rabbitmq.RegisterEventHandler("schedule_actualizado", scheduleActualizado)

	// Start consuming events
	err := s.rabbitmq.ConsumeEvent(ctx, "route.tarea_creada", "tarea.tarea_creada", "tarea_creada")
	if err != nil {
		log.Printf("⚠️  Failed to consume tarea_creada: %v", err)
	}

	err = s.rabbitmq.ConsumeEvent(ctx, "route.tarea_asignada", "tarea.tarea_asignada", "tarea_asignada")
	if err != nil {
		log.Printf("⚠️  Failed to consume tarea_asignada: %v", err)
	}

	err = s.rabbitmq.ConsumeEvent(ctx, "route.tarea_actualizada", "tarea.tarea_actualizada", "tarea_actualizada")
	if err != nil {
		log.Printf("⚠️  Failed to consume tarea_actualizada: %v", err)
	}

	err = s.rabbitmq.ConsumeEvent(ctx, "route.schedule_actualizado", "schedule.schedule_actualizado", "schedule_actualizado")
	if err != nil {
		log.Printf("⚠️  Failed to consume schedule_actualizado: %v", err)
	}

	log.Printf("✅ Event consumers initialized")
	return nil
}
