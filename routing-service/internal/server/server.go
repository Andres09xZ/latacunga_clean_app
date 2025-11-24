package server

import (
	"fmt"
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/service"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	router       *gin.Engine
	routeHandler *handlers.RouteHandler
	port         string
}

// NewServer crea una nueva instancia del servidor HTTP
func NewServer(routeService *service.RouteService, port string) *Server {
	router := gin.Default()
	routeHandler := handlers.NewRouteHandler(routeService)

	return &Server{
		router:       router,
		routeHandler: routeHandler,
		port:         port,
	}
}

// SetupRoutes configura todas las rutas del servidor
func (s *Server) SetupRoutes() {
	// Swagger documentation
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	s.router.GET("/health", s.routeHandler.HealthCheck)

	// API v1
	v1 := s.router.Group("/api/v1/routes")
	{
		v1.GET("", s.routeHandler.GetAllOptimizedRoutes)
		v1.GET("/zone/:zone_id", s.routeHandler.GetOptimizedRouteByZone)
	}

	log.Println("✅ Rutas HTTP configuradas:")
	log.Println("   GET /swagger/index.html - Documentación Swagger")
	log.Println("   GET /health")
	log.Println("   GET /api/v1/routes")
	log.Println("   GET /api/v1/routes/zone/:zone_id")
}

// Start inicia el servidor HTTP
func (s *Server) Start() error {
	log.Printf("🌐 Iniciando servidor HTTP en puerto %s", s.port)
	return s.router.Run(fmt.Sprintf(":%s", s.port))
}
