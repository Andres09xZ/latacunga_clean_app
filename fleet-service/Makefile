# Makefile para Fleet Service

.PHONY: help run build test clean deps migrate docker-build docker-run

help: ## Muestra esta ayuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Instala las dependencias
	go mod download
	go mod tidy

run: ## Ejecuta el servicio
	go run cmd/fleet-service/main.go

build: ## Compila el servicio
	go build -o bin/fleet-service cmd/fleet-service/main.go

test: ## Ejecuta los tests
	go test -v ./...

test-coverage: ## Ejecuta tests con cobertura
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: ## Limpia archivos generados
	rm -rf bin/
	rm -f coverage.out

docker-build: ## Construye la imagen Docker
	docker build -t fleet-service:latest .

docker-run: ## Ejecuta el contenedor Docker
	docker run -p 8082:8082 --env-file .env fleet-service:latest

migrate: ## Ejecuta solo las migraciones
	@echo "Las migraciones se ejecutan automáticamente al iniciar el servicio"

swagger: ## Genera la documentación Swagger
	swag init -g cmd/fleet-service/main.go -o docs
	@echo "Documentación Swagger generada en docs/"

swagger-install: ## Instala swag CLI
	go install github.com/swaggo/swag/cmd/swag@latest

dev: deps ## Configura el entorno de desarrollo
	cp .env.example .env
	@echo "Configuración completa. Edita .env con tus credenciales."
