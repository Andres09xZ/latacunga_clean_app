# Quick Reference - Auth Service

## 🎯 Endpoints Rápidos

### Ciudadanos (OTP)
```bash
# Solicitar OTP
POST /api/v1/auth/otp/send
Body: { "phone_e164": "+593999000111" }
Response: { "message": "OTP enviado", "expires_in": 300 }

# Verificar OTP
POST /api/v1/auth/otp/verify
Body: { "phone_e164": "+593999000111", "code": "123456" }
Response: { "token": {...}, "citizen": {...} }
```

### Operadores
```bash
# Registrar
POST /api/v1/auth/register
Body: { "name": "Juan", "email": "juan@ejemplo.com", "password": "SecurePass123!", "role": "operador" }
Response: { "id": "...", "email": "juan@ejemplo.com" }

# Login
POST /api/v1/auth/login
Body: { "email": "juan@ejemplo.com", "password": "SecurePass123!" }
Response: { "token": {...}, "operator": {...} }
```

## 📊 Estructura de Datos

### Citizen (Ciudadano)
```
ID          UUID    (PRIMARY KEY)
PhoneE164   TEXT    (UNIQUE, E.164 format: +593999000111)
VerifiedAt  TIME    (NULL = no verificado)
CreatedAt   TIME
UpdatedAt   TIME
```

### Operator (Operador)
```
ID           UUID     (PRIMARY KEY)
Name         TEXT
Email        CITEXT   (UNIQUE)
PasswordHash TEXT     (bcrypt)
Role         VARCHAR  (operador|despachador|admin)
Active       BOOLEAN  (default: true)
CreatedAt    TIME
UpdatedAt    TIME
```

### OTPRequest
```
ID           UUID     (PRIMARY KEY)
PhoneE164    TEXT     (FK → citizens)
Provider     TEXT     (default: 'twilio')
Status       VARCHAR  (requested|verified|failed|expired)
Attempts     INT      (0-5)
RequestedAt  TIME     (default: now())
VerifiedAt   TIME     (NULL hasta verificación)
ErrorCode    VARCHAR  (ej: RATE_LIMIT_EXCEEDED)
```

### OutboxEvent
```
ID            UUID     (PRIMARY KEY)
AggregateType TEXT     (citizen|operator)
AggregateID   UUID     (FK)
Type          TEXT     (ej: citizen.verified, operator.created)
Payload       JSONB    (event data)
Status        VARCHAR  (pending|published|failed)
CreatedAt     TIME
PublishedAt   TIME     (NULL hasta publicación)
```

## 🔐 JWT Claims

```json
{
  "sub": "operator-uuid",      // Subject (user ID)
  "email": "juan@ejemplo.com", // Email
  "role": "operador",          // Role: operador|despachador|admin
  "scope": "tareas:gestionar", // Permissions
  "iat": 1234567890,           // Issued at
  "exp": 1234596690            // Expires (8 horas después)
}
```

## 🔄 Flujos de Datos

### OTP Verification Flow
```
User Phone: +593999000111
     ↓
[OTP Service]
  ├─ Create Citizen record
  ├─ Generate 6-digit code
  ├─ Create OTPRequest (status: requested)
  ├─ Send via Twilio
  └─ Create Event (citizen.otp_requested)
     ↓
[User receives SMS]
     ↓
[Verify Endpoint]
  ├─ Validate code (max 5 attempts)
  ├─ Check expiration (5 min)
  ├─ Update OTPRequest (status: verified)
  ├─ Update Citizen (verified_at: now())
  ├─ Create Event (citizen.verified)
  └─ Generate JWT (scope: novedades:crear)
     ↓
[JWT Token with scope novedades:crear]
     ↓
[Can POST to report-service]
```

### Operator Login Flow
```
Email + Password
     ↓
[Login Endpoint]
  ├─ Find Operator by email
  ├─ Verify active = true
  ├─ Compare password (bcrypt)
  ├─ Generate JWT (HS256, 8h exp)
  ├─ Create Event (operator.login)
  └─ Return token + operator info
     ↓
[JWT Token with role scope]
     ↓
[Can GET/POST task-service]
```

## 📏 Validaciones

### Phone (E.164)
```regex
^+[1-9]\d{7,14}$

Ejemplos válidos:
✅ +593999000111     (Ecuador 9 dígitos)
✅ +34911234567      (España 9 dígitos)
✅ +12125552368      (USA)
✅ +552133334444     (Brazil)

Inválidos:
❌ 593999000111      (falta +)
❌ +0999000111       (no puede empezar con 0)
❌ +5939990001       (muy corto)
```

### Email
```regex
RFC 5322 format
Ejemplos: juan@ejemplo.com, maria.garcia@empresa.com
```

### Password
```
Mínimo 8 caracteres
Debe contener:
- Al menos 1 MAYÚSCULA
- Al menos 1 número
- Al menos 1 carácter especial (!@#$%^&*)

Ejemplos:
✅ SecurePass123!
✅ MyP@ssw0rd
❌ password        (sin mayúscula, número, especial)
❌ Pass123         (sin especial, muy corto)
```

### OTP Code
```
Exactamente 6 dígitos numéricos
Válido por 5 minutos (300 segundos)
Máximo 5 intentos
```

