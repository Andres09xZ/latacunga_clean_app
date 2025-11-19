# Guía de Implementación - Auth Service

## 📋 Overview

El auth-service implementa dos flujos de autenticación independientes:

1. **Ciudadanos**: Autenticación sin contraseña via OTP (teléfono)
2. **Operadores**: Autenticación con email/contraseña (internos del sistema)

## 🗄️ Esquema de Base de Datos

### Tablas Principales

**usuario.citizens**
```
- id (UUID, PK)
- phone_e164 (TEXT, UNIQUE, E.164 format: +593999000111)
- verified_at (TIMESTAMPTZ, NULL = no verificado)
- created_at, updated_at
```

**usuario.operators**
```
- id (UUID, PK)
- name (TEXT)
- email (CITEXT, UNIQUE)
- password_hash (TEXT, bcrypt con salt)
- role (VARCHAR, CHECK: operador|despachador|admin)
- active (BOOLEAN, default: true)
- created_at, updated_at
```

**usuario.otp_requests**
```
- id (UUID, PK)
- phone_e164 (TEXT, FK → citizens.phone_e164)
- provider (TEXT, default: 'twilio')
- status (VARCHAR, CHECK: requested|verified|failed|expired)
- attempts (INT, default: 0, max: 5)
- requested_at (TIMESTAMPTZ, default: now())
- verified_at (TIMESTAMPTZ, NULL hasta verificación)
- error_code (TEXT, ej: 'RATE_LIMIT_EXCEEDED', 'OTP_EXPIRED')
```

**usuario.idempotency_keys**
```
- key (TEXT, PK, client-provided UUID)
- request_fingerprint (TEXT, hash del request)
- response_payload (JSONB, respuesta guardada)
- created_at (TIMESTAMPTZ)
```

**usuario.outbox_events**
```
- id (UUID, PK)
- aggregate_type (TEXT, 'citizen' | 'operator')
- aggregate_id (UUID, FK a citizens.id o operators.id)
- type (TEXT, ej: 'citizen.verified', 'operator.created')
- payload (JSONB)
- status (VARCHAR, CHECK: pending|published|failed)
- created_at (TIMESTAMPTZ)
- published_at (TIMESTAMPTZ, NULL hasta publicación)
```

## 🔐 Flujos de Autenticación

### 1. Ciudadanos (OTP)

```
POST /api/v1/auth/otp/send
├── Validar teléfono E.164
├── Rate limiting: máx 3 solicitudes/minuto
├── Crear citizen si no existe (phone_e164 UNIQUE)
├── Generar código OTP (6 dígitos)
├── Crear OTPRequest con status='requested'
├── Enviar vía Twilio
├── Crear evento outbox: citizen.otp_requested
└── Responder: { message: "OTP enviado", expires_in: 300 }

POST /api/v1/auth/otp/verify
├── Validar request.phone_e164
├── Buscar OTPRequest más reciente
├── Verificar status != 'verified' y != 'expired'
├── Validar código (max 5 intentos)
├── Si válido:
│   ├── Marcar OTPRequest.verified_at = now()
│   ├── Actualizar OTPRequest.status = 'verified'
│   ├── Actualizar citizen.verified_at = now()
│   ├── Crear evento outbox: citizen.verified
│   ├── Generar JWT con scope='novedades:crear'
│   └── Responder: { token, expires_in }
└── Si inválido:
    ├── Incrementar attempts
    ├── Si attempts > 5: status='failed', error_code='MAX_ATTEMPTS_EXCEEDED'
    └── Rechazar (429 si rate-limited, 401 si inválido)
```

### 2. Operadores (Email/Contraseña)

```
POST /api/v1/auth/register
├── Validar email (formato y unicidad)
├── Validar contraseña (min 8 chars, complejidad)
├── Validar role (operador|despachador|admin)
├── Hash contraseña con bcrypt (salt rounds: 10)
├── Crear Operator
├── Crear evento outbox: operator.created
└── Responder: { id, name, email, role, created_at }

POST /api/v1/auth/login
├── Validar email y contraseña presentes
├── Buscar operator por email
├── Verificar active = true
├── Comparar contraseña con bcrypt
├── Generar JWT:
│   ├── sub = operator.id
│   ├── email = operator.email
│   ├── role = operator.role
│   ├── scope = determinar por role (ver tabla abajo)
│   ├── iat = now()
│   └── exp = now() + 8 horas
├── Crear evento outbox: operator.login
└── Responder: { token, expires_in: 28800 }
```

## 📊 Roles y Scopes

| Role | Scopes | Descripción |
|------|--------|-------------|
| operador | tareas:gestionar, reportes:leer | Gestiona tareas asignadas |
| despachador | tareas:asignar, reportes:leer, reportes:actualizar | Asigna tareas a operadores |
| admin | * | Acceso total a recursos |

