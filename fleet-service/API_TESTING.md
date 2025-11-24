# Fleet Service - API Testing with curl

## 1. Health Check

```bash
curl -X GET http://localhost:8082/health
```

## 2. Clock-In (Inicio de Turno)

```bash
curl -X POST http://localhost:8082/api/v1/shifts/clock-in \
  -H "Content-Type: application/json" \
  -d '{
    "driver_id": "550e8400-e29b-41d4-a716-446655440000",
    "truck_plate": "ABC-1234"
  }'
```

**Respuesta esperada:**
```json
{
  "shift_id": "uuid-del-turno",
  "message": "Turno iniciado exitosamente"
}
```

## 3. Clock-Out (Fin de Turno)

```bash
curl -X POST http://localhost:8082/api/v1/shifts/clock-out \
  -H "Content-Type: application/json" \
  -d '{
    "driver_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

**Respuesta esperada:**
```json
{
  "message": "Turno finalizado exitosamente"
}
```

## Errores Comunes

### Conductor no encontrado (404)
```json
{
  "error": "Conductor no encontrado"
}
```

### Conductor ya tiene turno activo (409)
```json
{
  "error": "El conductor ya tiene un turno activo"
}
```

### Camión no disponible (409)
```json
{
  "error": "El camión no está disponible (Estado: EN_USO)"
}
```

### Camión no encontrado (404)
```json
{
  "error": "Camión no encontrado"
}
```

## Testing con PowerShell (Windows)

### Clock-In
```powershell
$body = @{
    driver_id = "550e8400-e29b-41d4-a716-446655440000"
    truck_plate = "ABC-1234"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8082/api/v1/shifts/clock-in" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body
```

### Clock-Out
```powershell
$body = @{
    driver_id = "550e8400-e29b-41d4-a716-446655440000"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8082/api/v1/shifts/clock-out" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body
```

## Simulación de Eventos RabbitMQ

### 1. Crear Operador (Identity Service)
Publicar a exchange `identity.management` con routing key `identity.operator.created.v1`:

```json
{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "full_name": "Juan Pérez",
  "email": "juan.perez@example.com",
  "preferred_zone_id": 1,
  "can_drive_lateral": true,
  "can_drive_posterior": false
}
```

### 2. Solicitar Recurso (Scheduler Service)
Publicar a exchange `planning.scheduler` con routing key `planning.resource.requested.v1`:

```json
{
  "request_id": "req-123",
  "zone_id": 1,
  "truck_type": "CARGA_LATERAL",
  "route_id": "route-456",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 3. Completar Orden de Trabajo (Operations Service)
Publicar a exchange `operations.workorders` con routing key `workorder.completed.v1`:

```json
{
  "workorder_id": "wo-789",
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "route_id": "route-456",
  "status": "COMPLETED",
  "completed_at": "2024-01-15T12:45:00Z"
}
```

## Verificación en Base de Datos

### Ver estado de conductores
```sql
SELECT id, full_name, status FROM drivers;
```

### Ver turnos activos
```sql
SELECT * FROM active_shifts WHERE is_active = true;
```

### Ver camiones disponibles
```sql
SELECT plate, type, status FROM trucks WHERE status = 'DISPONIBLE';
```
