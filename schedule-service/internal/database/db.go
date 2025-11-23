package database

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds database connection parameters (can be populated from env vars)
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	TimeZone string
}

// LoadConfig reads environment variables and builds a Config
func LoadConfig() Config {
	cfg := Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		TimeZone: os.Getenv("DB_TIMEZONE"),
	}
	if cfg.SSLMode == "" { // Neon requires SSL
		cfg.SSLMode = "require"
	}
	if cfg.TimeZone == "" {
		cfg.TimeZone = "UTC"
	}
	return cfg
}

// DSN builds the postgres DSN for GORM
func (c Config) DSN() string {
	// use simple key=value format supported by lib/pq and pgx
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		c.Host, c.User, c.Password, c.DBName, c.Port, c.SSLMode, c.TimeZone,
	)
}

// New creates a new *gorm.DB connection with sane defaults for serverless Neon
func New(cfg Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}
	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, err
	}
	// Fine-tune connection pool (Neon serverless benefits from lower idle lifetime)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	sqlDB.SetConnMaxLifetime(15 * time.Minute)
	return db, nil
}

// MustNew convenience helper that panics on error (useful in main)
func MustNew() *gorm.DB {
	cfg := LoadConfig()
	db, err := New(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}
	return db
}