## 🚨 Error Codes

| Status | Error | Meaning | Action |
|--------|-------|---------|--------|
| 400 | INVALID_PHONE | Teléfono no es E.164 | Validar formato |
| 400 | INVALID_EMAIL | Email inválido | Validar RFC 5322 |
| 400 | WEAK_PASSWORD | Contraseña débil | Aumentar complejidad |
| 401 | INVALID_CREDENTIALS | Email/contraseña incorrecta | Retentar |
| 401 | OTP_INVALID | Código OTP incorrecto | Retentar |
| 401 | OTP_EXPIRED | Código OTP expirado (>5 min) | Solicitar nuevo |
| 403 | ACCOUNT_INACTIVE | Operador desactivado | Contactar admin |
| 409 | EMAIL_EXISTS | Email ya registrado | Usar otro email |
| 429 | RATE_LIMIT_EXCEEDED | Demasiadas solicitudes | Esperar 1 minuto |
| 503 | TWILIO_ERROR | Error al enviar SMS | Retentar más tarde |

## 🏷️ Roles y Permisos

### operador
```
Scope: tareas:gestionar reportes:leer
Puede:
  ✅ Ver sus tareas asignadas
  ✅ Actualizar estado de tarea
  ✅ Leer reportes (read-only)
No puede:
  ❌ Crear operadores
  ❌ Asignar tareas
  ❌ Modificar reportes
```

### despachador
```
Scope: tareas:asignar reportes:leer reportes:actualizar
Puede:
  ✅ Asignar tareas a operadores
  ✅ Ver y actualizar reportes
  ✅ Ver tareas
No puede:
  ❌ Crear operadores
  ❌ Cambiar roles
```

### admin
```
Scope: * (todo)
Puede:
  ✅ Todo: crear operadores, asignar roles, etc.
```

## 📈 Rate Limits

| Recurso | Límite | Ventana | Respuesta |
|---------|--------|---------|-----------|
| OTP Send | 3 solicitudes | 1 minuto | 429 Too Many |
| OTP Verify | 5 intentos | Por solicitud | 429 Too Many |
| Login | Sin límite | - | Validación normal |
| Register | Sin límite | - | Validación normal |

## 💾 Base de Datos

### Conexión
```bash
psql postgresql://user:password@host:5432/latacunga_clean
```

### Tablas del Auth Service
```sql
\d usuario.*

-- Tablas:
-- usuario.operators        - Operadores internos
-- usuario.citizens         - Ciudadanos OTP
-- usuario.otp_requests     - Solicitudes OTP
-- usuario.idempotency_keys - Prevención duplicados
-- usuario.outbox_events    - Event sourcing
```

### Ver datos rápido
```sql
-- Operadores
SELECT email, role, active FROM usuario.operators;

-- Ciudadanos
SELECT phone_e164, verified_at FROM usuario.citizens;

-- OTP recientes
SELECT phone_e164, status, requested_at 
FROM usuario.otp_requests 
ORDER BY requested_at DESC LIMIT 10;

-- Eventos pendientes
SELECT type, aggregate_id, created_at 
FROM usuario.outbox_events 
WHERE status = 'pending';
```

## 🔍 Debug Checklist

- [ ] `.env` contiene `JWT_SECRET` (mínimo 32 chars)
- [ ] `.env` contiene `DB_URL` válido
- [ ] `.env` contiene `TWILIO_*` si usa OTP
- [ ] Base de datos conectada: `psql $DB_URL -c "SELECT 1"`
- [ ] Tablas creadas: `psql $DB_URL -c "\d usuario.operators"`
- [ ] Server inicia sin errores: `go run cmd/server/main.go`
- [ ] Swagger accesible: http://localhost:8080/swagger/index.html
- [ ] Endpoint responde: `curl http://localhost:8080/api/v1/health`

## 🧪 Test Commands

```bash
# Verificar compilación
go build ./...

# Ejecutar tests
go test ./... -v

# BDD tests
go test ./bdd/... -v

# Con coverage
go test ./... -cover

# Específico
go test -run TestOperatorLogin ./... -v
```

## 📱 Ejemplos cURL

```bash
# Solicitar OTP
curl -X POST http://localhost:8080/api/v1/auth/otp/send \
  -H "Content-Type: application/json" \
  -d '{"phone_e164": "+593999000111"}' | jq

# Registrar operador
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Juan",
    "email": "juan@ejemplo.com",
    "password": "SecurePass123!",
    "role": "operador"
  }' | jq

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "juan@ejemplo.com",
    "password": "SecurePass123!"
  }' | jq

# Usar token
curl -X GET http://localhost:8080/api/v1/admin/reports \
  -H "Authorization: Bearer eyJhbGc..." | jq
```

## 🔗 Enlaces Útiles

- **Documentación Completa**: `IMPLEMENTATION_GUIDE.md`
- **README**: `README_NEW.md`
- **BDD Features**: `bdd/features/auth_service_complete.feature`
- **SQL Queries**: `SQL_TESTING_GUIDE.md`
- **Changelog**: `CHANGELOG.md`

---

**Última actualización**: 12 Noviembre 2025
**Versión**: 2.0 (Citizens OTP + Operators)
**Status**: 🟡 En desarrollo (esquema completado, handlers pendientes)
