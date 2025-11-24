package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/Andres09xZ/latacunga_clean_app/routing-service/docs"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/osrm"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/server"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/service"
	"github.com/joho/godotenv"
)

// @title Routing Service API
// @version 1.0
// @description API para obtener rutas optimizadas de limpieza por zona
// @host localhost:8086
// @BasePath /

func main() {
	log.Println("🚀 Iniciando Routing Service...")

	// Cargar variables de entorno
	paths := []string{".env", "cmd/server/.env", "../../.env"}
	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("✅ Variables de entorno cargadas desde %s", path)
			break
		}
	}

	// Obtener configuración desde variables de entorno
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("❌ DB_URL no está configurado")
	}

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	osrmURL := os.Getenv("OSRM_URL")
	if osrmURL == "" {
		osrmURL = "http://localhost:5000"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	// Conectar a la base de datos
	log.Println("🔌 Conectando a PostgreSQL...")
	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("❌ Error al conectar con la base de datos: %v", err)
	}

	// Ejecutar migraciones
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("❌ Error al ejecutar migraciones: %v", err)
	}

	// Inicializar cliente OSRM
	log.Printf("🗺️  Configurando cliente OSRM en %s...", osrmURL)
	osrmClient := osrm.NewOSRMClient(osrmURL)

	// Crear servicio de rutas
	routeService := service.NewRouteService(db, osrmClient)

	// Inicializar RabbitMQ
	log.Println("🐰 Conectando a RabbitMQ...")
	if err := messaging.InitRabbitMQ(rabbitmqURL); err != nil {
		log.Fatalf("❌ Error al conectar con RabbitMQ: %v", err)
	}
	defer messaging.CloseRabbitMQ()

	// Iniciar consumidor de mensajes legacy (en goroutine)
	if err := messaging.ConsumeRoutingRequests(routeService); err != nil {
		log.Fatalf("❌ Error al iniciar consumidor: %v", err)
	}

	// Iniciar consumidor RPC (Request-Reply pattern)
	if err := messaging.ConsumeRPCRequests(routeService); err != nil {
		log.Fatalf("❌ Error al iniciar consumidor RPC: %v", err)
	}

	log.Println("✅ Routing Service iniciado correctamente")
	log.Println("👂 Esperando mensajes en cola q.routing.plan-requests...")
	log.Println("🔄 Consumidor RPC escuchando en q.routing.rpc (Request-Reply pattern)")

	// Crear y configurar servidor HTTP
	httpServer := server.NewServer(routeService, port)
	httpServer.SetupRoutes()

	// Iniciar servidor HTTP en goroutine
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Fatalf("❌ Error al iniciar servidor HTTP: %v", err)
		}
	}()

	// Esperar señal de terminación
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("🛑 Cerrando Routing Service...")
}
