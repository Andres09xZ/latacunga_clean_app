# Incident Service (Servicio de Incidentes)

Servicio para la gestión de incidentes reportados por ciudadanos siguiendo el patrón **offline-first** con soporte de idempotencia.

## 📋 Características

- **Offline-first**: Soporte de `Idempotency-Key` para sincronización sin duplicados
- **Auto-captura de fecha**: El servidor captura automáticamente `incident_day` (fecha UTC)
- **PostGIS**: Geolocalización de incidentes con puntos geográficos
- **JWT Authentication**: Autenticación con roles (ciudadano, operador, admin)
- **Event-driven**: Publicación de eventos a RabbitMQ
- **5 estados**: Flujo de trabajo completo (emitido → valido → convertido_en_tarea/rechazado → cerrado)
- **4 tipos**: punto_acopio, zona_critica, animal_muerto, zona_reciclaje

## 🚀 Instalación

### Prerrequisitos

- Go 1.21+
- PostgreSQL con extensión PostGIS
- RabbitMQ

### 1. Clonar e instalar dependencias

```bash
cd incident-service
go mod tidy
```

### 2. Configurar variables de entorno

Crea un archivo `.env`:

```env
DB_URL=postgres://usuario:password@localhost:5432/dbname?sslmode=disable
JWT_SECRET=tu_secret_key_aqui
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
PORT=8083
```

### 3. Ejecutar migraciones

```bash
psql -d tu_base_de_datos -f migrations/001_create_incidents_schema.sql
```

### 4. Iniciar servicio

```bash
go run cmd/server/main.go
```

El servicio estará disponible en `http://localhost:8083`

## 📡 API Endpoints

### Health Check

```http
GET /health
```

**Response 200:**
```json
{
  "status": "ok",
  "service": "incident-service"
}
```

### 1. Crear Incidente (Offline-first)

