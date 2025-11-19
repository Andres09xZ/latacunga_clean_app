# Auth Service - Latacunga Clean App

## 📖 Descripción

Servicio de autenticación y autorización que soporta dos modelos de usuarios:

- **👤 Ciudadanos**: Autenticación sin contraseña mediante OTP via Twilio
- **👔 Operadores**: Autenticación con email/contraseña (internos del sistema)

## 🏗️ Arquitectura

```
┌─────────────────────────────────────────────┐
│         API Gateway / Load Balancer         │
└──────────────────┬──────────────────────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
    ┌───▼────────────┐   ┌───▼────────────┐
    │   Ciudadanos   │   │   Operadores   │
    │   (OTP/Phone)  │   │ (Email/Pass)   │
    └───┬────────────┘   └───┬────────────┘
        │                     │
        └──────────┬──────────┘
                   │
        ┌──────────▼──────────┐
        │   Auth Service      │
        │   PostgreSQL        │
        │   JWT Tokens        │
        └──────────┬──────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
    ┌───▼────────────┐   ┌───▼────────────┐
    │ Report Service │   │  Task Service  │
    │  (Novedades)   │   │  (Operaciones) │
    └────────────────┘   └────────────────┘
```

## 🗂️ Estructura del Proyecto

```
auth-service/
├── cmd/server/               # Entry point
│   ├── main.go              # Swagger + init
│   ├── commands/
│   │   ├── login.go
│   │   └── register.go
│   └── docs/                # Swagger generated
├── internal/
│   ├── auth/                # JWT logic
│   │   └── jwt.go
│   ├── cqrs/                # Query handlers
│   │   └── queries/
│   │       └── get_by_email.go
│   ├── database/            # DB initialization
│   │   └── database.go
│   ├── handlers/            # HTTP handlers
│   │   ├── auth_handler.go (NEW: ciudadanos + operadores)
│   │   ├── handler.go
│   │   └── user_handler.go
│   ├── models/              # Data structures
│   │   ├── auth_user.go     (legacy models)
│   │   ├── auth_models.go   (NEW: Citizen, Operator, OTP, Events)
│   │   └── tokens.go
│   ├── repository/          # Database queries
│   │   └── user_repository.go
│   └── server/              # Route registration
│       └── server.go
├── middleware/              # HTTP middleware
│   ├── auth_role.go
│   └── jwt_middleware.go
├── migrations/              # SQL schemas
│   ├── 001_add_id_user.sql
│   ├── 002_nullable_email_password.sql
│   ├── 003_add_tokens_and_otp.sql
│   ├── 004_update_schema_v2.sql
│   └── 005_citizen_otp_schema.sql (NEW)
├── bdd/                     # Behavioral tests
│   ├── features/
│   │   ├── features_v1_auth-service_Version2.feature (legacy)
│   │   └── auth_service_complete.feature (NEW: BDD scenarios)
│   └── bdd_test.go
├── .env                     # Configuration
├── go.mod                   # Dependencies
├── go.sum
├── IMPLEMENTATION_GUIDE.md  # NEW: Detailed guide
└── README.md               # This file
```

## 🚀 Inicio Rápido

### Requisitos
- Go 1.20+
- PostgreSQL 14+
- Twilio Account (para OTP)

### Configuración

1. **Variables de entorno** (`.env`):
```env
# Server
PORT=8080
DB_URL=postgresql://user:password@localhost/latacunga_clean

# JWT
JWT_SECRET=your-secret-key-min-32-chars-long
JWT_EXPIRATION_HOURS=8

# Twilio (ciudadanos OTP)
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_PHONE_NUMBER=+1234567890

# Feature flags
ENABLE_OTP=true
ENABLE_OPERATORS=true
```

2. **Instalar dependencias**:
```bash
cd auth-service
go mod download
```

3. **Ejecutar migraciones**:
```bash
go run cmd/server/main.go
# Las migraciones se ejecutan automáticamente al iniciar
```

4. **Iniciar servidor**:
```bash
go run cmd/server/main.go
# Escucha en http://localhost:8080
# Swagger UI: http://localhost:8080/swagger/index.html
```

## 📝 API Endpoints

### Ciudadanos (OTP)

#### Solicitar OTP
```http
POST /api/v1/auth/otp/send
Content-Type: application/json

{
  "phone_e164": "+593999000111"
}
```

**Respuesta (200 OK):**
```json
{
  "message": "OTP enviado a +593999000111",
  "phone_e164": "+593999000111",
  "expires_in": 300
}
```

**Errores:**
- `400 Bad Request`: Teléfono inválido
- `429 Too Many Requests`: Límite de 3/minuto excedido
- `503 Service Unavailable`: Error con Twilio

#### Verificar OTP
```http
POST /api/v1/auth/otp/verify
Content-Type: application/json

{
  "phone_e164": "+593999000111",
  "code": "123456"
}
```

**Respuesta (200 OK):**
```json
{
  "token": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 28800,
    "scope": "novedades:crear"
  },
  "citizen": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "phone_e164": "+593999000111",
    "verified_at": "2025-11-12T10:30:00Z",
    "created_at": "2025-11-12T10:25:00Z"
  }
}
```

**Errores:**
- `400 Bad Request`: Parámetros inválidos
- `401 Unauthorized`: Código incorrecto o expirado
- `429 Too Many Requests`: Máximo 5 intentos excedido

