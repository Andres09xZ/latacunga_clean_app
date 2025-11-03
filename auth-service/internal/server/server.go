package server

import (
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/internal/handlers"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Start arranca el servidor Gin y registra handlers mínimos.
func Start() {
	r := gin.Default()

	// Health
	r.GET("/health", handlers.HealthHandler)
	r.GET("/hello-world", handlers.HelloHandler)

	// Auth routes
	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
		// OTP endpoints for citizen users
		authGroup.POST("/otp/send", handlers.SendOTP)
		authGroup.POST("/otp/verify", handlers.VerifyOTP)
	}

	// Swagger UI (requiere generar docs con `swag init`)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	addr := ":8080"
	log.Printf("starting auth service on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
