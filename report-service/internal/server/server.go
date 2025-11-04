package server

import (
	"log"

	"github.com/Andres09xZ/latacunga_clean_app/report-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/report-service/internal/handlers"
	"github.com/Andres09xZ/latacunga_clean_app/report-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/report-service/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Start arranca el servidor Gin para report-service.
func Start() {
	database.Connect()
	database.DB.AutoMigrate(&models.Report{})

	r := gin.Default()

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Reports routes (assume JWT middleware from auth-service or shared)
	r.POST("/api/v1/reports", middleware.JWTAuth(), middleware.RequireRole("user"), handlers.CreateReport)
	r.POST("/api/v1/reports/batch", middleware.JWTAuth(), middleware.RequireRole("user"), handlers.CreateBatchReports)
	r.GET("/api/v1/reports", middleware.JWTAuth(), middleware.RequireRole("admin", "operator"), handlers.ListReports)

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	addr := ":8081" // different port
	log.Printf("starting report service on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
