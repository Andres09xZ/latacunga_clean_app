package server

import (
	"log"
	"os"

	_ "github.com/Andres09xZ/latacunga_clean_app/auth-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/auth-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/auth-service/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	files "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Start starts the server
func Start() {
	// Initialize database
	database.InitDB()

	r := gin.Default()

	// CORS middleware
	r.Use(cors.Default())

	// Auth routes
	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
		authGroup.POST("/otp/send", handlers.RequestOTP)
		authGroup.POST("/otp/verify", handlers.VerifyOTP)
	}

	// Admin routes (example)
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.JWTAuth(), middleware.RequireRole("admin"))
	{
		admin.GET("/reports", func(c *gin.Context) {
			c.JSON(403, gin.H{"message": "Acceso denegado"})
		})
	}

	// Swagger (especificar URL del spec para evitar problemas de ruta)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(files.Handler, ginSwagger.URL("http://localhost:8080/swagger/doc.json")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("starting auth service on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
