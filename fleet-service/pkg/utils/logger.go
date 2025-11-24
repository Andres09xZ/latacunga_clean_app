package utils

import (
	"log"
	"os"
)

var (
	// InfoLogger para mensajes informativos
	InfoLogger *log.Logger
	// WarnLogger para advertencias
	WarnLogger *log.Logger
	// ErrorLogger para errores
	ErrorLogger *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarnLogger = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
