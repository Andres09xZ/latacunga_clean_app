# 📚 Guía de Uso de Swagger - Fleet Service

## ✅ Documentación Swagger Implementada

El Fleet Service ahora cuenta con documentación completa de la API usando **Swagger/OpenAPI 3.0**.

## 🚀 Acceso a la Documentación

### 1. Iniciar el servicio
```bash
go run cmd/fleet-service/main.go
```

### 2. Abrir Swagger UI
Una vez que el servicio esté corriendo, accede a:

```
http://localhost:8084/swagger/index.html
```

## 📋 Endpoints Documentados

### ✅ Health Check
- **GET** `/health`
- Verifica el estado del servicio
- No requiere parámetros

### ✅ Turnos (Shifts)

#### 1. Clock-In (Iniciar Turno)
- **POST** `/api/v1/shifts/clock-in`
- Registra el inicio de turno de un conductor con un camión

**Request Body:**
```json
{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "truck_plate": "ABC-1234"
}
```

**Respuestas:**
- `200`: Turno iniciado exitosamente
- `400`: Datos inválidos o ID inválido
- `404`: Conductor o camión no encontrado
- `409`: Conductor ya tiene turno activo o camión no disponible
- `500`: Error interno

#### 2. Clock-Out (Finalizar Turno)
- **POST** `/api/v1/shifts/clock-out`
- Registra el fin de turno de un conductor

**Request Body:**
```json
{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Respuestas:**
- `200`: Turno finalizado exitosamente
- `400`: Datos inválidos o ID inválido
- `404`: No se encontró turno activo
- `500`: Error interno

## 🧪 Probar la API desde Swagger UI

### Paso 1: Acceder a Swagger UI
```
http://localhost:8084/swagger/index.html
```

### Paso 2: Probar Clock-In

1. Busca la sección **Turnos**
2. Haz clic en **POST /api/v1/shifts/clock-in**
3. Haz clic en **"Try it out"**
4. Edita el JSON con datos reales:
   ```json
   {
     "driver_id": "tu-uuid-de-conductor",
     "truck_plate": "ABC-1234"
   }
   ```
5. Haz clic en **"Execute"**
6. Revisa la respuesta

### Paso 3: Probar Clock-Out

1. Busca **POST /api/v1/shifts/clock-out**
2. Haz clic en **"Try it out"**
3. Ingresa el mismo `driver_id` usado en clock-in
4. Haz clic en **"Execute"**

## 🔄 Regenerar Documentación

Si modificas los comentarios de Swagger en el código:

```bash
# Opción 1: Usar Makefile
make swagger

# Opción 2: Comando directo
swag init -g cmd/fleet-service/main.go -o docs
```

## 📦 Archivos Generados

La documentación se genera en la carpeta `docs/`:

```
docs/
├── docs.go          # Código Go con la documentación
├── swagger.json     # Especificación OpenAPI en JSON
├── swagger.yaml     # Especificación OpenAPI en YAML
└── README.md        # Documentación sobre Swagger
```

## 🔧 Anotaciones Implementadas

### En `main.go`

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

### En `shift_handler.go`

Cada endpoint tiene:
- `@Summary`: Título corto
- `@Description`: Descripción detallada
- `@Tags`: Categoría del endpoint
- `@Accept`: Tipo de contenido aceptado
- `@Produce`: Tipo de contenido de respuesta
- `@Param`: Parámetros del request
- `@Success`: Respuesta exitosa con código y modelo
- `@Failure`: Respuestas de error con códigos
- `@Router`: Ruta y método HTTP

## 📤 Exportar Especificación

### Descargar JSON
```
http://localhost:8084/swagger/doc.json
```

### Descargar YAML
```bash
# El archivo está en: docs/swagger.yaml
```

### Importar en Postman
1. Abre Postman
2. File → Import
3. Selecciona `docs/swagger.json`
4. ¡Listo! Todos los endpoints se importan automáticamente

### Generar Cliente en Otro Lenguaje
Usa herramientas como:
- **OpenAPI Generator**: https://openapi-generator.tech/
- **Swagger Codegen**: https://swagger.io/tools/swagger-codegen/

```bash
# Ejemplo: Generar cliente en Python
openapi-generator-cli generate -i docs/swagger.yaml -g python -o ./client-python
```

## 🎨 Personalizar Swagger UI

Para cambiar el host o basePath en producción, edita las anotaciones en `main.go`:

```go
// @host api.latacunga-clean.com
// @BasePath /api/v1
// @schemes https
```

Luego regenera:
```bash
make swagger
```

## 🔒 Agregar Autenticación (Futuro)

Para agregar autenticación con Bearer Token:

1. Agrega en `main.go`:
```go
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
```

2. En cada endpoint protegido:
```go
// @Security BearerAuth
```

## 📝 Agregar Más Modelos

Para documentar nuevos modelos:

```go
type NuevoModelo struct {
    Campo1 string `json:"campo1" example:"valor ejemplo"`
    Campo2 int    `json:"campo2" example:"123"`
}
```

## 🌐 URLs Útiles

| Descripción | URL |
|-------------|-----|
| **Swagger UI** | http://localhost:8084/swagger/index.html |
| **JSON Spec** | http://localhost:8084/swagger/doc.json |
| **YAML File** | docs/swagger.yaml |
| **Health Check** | http://localhost:8084/health |

## ✅ Checklist de Implementación

- [x] Dependencias de Swagger instaladas
- [x] Anotaciones globales en `main.go`
- [x] Anotaciones en todos los endpoints
- [x] Modelos con ejemplos
- [x] Respuestas de error documentadas
- [x] Swagger UI integrado
- [x] Comandos en Makefile
- [x] Documentación de uso

## 🎯 Próximos Pasos

1. **Ejecutar el servicio**:
   ```bash
   go run cmd/fleet-service/main.go
   ```

2. **Abrir Swagger UI**:
   ```
   http://localhost:8084/swagger/index.html
   ```

3. **Probar los endpoints** desde la interfaz interactiva

4. **Compartir la especificación** con el equipo frontend

## 💡 Tips

- **Actualizar documentación**: Siempre que cambies un endpoint, actualiza los comentarios y ejecuta `make swagger`
- **Validar JSON**: Usa https://editor.swagger.io/ para validar tu especificación
- **Mock Server**: Usa la especificación para crear un servidor mock con Prism: https://stoplight.io/open-source/prism

---

**¡La documentación Swagger está lista para usar!** 🎉
