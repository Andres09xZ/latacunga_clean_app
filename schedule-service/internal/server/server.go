package server

import (
	"fmt"
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/schedule-service/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	Router *gin.Engine
	db     *gorm.DB
	port   string
}

// NewServer creates a new server instance
func NewServer(port string, db *gorm.DB) *Server {
	router := gin.Default()
	return &Server{
		Router: router,
		db:     db,
		port:   port,
	}
}

// Setup configures all routes and middleware
func (s *Server) Setup(jwtSecret string) error {
	// Public health check route
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "schedule-service"})
	})

	// Initialize handlers
	scheduleHandler := handlers.NewScheduleHandler(s.db)
	zoneHandler := handlers.NewZoneHandler(s.db)

	// Protected routes group
	protected := s.Router.Group("/api/v1")
	protected.Use(middleware.JWTAuth())

	// Schedule endpoints
	protected.GET("/schedule/:date", scheduleHandler.GetDailySchedule)
	protected.POST("/tasks/:id/assign", scheduleHandler.AssignTaskToShift)

	// Shift endpoints
	protected.GET("/shifts", scheduleHandler.ListShifts)
	protected.POST("/shifts", scheduleHandler.CreateShift)
	protected.GET("/shifts/:id", scheduleHandler.GetShift)
	protected.PUT("/shifts/:id", scheduleHandler.UpdateShift)
	protected.DELETE("/shifts/:id", scheduleHandler.DeleteShift)

	log.Println("✅ Routes configured")

	// Public zone endpoints (documented separately, no auth required for read operations)
	s.Router.GET("/api/zones", zoneHandler.GetAllZones)
	s.Router.GET("/api/zones/:id", zoneHandler.GetZoneByID)
	s.Router.GET("/api/zones/route/:route_name", zoneHandler.GetZonesByRoute)
	s.Router.GET("/api/zones/day/:day", zoneHandler.GetZonesByDay)
	s.Router.GET("/api/zones/search", zoneHandler.SearchZoneByPoint)
	s.Router.GET("/api/zones/nearest", zoneHandler.GetNearestZone)
	s.Router.GET("/api/zones/summary/routes", zoneHandler.GetZonesSummaryByRoute)
	s.Router.GET("/api/zones/summary/days", zoneHandler.GetZonesSummaryByDay)
	s.Router.GET("/api/zones/geojson", zoneHandler.GetZonesGeoJSON)

	log.Println("🗺️ Zone routes configured")
	return nil
}

// StartEventConsumer starts listening for RabbitMQ events
func (s *Server) StartEventConsumer() {
	scheduleHandler := handlers.NewScheduleHandler(s.db)

	go func() {
		err := messaging.ConsumeTaskCreatedEvents(func(task *messaging.TaskCreatedEvent) error {
			log.Printf("📨 Received task creation event: %s (Report Day: %s)", task.ID, task.ReportDay.Format("2006-01-02"))
			return scheduleHandler.HandleTaskCreatedEvent(task)
		})
		if err != nil {
			log.Printf("❌ Error consuming events: %v", err)
		}
	}()

	log.Println("👂 Event consumer started")
}

// Start runs the server
func (s *Server) Start() error {
	log.Printf("🚀 Starting schedule-service on port %s", s.port)
	return s.Router.Run(fmt.Sprintf(":%s", s.port))
}

// GetDB returns database instance
func (s *Server) GetDB() *gorm.DB {
	return s.db
}

// Close closes database connection
func (s *Server) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
