Validation Service
===================

**Microservicio desacoplado con base de datos propia para validación manual de incidentes.**

Este servicio almacena localmente todos los incidentes reportados y permite a administradores validarlos/rechazarlos manualmente sin depender del incident-service.

## Arquitectura Desacoplada

✅ **Base de datos propia** - Esquema `validacion` en PostgreSQL
✅ **Sin llamadas HTTP** - Comunicación 100% via RabbitMQ
✅ **Event-driven** - Consume y publica eventos
✅ **Independiente** - Puede funcionar sin incident-service activo

Features
--------
- **Base de datos local**: Tabla `validacion.incidentes_pendientes`
- **GET /api/v1/incidents/pending** - Lista incidentes desde BD local
- **POST /api/v1/incidents/:id/validate** - Valida/rechaza y actualiza BD local + publica evento
- **RabbitMQ Consumer** - Escucha `incidents.submitted.v1` y guarda en BD local
- **RabbitMQ Producer** - Publica `incidents.validated.v1` cuando se valida
- **Swagger UI** - Documentación interactiva en `/swagger/index.html`

Configuration
-------------
Variables de entorno en `.env`:
- `DB_URL` - URL de conexión a PostgreSQL (required)
- `RABBITMQ_URL` - URL de conexión a RabbitMQ (default: `amqp://tesis:tesis@localhost:5672/`)
- `PORT` - Puerto HTTP (default: `8082`)

Database Setup
--------------
1. Ejecutar migración para crear esquema y tabla:

```powershell
cd "D:\Octavo Semestre\Tesis\backend_latacunga_clean\incident-service\validation-service"
go run migrate.go
```

Esto creará:
- Esquema: `validacion`
- Tabla: `validacion.incidentes_pendientes`
- Índices en: estado, fecha_evento, tipo
- Trigger: actualizar_timestamp

Run locally
-----------
1. Asegurar que RabbitMQ esté corriendo

2. Configurar `.env` con credenciales de BD

3. Ejecutar migración (primera vez):
```powershell
go run migrate.go
```

4. Iniciar validation-service:

```powershell
go mod tidy
go run main.go
```

5. Abrir Swagger: http://localhost:8082/swagger/index.html

API Endpoints
-------------

### Listar incidentes pendientes (desde BD local)
```powershell
curl http://localhost:8082/api/v1/incidents/pending
```

Respuesta:
```json
{
  "success": true,
  "count": 2,
  "data": [
    {
      "incidente_id": "72bed5f5-5539-4422-a1cf-14eadc41cdf8",
      "tipo": "punto_acopio",
      "descripcion": "Basura acumulada",
      "ciudadano_id": "550e8400-e29b-41d4-a716-446655440000",
      "estado": "pendiente_validacion",
      "fecha_evento": "2025-11-20T23:10:00Z",
      "dia_incidente": "2025-11-20",
      "num_fotos": 2,
      "direccion": "Calle Principal",
      "latitud": -0.9346,
      "longitud": -78.6156,
      "recibido_en": "2025-11-20T23:10:05Z",
      "actualizado_en": "2025-11-20T23:10:05Z"
    }
  ]
}
```

### Validar un incidente (aprobar)
```powershell
curl -X POST http://localhost:8082/api/v1/incidents/72bed5f5-5539-4422-a1cf-14eadc41cdf8/validate `
  -H "Content-Type: application/json" `
  -d '{\"status\":\"incidente_valido\",\"notes\":\"Verificado por administrador\"}'
```

### Rechazar un incidente
```powershell
curl -X POST http://localhost:8082/api/v1/incidents/72bed5f5-5539-4422-a1cf-14eadc41cdf8/validate `
  -H "Content-Type: application/json" `
  -d '{\"status\":\"incidente_rechazado\",\"notes\":\"Faltan evidencias\"}'
```

Flujo Completo (Desacoplado)
----------------------------

1. **incident-service** crea incidente → Publica `incidents.submitted.v1` a RabbitMQ

2. **validation-service** consume evento → **Guarda en BD local** (tabla `validacion.incidentes_pendientes`)

3. **Administrador** consulta `GET /api/v1/incidents/pending` → **Lee desde BD local** (sin HTTP al incident-service)

4. **Administrador** valida `POST /api/v1/incidents/:id/validate` → **Actualiza BD local** + publica `incidents.validated.v1`

5. **incident-service** consume `incidents.validated.v1` → Actualiza su propia BD

## Ventajas de la arquitectura desacoplada:

✅ **Independencia**: validation-service puede funcionar aunque incident-service esté caído
✅ **Escalabilidad**: Cada servicio gestiona su propia BD
✅ **No HTTP**: Cero dependencias de APIs REST entre servicios
✅ **Event-driven**: Comunicación asíncrona via RabbitMQ
✅ **Auditabilidad**: Historial completo de validaciones en BD local

Events Consumed
---------------
`incidents.submitted.v1` (de incident-service):

```json
{
  "incident_id": "72bed5f5-5539-4422-a1cf-14eadc41cdf8",
  "type": "punto_acopio",
  "description": "Basura acumulada",
  "reporter_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "incidente_no_validado",
  "event_timestamp": "2025-11-20T23:10:00Z",
  "incident_day": "2025-11-20",
  "photos_count": 2,
  "address": "Calle Principal",
  "location": {
    "latitude": -0.9346,
    "longitude": -78.6156
  }
}
```

Events Published
----------------
`incidents.validated.v1` (para incident-service):

```json
{
  "incident_id": "72bed5f5-5539-4422-a1cf-14eadc41cdf8",
  "status": "incidente_valido",
  "validator": "admin-manual",
  "validated_at": "2025-11-20T23:15:02Z",
  "notes": "Incidente verificado por administrador"
}
```

Status válidos:
- `incidente_valido` - Incidente aprobado
- `incidente_rechazado` - Incidente rechazado

Database Schema
---------------
```sql
validacion.incidentes_pendientes
├── incidente_id (UUID, PK)
├── tipo (VARCHAR)
├── descripcion (TEXT)
├── ciudadano_id (UUID)
├── estado (VARCHAR) -- pendiente_validacion, validado, rechazado
├── fecha_evento (TIMESTAMPTZ)
├── dia_incidente (DATE)
├── num_fotos (INTEGER)
├── direccion (TEXT)
├── latitud (DECIMAL)
├── longitud (DECIMAL)
├── geometria (GEOMETRY)
├── validado_por (VARCHAR)
├── fecha_validacion (TIMESTAMPTZ)
├── notas_validacion (TEXT)
├── recibido_en (TIMESTAMPTZ)
└── actualizado_en (TIMESTAMPTZ)
```

Architecture
------------
```
                        RabbitMQ
                           │
                           │
    ┌──────────────────────┼──────────────────────┐
    │                      │                      │
    ▼                      ▼                      ▼
incident-service   validation-service      otros servicios
    │                      │
    │                      │
    ▼                      ▼
PostgreSQL             PostgreSQL
(incidentes)          (validacion)
```

**🔥 100% Desacoplado - Zero HTTP entre servicios**
**📦 Cada servicio gestiona su propia base de datos**
**🚀 Comunicación asíncrona via RabbitMQ**