### Operadores (Email/Contraseña)

#### Registrar Operador
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "name": "Juan García",
  "email": "juan@ejemplo.com",
  "password": "SecurePass123!",
  "role": "operador"
}
```

**Respuesta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "Juan García",
  "email": "juan@ejemplo.com",
  "role": "operador",
  "active": true,
  "created_at": "2025-11-12T09:00:00Z"
}
```

**Errores:**
- `400 Bad Request`: Email o contraseña inválidos
- `409 Conflict`: Email ya existe

#### Login de Operador
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "juan@ejemplo.com",
  "password": "SecurePass123!"
}
```

**Respuesta (200 OK):**
```json
{
  "token": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 28800,
    "scope": "tareas:gestionar reportes:leer"
  },
  "operator": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "Juan García",
    "email": "juan@ejemplo.com",
    "role": "operador",
    "active": true,
    "created_at": "2025-11-12T09:00:00Z"
  }
}
```

**Errores:**
- `401 Unauthorized`: Email o contraseña incorrectos
- `403 Forbidden`: Cuenta inactiva

## 🔒 Seguridad

### Password Hashing
- Algoritmo: bcrypt
- Salt rounds: 10
- Las contraseñas NUNCA se almacenan en texto plano

### JWT Token
- Algoritmo: HS256
- Expiración: 8 horas
- Claims incluidos: `sub`, `email`, `role`, `scope`, `iat`, `exp`
- Firmado con `JWT_SECRET` del `.env`

### Rate Limiting
- OTP: máximo 3 solicitudes por minuto por teléfono
- Reintentos: máximo 5 intentos de OTP por solicitud
- IP: implementar límites globales en API Gateway

### Validación
- Teléfono: E.164 format (`+[1-9]\d{7,14}`)
- Email: RFC 5322 format
- Contraseña: mínimo 8 caracteres, complejidad requerida
- OTP: exactamente 6 dígitos

## 🧪 Testing

### Unit Tests
```bash
go test ./... -v
```

### BDD Tests (Gherkin)
```bash
# Todas las pruebas
go test -v ./bdd/...

# Solo ciudadanos
go test -v ./bdd/... -run "TestCitizenOTP"

# Solo operadores
go test -v ./bdd/... -run "TestOperatorAuth"
```

### Test Manual con Curl
```bash
# Solicitar OTP
curl -X POST http://localhost:8080/api/v1/auth/otp/send \
  -H "Content-Type: application/json" \
  -d '{"phone_e164": "+593999000111"}'

# Verificar OTP (nota: usar código recibido via Twilio)
curl -X POST http://localhost:8080/api/v1/auth/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"phone_e164": "+593999000111", "code": "123456"}'

# Registrar operador
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Juan",
    "email": "juan@ejemplo.com",
    "password": "SecurePass123!",
    "role": "operador"
  }'

# Login operador
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "juan@ejemplo.com",
    "password": "SecurePass123!"
  }'
```

## 📊 Modelos de Datos

Ver `IMPLEMENTATION_GUIDE.md` para esquema completo.

### Principales tablas:
- `usuario.operators` - Operadores internos
- `usuario.citizens` - Ciudadanos (OTP)
- `usuario.otp_requests` - Solicitudes OTP (auditoría)
- `usuario.idempotency_keys` - Prevención de duplicados
- `usuario.outbox_events` - Event sourcing

## 🔄 Flujos de Integración

### Con Report Service
```
Ciudadano verifica OTP
    ↓
Recibe token con scope "novedades:crear"
    ↓
POST /report-service/api/v1/reports
(envía token JWT en Authorization header)
    ↓
Report Service valida JWT contra Auth Service
```

### Con Task Service
```
Operador login
    ↓
Recibe token con scope "tareas:gestionar"
    ↓
GET /task-service/api/v1/tasks
(envía token JWT)
    ↓
Task Service valida JWT + role
```

## 📚 Documentación Adicional

- `IMPLEMENTATION_GUIDE.md` - Guía detallada de implementación
- `bdd/features/auth_service_complete.feature` - Escenarios BDD
- `internal/handlers/auth_handler.go` - Comentarios de implementación

## 🐛 Troubleshooting

### Error: "DB_URL environment variable is not set"
```bash
# Asegurar que .env está en el directorio correcto
cd auth-service
echo "DB_URL=postgresql://..." > .env
go run cmd/server/main.go
```

### Error: "CORS error in Swagger UI"
- CORS ya está habilitado en `server.go`
- Verificar que el servidor responde con headers CORS
- Ver middleware en `server.go`

### Error: "OTP_EXPIRED" al verificar
- Los OTP expiran después de 5 minutos
- Solicitar un nuevo OTP

### Error: "MAX_ATTEMPTS_EXCEEDED"
- Máximo 5 intentos fallidos por OTP
- Solicitar un nuevo OTP

## 📞 Contacto y Soporte

Para problemas o preguntas:
- Revisar `IMPLEMENTATION_GUIDE.md`
- Ver logs del servidor: `go run cmd/server/main.go 2>&1 | grep -i error`
- Verificar conexión a BD: `psql $DB_URL -c "SELECT 1"`

## 📄 Licencia

Proyecto privado de Latacunga Clean App
