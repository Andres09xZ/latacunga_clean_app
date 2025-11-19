# Validation Service

Servicio de validación de incidencias con patrón CQRS, eventos y RabbitMQ.

## Características

- ✅ Validación manual de incidencias (mark-valid, reject)
- ✅ Consumo de eventos de creación de incidencias
- ✅ Publicación de eventos de validación
- ✅ Idempotencia con idempotency keys
- ✅ JWT Authentication con roles (operador, admin, validador)
- ✅ Swagger/OpenAPI documentation
- ✅ Automatic database migrations

## Endpoints

### Validar Incidencia
```bash
POST /api/v1/validations/{incident_id}/mark-valid
Authorization: Bearer <token>
Content-Type: application/json

# Response: 200
{
  "id": "uuid",
  "incident_id": "INC-123",
  "status": "valida",
  "validator_kind": "manual",
  "decided_at": "2024-01-01T12:00:00Z"
}
```

### Rechazar Incidencia
```bash
POST /api/v1/validations/{incident_id}/reject
Authorization: Bearer <token>
Content-Type: application/json

{
  "reason": "imagen no pertinente"
}

# Response: 200
{
  "id": "uuid",
  "incident_id": "INC-123",
  "status": "rechazada",
  "reason": "imagen no pertinente",
  "validator_kind": "manual",
  "decided_at": "2024-01-01T12:00:00Z"
}
```

## RabbitMQ Events

### Escucha
- **Queue**: validation.q
- **Exchange**: waste.events (topic)
- **Routing Key**: incidencia_creada

### Publica
- **Exchange**: waste.events (topic)
- **Routing Keys**: 
  - incidencia_validada
  - incidencia_rechazada

## Base de Datos

**Schema**: validacion

**Tablas**:
- `validations` - Auditoría de validaciones
- `idempotency_keys` - Prevención de duplicados
- `outbox_events` - Eventos a publicar

## Configuración

```bash
cp cmd/server/.env.example cmd/server/.env
# Editar variables según tu entorno
```

## Ejecución

```bash
go run ./cmd/server/main.go
```

Servidor en: http://localhost:8082
Swagger: http://localhost:8082/swagger/index.html

## Testing

```bash
go test ./...
```

## Validación BDD

```bash
go test -v -tags=bdd ./bdd
```
