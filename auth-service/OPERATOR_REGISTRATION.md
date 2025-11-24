# Registro de Operadores - Auth Service

## Endpoint de Registro de Operadores

### POST /api/v1/admin/operators

Crea un nuevo operador con perfil completo incluyendo licencia, zona preferida y capacidades de vehículos.

**Autenticación**: Requiere JWT token con rol "admin"

#### Request Body

```json
{
  "full_name": "Juan Pérez García",
  "username": "jperez",
  "password": "SecurePass123!",
  "license_id": "LIC-2024-001",
  "preferred_zone_id": 3,
  "can_drive_lateral": true,
  "can_drive_compactor": false,
  "email": "juan.perez@example.com",
  "role": "operador"
}
```

#### Campos

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `full_name` | string | Sí | Nombre completo del operador (mínimo 2 caracteres) |
| `username` | string | Sí | Nombre de usuario único (mínimo 3 caracteres) |
| `password` | string | Sí | Contraseña (mínimo 8 caracteres) |
| `license_id` | string | Sí | ID de la licencia de conducir |
| `preferred_zone_id` | integer | No | ID de la zona preferida de trabajo |
| `can_drive_lateral` | boolean | No | Puede conducir vehículos laterales (default: false) |
| `can_drive_compactor` | boolean | No | Puede conducir compactadores (default: false) |
| `email` | string | No | Correo electrónico del operador |
| `role` | string | Sí | Debe ser "operador" |

#### Response Success (201 Created)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "full_name": "Juan Pérez García",
  "username": "jperez",
  "email": "juan.perez@example.com",
  "role": "operador",
  "license_id": "LIC-2024-001",
  "preferred_zone_id": 3,
  "can_drive_lateral": true,
  "can_drive_compactor": false,
  "active": true,
  "created_at": "2025-01-23T19:00:00Z"
}
```

#### Response Error (400 Bad Request)

```json
{
  "error": "Username already exists"
}
```

#### Response Error (401 Unauthorized)

```json
{
  "error": "Invalid or missing token"
}
```

#### Response Error (403 Forbidden)

```json
{
  "error": "Admin role required"
}
```

## Eventos RabbitMQ

Cuando se crea un operador exitosamente, se publica un evento a RabbitMQ:

**Exchange**: `city.cleaning.identity`
**Routing Key**: `identity.operator.created.v1`
**Payload**:

```json
{
  "event_type": "operator.created",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "jperez",
  "full_name": "Juan Pérez García",
  "license_id": "LIC-2024-001",
  "preferred_zone_id": 3,
  "can_drive_lateral": true,
  "can_drive_compactor": false,
  "email": "juan.perez@example.com",
  "event_timestamp": "2025-01-23T19:00:00Z"
}
```

## Configuración

Asegúrate de tener configuradas las siguientes variables de entorno:

```env
DB_URL=postgresql://user:password@host:port/database
JWT_SECRET=tu_secreto_seguro
PORT=8080
RABBITMQ_URL=amqp://user:password@localhost:5672/
```

## Base de Datos

### Tabla: users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(20) UNIQUE,
    password_hash VARCHAR(128),
    role VARCHAR(50) NOT NULL,
    display_name VARCHAR(255),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Tabla: operator_profiles

```sql
CREATE TABLE operator_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(200) NOT NULL,
    username VARCHAR(100) NOT NULL UNIQUE,
    license_id VARCHAR(50) NOT NULL,
    preferred_zone_id INTEGER,
    can_drive_lateral BOOLEAN DEFAULT FALSE,
    can_drive_compactor BOOLEAN DEFAULT FALSE,
    badge_id VARCHAR(50),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## Ejemplo de Uso con cURL

```bash
# 1. Primero, hacer login como admin
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin123!"
  }'

# Respuesta incluye access_token
# {
#   "access_token": "eyJhbGc...",
#   "refresh_token": "eyJhbGc..."
# }

# 2. Registrar nuevo operador
curl -X POST http://localhost:8080/api/v1/admin/operators \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGc..." \
  -d '{
    "full_name": "Juan Pérez García",
    "username": "jperez",
    "password": "SecurePass123!",
    "license_id": "LIC-2024-001",
    "preferred_zone_id": 3,
    "can_drive_lateral": true,
    "can_drive_compactor": false,
    "email": "juan.perez@example.com",
    "role": "operador"
  }'
```

## Swagger UI

La documentación interactiva está disponible en:

```
http://localhost:8080/swagger/index.html
```

## Validaciones

- **username**: Debe ser único en la tabla `operator_profiles`
- **password**: Se hashea con bcrypt antes de almacenar
- **role**: Solo acepta "operador" para este endpoint
- **Transacción**: La creación del usuario y perfil se hace en una transacción atómica

## Integración con Fleet Service

El Fleet Service puede suscribirse al evento `identity.operator.created.v1` para:
- Crear el registro del operador en su base de datos local
- Sincronizar información de capacidades de conducción
- Asignar el operador a zonas según su preferencia

## Testing

Para probar el endpoint:

1. Asegúrate de que RabbitMQ esté corriendo:
```bash
docker-compose up -d rabbitmq
```

2. Inicia el servicio:
```bash
cd auth-service
go run cmd/server/main.go
```

3. Verifica que las tablas se crearon correctamente en PostgreSQL

4. Usa la colección de Postman o cURL para probar el endpoint
