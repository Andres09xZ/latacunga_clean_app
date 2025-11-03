package database

import (

	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB


// Sirve para conectarse a la base de datos PostgreSQL alojada en Neon.
func Connect(){
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	var err error 
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Error conectando a Neon PostgreSQL:", err)
	}

	log.Println("Conectado a Neon PostgreSQL")

}