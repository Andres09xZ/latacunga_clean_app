package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config almacena toda la configuración del servicio
type Config struct {
	Database  DatabaseConfig
	RabbitMQ  RabbitMQConfig
	Server    ServerConfig
	Exchanges ExchangeConfig
	Queues    QueueConfig
}

// DatabaseConfig contiene la configuración de la base de datos
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RabbitMQConfig contiene la configuración de RabbitMQ
type RabbitMQConfig struct {
	URL string
}

// ServerConfig contiene la configuración del servidor HTTP
type ServerConfig struct {
	Port    string
	GinMode string
}

// ExchangeConfig contiene los nombres de los exchanges
type ExchangeConfig struct {
	CityCleaningResources string
	IdentityManagement    string
	PlanningScheduler     string
	OperationsWorkorders  string
}

// QueueConfig contiene los nombres de las colas
type QueueConfig struct {
	IdentitySync     string
	ResourceRequests string
	WorkorderUpdates string
}

// LoadConfig carga la configuración desde variables de entorno
func LoadConfig() (*Config, error) {
	// Intentar cargar el archivo .env (no es error si no existe en producción)
	_ = godotenv.Load()

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "fleet_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8082"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		Exchanges: ExchangeConfig{
			CityCleaningResources: getEnv("EXCHANGE_CITY_CLEANING", "city.cleaning.resources"),
			IdentityManagement:    getEnv("EXCHANGE_IDENTITY", "identity.management"),
			PlanningScheduler:     getEnv("EXCHANGE_PLANNING", "planning.scheduler"),
			OperationsWorkorders:  getEnv("EXCHANGE_OPERATIONS", "operations.workorders"),
		},
		Queues: QueueConfig{
			IdentitySync:     getEnv("QUEUE_IDENTITY_SYNC", "q.fleet.identity-sync"),
			ResourceRequests: getEnv("QUEUE_RESOURCE_REQUESTS", "q.fleet.resource-requests"),
			WorkorderUpdates: getEnv("QUEUE_WORKORDER_UPDATES", "q.fleet.workorder-updates"),
		},
	}

	// Validar configuración crítica
	if err := config.Validate(); err != nil {
		return nil, err
	}

	log.Println("✓ Configuración cargada exitosamente")
	return config, nil
}

// Validate verifica que la configuración sea válida
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST no puede estar vacío")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER no puede estar vacío")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME no puede estar vacío")
	}
	if c.RabbitMQ.URL == "" {
		return fmt.Errorf("RABBITMQ_URL no puede estar vacío")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT no puede estar vacío")
	}
	return nil
}

// GetDSN retorna el Data Source Name para PostgreSQL
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.DBName,
		c.SSLMode,
	)
}

// getEnv obtiene una variable de entorno o retorna un valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
