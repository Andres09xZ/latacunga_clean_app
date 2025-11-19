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

	// Protected routes group
	protected := s.Router.Group("")
	protected.Use(middleware.JWTAuth(jwtSecret))

	// Shift endpoints
	protected.POST("/api/v1/shifts", scheduleHandler.CreateShift)
	protected.GET("/api/v1/shifts", scheduleHandler.ListShifts)
	protected.GET("/api/v1/shifts/:id", scheduleHandler.GetShift)
	protected.PUT("/api/v1/shifts/:id", scheduleHandler.UpdateShift)
	protected.DELETE("/api/v1/shifts/:id", scheduleHandler.DeleteShift)

	// Schedule endpoints (require operator role)
	protected.Use(middleware.RequireOperatorRole())
	protected.GET("/api/v1/schedule/:date", scheduleHandler.GetDailySchedule)
	protected.POST("/api/v1/tasks/:id/assign", scheduleHandler.AssignTaskToShift)

	log.Println("✅ Routes configured")
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