## 📋 Request/Response DTOs

### Citizen OTP

**POST /api/v1/auth/otp/send**
```json
{
  "phone_e164": "+593999000111"
}
```

**Respuesta (200):**
```json
{
  "message": "OTP enviado a +593999000111",
  "phone_e164": "+593999000111",
  "expires_in": 300
}
```

**POST /api/v1/auth/otp/verify**
```json
{
  "phone_e164": "+593999000111",
  "code": "123456"
}
```

**Respuesta (200):**
```json
{
  "token": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
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

### Operator

**POST /api/v1/auth/register**
```json
{
  "name": "Juan García",
  "email": "juan@ejemplo.com",
  "password": "SecurePass123!",
  "role": "operador"
}
```

**POST /api/v1/auth/login**
```json
{
  "email": "juan@ejemplo.com",
  "password": "SecurePass123!"
}
```

**Respuesta (200):**
```json
{
  "token": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
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

## 🛡️ Validaciones

### Teléfono (E.164)
```regex
^+[1-9]\d{7,14}$
```
Ejemplos válidos: `+593999000111`, `+34911234567`

### Contraseña (Operadores)
- Mínimo 8 caracteres
- Al menos 1 mayúscula
- Al menos 1 número
- Al menos 1 carácter especial

### Email
- Formato RFC 5322
- Único en la tabla operators

### Código OTP
- Exactamente 6 dígitos
- Válido por 5 minutos (300 segundos)
- Máximo 5 intentos

## 🔄 Idempotencia

Para endpoints sensibles, enviar header:
```
Idempotency-Key: <UUID>
```

El servidor:
1. Calcula fingerprint del request: SHA256(method + path + body)
2. Busca en idempotency_keys
3. Si existe y fingerprint coincide: devuelve response guardada
4. Si no existe: procesa y guarda response por 24 horas

## 📤 Event Sourcing (Outbox Pattern)

**Eventos generados:**

| Evento | Agregado | Payload |
|--------|----------|---------|
| citizen.otp_requested | citizen.id | { phone_e164, attempts: 0 } |
| citizen.verified | citizen.id | { phone_e164, verified_at } |
| operator.created | operator.id | { name, email, role } |
| operator.login | operator.id | { email, timestamp } |
| operator.role_changed | operator.id | { previous_role, new_role } |

**Procesamiento:**
1. Al crear evento, status = 'pending'
2. Publicador: Lee desde outbox_events WHERE status='pending' cada 5s
3. Publica a message broker (Kafka/RabbitMQ)
4. Al confirmar: actualiza status='published', published_at=now()
5. Si falla: reintenta 3 veces, luego status='failed' para revisión manual

## 🧪 Pruebas (BDD)

Escenarios en `bdd/features/auth_service_complete.feature`:

```bash
# Ejecutar todas las pruebas
go test -v ./bdd/...

# Ejecutar solo ciudadanos
go test -v ./bdd/... -tags "@ciudadano"

# Ejecutar solo operadores
go test -v ./bdd/... -tags "@operadores"
```

## 🚀 Implementación por Fases

### Fase 1: Operadores (Email/Contraseña)
- [ ] POST /api/v1/auth/register
- [ ] POST /api/v1/auth/login
- [ ] JWT token generation
- [ ] Role-based access control

### Fase 2: Ciudadanos (OTP)
- [ ] POST /api/v1/auth/otp/send
- [ ] POST /api/v1/auth/otp/verify
- [ ] Rate limiting (3 req/min)
- [ ] OTP expiration (5 min)

### Fase 3: Idempotencia y Eventos
- [ ] Idempotency-Key handling
- [ ] Outbox event creation
- [ ] Event publisher worker

### Fase 4: Integración
- [ ] Middleware de autenticación
- [ ] Middleware de autorización (roles)
- [ ] Refresh tokens (opcional)
- [ ] Rate limiting por usuario

## 📌 Notas Importantes

1. **Hash de contraseñas**: Usar bcrypt, nunca almacenar en texto plano
2. **Teléfono**: Siempre almacenar en formato E.164
3. **Eventos**: NO eliminar de outbox, solo marcar como publicados
4. **JWT**: Firmar con HS256 y clave segura de la variable ENV
5. **CORS**: Ya habilitado en server.go
6. **Timezone**: Usar TIMESTAMPTZ en BD, siempre UTC en código

## 🔗 Relaciones entre Servicios

```
auth-service (ciudadanos verificados)
    ↓
report-service (crea reportes)
    ↓
task-service (crea tareas para operadores)
    ↓
Operadores del auth-service (ejecutan tareas)
```
