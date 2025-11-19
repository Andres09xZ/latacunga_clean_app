package server

import (
	"log"

	_ "task-service/docs"
	"task-service/internal/database"
	"task-service/internal/events"
	"task-service/internal/handlers"
	"task-service/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Initialize database
	database.InitDB()

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Routes
	api := r.Group("/api/v1")
	api.Use(middleware.JWTAuth())
	{
		// Tasks
		tasks := api.Group("/tasks")
		{
			tasks.GET("/available", handlers.AvailableTasks)
			tasks.GET("", handlers.ListTasks)
			tasks.GET("/:taskId", handlers.GetTask)
			tasks.POST("/:taskId/claim", handlers.ClaimTask)
			tasks.PUT("/:taskId/status", handlers.UpdateTaskStatus)
		}

		// Actors (workers/operators)
		actors := api.Group("/actors")
		{
			actors.POST("/", handlers.CreateActor)
		}
	}

	return r
}

func StartServer() {
	r := SetupRouter()

	// Start event consumer in background
	go events.StartEventConsumer()

	log.Println("Task Service starting on :8082")
	r.Run(":8082")
}
