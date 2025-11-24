package database

import (
	"fmt"
	"log"

	"github.com/fleet-service/internal/config"
	"github.com/fleet-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establece la conexión con la base de datos PostgreSQL
func Connect(cfg *config.DatabaseConfig) error {
	dsn := cfg.GetDSN()

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("error al conectar con la base de datos: %w", err)
	}

	// Verificar la conexión
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("error al obtener la instancia de DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("error al hacer ping a la base de datos: %w", err)
	}

	// Configurar pool de conexiones
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("✓ Conectado a PostgreSQL exitosamente")
	return nil
}

// Migrate ejecuta las migraciones automáticas
func Migrate() error {
	log.Println("Ejecutando migraciones...")

	// 🔧 Limpieza previa: Eliminar registros huérfanos con driver_id NULL
	log.Println("🧹 Limpiando registros huérfanos...")
	
	// Eliminar operator_profiles con driver_id NULL
	result := DB.Exec("DELETE FROM operator_profiles WHERE driver_id IS NULL")
	if result.Error != nil {
		log.Printf("⚠️  Advertencia al limpiar operator_profiles: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("🗑️  Eliminados %d operator_profiles huérfanos", result.RowsAffected)
	}

	// Eliminar active_shifts con driver_id o truck_id NULL
	result = DB.Exec("DELETE FROM active_shifts WHERE driver_id IS NULL OR truck_id IS NULL")
	if result.Error != nil {
		log.Printf("⚠️  Advertencia al limpiar active_shifts: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("🗑️  Eliminados %d active_shifts huérfanos", result.RowsAffected)
	}

	log.Println("✓ Limpieza completada")

	// Ejecutar migraciones automáticas
	err := DB.AutoMigrate(
		&models.Truck{},
		&models.Driver{},
		&models.OperatorProfile{},
		&models.ActiveShift{},
	)

	if err != nil {
		return fmt.Errorf("error al ejecutar migraciones: %w", err)
	}

	log.Println("✓ Migraciones ejecutadas exitosamente")
	return nil
}

// Close cierra la conexión con la base de datos
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB retorna la instancia de la base de datos
func GetDB() *gorm.DB {
	return DB
}
