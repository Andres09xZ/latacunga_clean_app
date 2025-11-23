package server

import (
	"fmt"
	"log"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/repository"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/service"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// initTriggerLogic inicializa la lógica de disparo para RabbitMQ
func initTriggerLogic(
	schedulerRepo repository.ISchedulerRepository,
	pendingRepo repository.IPendingItemRepository,
	zoneRepo repository.IZoneRepository,
	rabbitConn *amqp.Connection,
) (*service.TriggerLogic, error) {
	return service.NewTriggerLogic(schedulerRepo, pendingRepo, zoneRepo, rabbitConn)
}

type Server struct {
	Router     *gin.Engine
	db         *gorm.DB
	port       string
	rabbitConn *amqp.Connection
}

// NewServer creates a new server instance
func NewServer(port string, db *gorm.DB) *Server {
	router := gin.Default()

	// Intentar conectar a RabbitMQ (opcional)
	var rabbitConn *amqp.Connection
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL != "" {
		conn, err := amqp.Dial(rabbitURL)
		if err != nil {
			log.Printf("⚠️ RabbitMQ connection failed: %v", err)
		} else {
			rabbitConn = conn
			log.Println("✅ RabbitMQ connected")
		}
	} else {
		log.Println("⚠️ RABBITMQ_URL not configured")
	}

	return &Server{
		Router:     router,
		db:         db,
		port:       port,
		rabbitConn: rabbitConn,
	}
}

// Setup configures all routes and middleware
func (s *Server) Setup() error {
	// Health check handlers
	healthHandler := handlers.NewHealthHandler(s.rabbitConn)
	s.Router.GET("/health", healthHandler.FullHealth)
	s.Router.GET("/health/rabbitmq", healthHandler.RabbitMQHealth)

	// Swagger documentation
	s.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	repo := repository.NewZoneRepository(s.db)
	pendingRepo := repository.NewPendingItemRepository(s.db)
	schedulerRepo := repository.NewSchedulerRepository(s.db)
	
	planning := handlers.NewPlanningHandler(repo, pendingRepo)
	
	// Inicializar TriggerLogic si RabbitMQ está disponible
	if s.rabbitConn != nil {
		triggerLogic, err := initTriggerLogic(schedulerRepo, pendingRepo, repo, s.rabbitConn)
		if err != nil {
			log.Printf("⚠️ TriggerLogic initialization failed: %v", err)
		} else {
			// Inyectar TriggerLogic al servicio (no al handler)
			planningService := planning.GetService()
			planningService.SetTriggerLogic(triggerLogic)
			planningService.SetSchedulerRepo(schedulerRepo)
			log.Println("✅ TriggerLogic initialized - routing requests will be published to RabbitMQ")
		}
	}

	v1 := s.Router.Group("/api/v1/planning")
	v1.GET("/zones", planning.ListZones)
	v1.GET("/zones/:id/metrics", planning.ZoneMetrics)
	v1.POST("/zones/:id/trigger", planning.ForceTrigger)
	v1.PUT("/config/thresholds", planning.UpdateThresholds)
	v1.POST("/simulate", planning.SimulateIncident)

	log.Println("✅ Planning Core routes ready")
	log.Println("✅ Health check endpoints ready: /health and /health/rabbitmq")
	log.Println("📚 Swagger UI available at /swagger/index.html")
	return nil
}

// StartEventConsumer starts listening for RabbitMQ events
// StartEventConsumer left empty (messaging optional in minimal core)
func (s *Server) StartEventConsumer() {}

// Start runs the server
func (s *Server) Start() error {
	log.Printf("🚀 Starting schedule-service on port %s", s.port)
	return s.Router.Run(fmt.Sprintf(":%s", s.port))
}

// GetDB returns database instance
func (s *Server) GetDB() *gorm.DB {
	return s.db
}

// Close closes database and RabbitMQ connections
func (s *Server) Close() error {
	// Cerrar RabbitMQ
	if s.rabbitConn != nil && !s.rabbitConn.IsClosed() {
		if err := s.rabbitConn.Close(); err != nil {
			log.Printf("⚠️ Error closing RabbitMQ connection: %v", err)
		} else {
			log.Println("✅ RabbitMQ connection closed")
		}
	}

	// Cerrar base de datos
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