```http
POST /api/v1/incidents
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

**Request Body:**
```json
{
  "idempotency_key": "uuid-generado-en-cliente",
  "type": "animal_muerto",
  "title": "Animal fallecido en vía pública",
  "description": "Perro atropellado en Av. Principal",
  "latitude": -0.9367,
  "longitude": -78.6185,
  "photo_url": "https://s3.amazonaws.com/foto.jpg",
  "evidencia": "data:image/jpeg;base64,..."
}
```

**Response 201:**
```json
{
  "id": "uuid-del-incidente",
  "reporter_kind": "ciudadano",
  "reporter_id": "user-uuid",
  "type": "animal_muerto",
  "status": "emitido",
  "title": "Animal fallecido en vía pública",
  "description": "Perro atropellado en Av. Principal",
  "latitude": -0.9367,
  "longitude": -78.6185,
  "incident_day": "2025-01-15T00:00:00Z",
  "photos_count": 1,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Notas:**
- Solo usuarios con rol **ciudadano** pueden crear incidentes
- El `incident_day` es capturado automáticamente por el servidor (fecha UTC)
- Si se proporciona `idempotency_key` y ya existe, se devuelve el incidente existente (HTTP 200)
- La `evidencia` (foto base64) se guarda automáticamente como attachment inicial

---

### 2. Listar Incidentes

```http
GET /api/v1/incidents?page=1&page_size=20&type=animal_muerto&status=emitido
```

**Query Parameters:**
- `page` (opcional): Número de página (default: 1)
- `page_size` (opcional): Registros por página (default: 20)
- `type` (opcional): Filtrar por tipo (punto_acopio, zona_critica, animal_muerto, zona_reciclaje)
- `status` (opcional): Filtrar por estado (emitido, valido, rechazado, convertido_en_tarea, cerrado)

**Response 200:**
```json
{
  "incidents": [
    {
      "id": "uuid",
      "type": "animal_muerto",
      "status": "emitido",
      "title": "...",
      "latitude": -0.9367,
      "longitude": -78.6185,
      "incident_day": "2025-01-15T00:00:00Z",
      "photos_count": 1,
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 45,
  "total_pages": 3
}
```

---

### 3. Obtener Incidente

```http
GET /api/v1/incidents/:id
```

**Response 200:**
```json
{
  "id": "uuid",
  "reporter_kind": "ciudadano",
  "type": "animal_muerto",
  "status": "valido",
  "title": "...",
  "description": "...",
  "latitude": -0.9367,
  "longitude": -78.6185,
  "incident_day": "2025-01-15T00:00:00Z",
  "photos_count": 2,
  "attachments": [
    {
      "id": "uuid",
      "file_url": "https://...",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "events": [
    {
      "event_type": "incidente_creado",
      "old_status": null,
      "new_status": "emitido",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:31:00Z"
}
```

---

### 4. Actualizar Estado

```http
PUT /api/v1/incidents/:id/status
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

**Request Body:**
```json
{
  "status": "valido",
  "notes": "Incidente verificado y aprobado por operador"
}
```

**Response 200:**
```json
{
  "id": "uuid",
  "status": "valido",
  "events": [
    {
      "event_type": "estado_actualizado",
      "old_status": "emitido",
      "new_status": "valido",
      "notes": "Incidente verificado y aprobado por operador",
      "created_at": "2025-01-15T11:00:00Z"
    }
  ]
}
```

**Notas:**
- Requiere rol **operador** o **admin**
- Estados válidos: emitido, valido, rechazado, convertido_en_tarea, cerrado
- Transiciones de estado permitidas:
  - `emitido` → `valido`, `rechazado`
  - `valido` → `convertido_en_tarea`, `rechazado`
  - `convertido_en_tarea` → `cerrado`
  - `rechazado` → `cerrado`

---

### 5. Agregar Foto/Archivo

```http
POST /api/v1/incidents/:id/attachments
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

**Request Body:**
```json
{
  "file_url": "https://s3.amazonaws.com/foto2.jpg",
  "description": "Foto adicional del incidente"
}
```

**Response 201:**
```json
{
  "id": "uuid-del-attachment",
  "incident_id": "uuid-del-incidente",
  "file_url": "https://s3.amazonaws.com/foto2.jpg",
  "description": "Foto adicional del incidente",
  "created_at": "2025-01-15T12:00:00Z"
}
```

---

## 🔐 Autenticación

Todas las rutas protegidas requieren un token JWT en el header:

```http
Authorization: Bearer <token>
```

El token debe contener los siguientes claims:
```json
{
  "user_id": "uuid-del-usuario",
  "role": "ciudadano|operador|admin"
}
```

## 📊 Flujo de Estados

```
emitido → valido → convertido_en_tarea → cerrado
      ↓     ↓         ↓
   rechazado → cerrado
```

## 🔄 Patrón Offline-First

1. **Cliente sin conexión**:
   - Genera UUID para el incidente
   - Genera `Idempotency-Key` (UUID único)
   - Guarda incidente localmente

2. **Cliente recupera conexión**:
   - Envía incidente con `idempotency_key`
   - Si el servidor ya procesó esa key, devuelve el incidente existente
   - Si es nueva, crea el incidente

3. **Ventajas**:
   - No se crean duplicados
   - Sincronización segura
   - UX sin interrupciones

## 🗄️ Esquema de Base de Datos

El servicio utiliza PostgreSQL con PostGIS. Ver `migrations/001_create_incidents_schema.sql` para el esquema completo.

**Tablas principales:**
- `incidentes.incidents`: Incidentes reportados
- `incidentes.incident_attachments`: Fotos/archivos adjuntos
- `incidentes.incident_events`: Auditoría de eventos
- `incidentes.idempotency_keys`: Control de duplicados
- `incidentes.outbox_events`: Cola de eventos para RabbitMQ

## 🐰 RabbitMQ Events

El servicio publica eventos a RabbitMQ en el exchange **`incidentes`** (type: `topic`):

### Eventos publicados:
- `incidente_creado`: Cuando se crea un incidente
- `estado_actualizado`: Cuando cambia el estado
- `foto_agregada`: Cuando se agrega una foto

**Formato del mensaje:**
```json
{
  "event_type": "incidente_creado",
  "incident_id": "uuid",
  "incident_type": "animal_muerto",
  "status": "emitido",
  "incident_day": "2025-01-15T00:00:00Z",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## 🧪 Testing

### Compilar el proyecto
```bash
go build ./cmd/server
```

### Ejecutar tests (cuando estén disponibles)
```bash
go test ./...
```

### Test manual con curl

1. **Health Check:**
```bash
curl http://localhost:8083/health
```

2. **Crear Incidente:**
```bash
curl -X POST http://localhost:8083/api/v1/incidents \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "idempotency_key": "550e8400-e29b-41d4-a716-446655440000",
    "type": "animal_muerto",
    "title": "Test Incident",
    "description": "Test description",
    "latitude": -0.9367,
    "longitude": -78.6185
  }'
```

## 📁 Estructura del Proyecto

```
incident-service/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── auth/
│   │   └── jwt.go               # JWT generation/validation
│   ├── database/
│   │   └── database.go          # Database connection
│   ├── handlers/
│   │   └── incident_handler.go # HTTP handlers
│   ├── models/
│   │   └── incident.go          # Data models & DTOs
│   └── server/
│       └── server.go            # Router configuration
├── middleware/
│   └── jwt_middleware.go        # JWT & role middleware
├── migrations/
│   └── 001_create_incidents_schema.sql
├── .env                         # Environment variables
├── go.mod
└── README.md
```

## 🌍 Deployment

### Docker (opcional)
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o incident-service ./cmd/server
CMD ["./incident-service"]
```

### Variables de entorno requeridas en producción:
- `DB_URL`: URL de PostgreSQL
- `JWT_SECRET`: Secret para firmar tokens JWT
- `RABBITMQ_URL`: URL de RabbitMQ
- `PORT`: Puerto del servicio (default: 8083)

## 📝 Notas Importantes

1. **incident_day es auto-capturado**: No envíes este campo en el request, el servidor lo genera automáticamente en UTC
2. **Idempotency-Key es opcional pero recomendado**: Úsalo para sincronización offline-first
3. **Evidencia (foto) base64**: Puedes enviar la foto inicial como base64 en el campo `evidencia`
4. **Transiciones de estado**: No todas las transiciones son válidas, verifica el flujo permitido
5. **Roles**: Solo `ciudadano` crea incidentes, solo `operador`/`admin` actualizan estados

## 🤝 Contribuir

Para contribuir al proyecto:
1. Crear una rama: `git checkout -b feature/nueva-funcionalidad`
2. Hacer cambios y commit
3. Push y crear Pull Request

## 📄 Licencia

[Especificar licencia]
