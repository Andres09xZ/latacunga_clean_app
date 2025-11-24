# Documentación Swagger - Fleet Service

## 📚 Acceso a la Documentación

Una vez que el servicio esté en ejecución, puedes acceder a la documentación interactiva de Swagger en:

```
http://localhost:8082/swagger/index.html
```

## 🔄 Regenerar Documentación

Si realizas cambios en los comentarios de las anotaciones Swagger, regenera la documentación con:

```bash
swag init -g cmd/fleet-service/main.go -o docs
```

O usando el Makefile:

```bash
make swagger
```

## 📝 Estructura de Anotaciones

### Anotaciones Globales (en main.go)

```go
// @title Fleet Service API
// @version 1.0
// @description API para gestión de flota de camiones, conductores y turnos
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email soporte@latacunga-clean.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8082
// @BasePath /api/v1

// @schemes http https
```

### Anotaciones de Endpoints

Cada endpoint debe estar documentado con:

- `@Summary`: Resumen corto del endpoint
- `@Description`: Descripción detallada
- `@Tags`: Categoría/tag del endpoint
- `@Accept`: Tipo de contenido aceptado (json, xml, etc.)
- `@Produce`: Tipo de contenido producido
- `@Param`: Parámetros del endpoint
- `@Success`: Respuesta exitosa
- `@Failure`: Respuestas de error
- `@Router`: Ruta y método HTTP

### Ejemplo:

```go
// ClockIn registra el inicio de un turno
// @Summary Iniciar turno (Clock-in)
// @Description Registra el inicio de turno de un conductor con un camión asignado
// @Tags Turnos
// @Accept json
// @Produce json
// @Param request body ClockInRequest true "Datos del clock-in"
// @Success 200 {object} ClockInResponse "Turno iniciado exitosamente"
// @Failure 400 {object} ErrorResponse "Datos inválidos"
// @Failure 404 {object} ErrorResponse "Conductor no encontrado"
// @Router /shifts/clock-in [post]
func (h *ShiftHandler) ClockIn(c *gin.Context) {
    // ...
}
```

## 📦 Modelos Documentados

Todos los modelos de request y response deben tener ejemplos:

```go
type ClockInRequest struct {
    DriverID   string `json:"driver_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
    TruckPlate string `json:"truck_plate" binding:"required" example:"ABC-1234"`
}
```

## 🎯 Endpoints Documentados

### Turnos (Shifts)

1. **POST /api/v1/shifts/clock-in** - Iniciar turno
   - Request: `ClockInRequest`
   - Response: `ClockInResponse`
   - Códigos: 200, 400, 404, 409, 500

2. **POST /api/v1/shifts/clock-out** - Finalizar turno
   - Request: `ClockOutRequest`
   - Response: `ClockOutResponse`
   - Códigos: 200, 400, 404, 500

### Health Check

1. **GET /health** - Verificar estado del servicio
   - Response: `{"service": "fleet-service", "status": "healthy"}`
   - Código: 200

## 🧪 Testing desde Swagger UI

1. Accede a `http://localhost:8082/swagger/index.html`
2. Expande el endpoint que deseas probar
3. Haz clic en "Try it out"
4. Ingresa los datos de ejemplo o personalizados
5. Haz clic en "Execute"
6. Revisa la respuesta

## 📤 Exportar Documentación

Los archivos generados en `docs/` incluyen:

- `docs.go`: Código Go con la documentación
- `swagger.json`: Especificación OpenAPI en JSON
- `swagger.yaml`: Especificación OpenAPI en YAML

Puedes usar estos archivos para:
- Importar en Postman
- Generar clientes en otros lenguajes
- Publicar en portales de API

## 🔧 Configuración Avanzada

### Agregar Autenticación (Futuro)

```go
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// En el endpoint:
// @Security BearerAuth
```

### Agregar Más Tags

Organiza los endpoints por categorías:

```go
// @Tags Conductores
// @Tags Camiones
// @Tags Reportes
```

## 📖 Referencias

- [Swaggo Documentation](https://github.com/swaggo/swag)
- [OpenAPI Specification](https://swagger.io/specification/)
- [Gin Swagger](https://github.com/swaggo/gin-swagger)
