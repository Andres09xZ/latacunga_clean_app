# Fleet Service - Quick API Reference

## 🚀 Base URL
```
http://localhost:8082/api/v1
```

---

## 🚛 Trucks (Camiones)

### Create Truck
```bash
POST /trucks
Content-Type: application/json

{
  "plate": "GYE-1234",
  "type": "CARGA_LATERAL",    # CARGA_LATERAL | CARGA_POSTERIOR
  "status": "DISPONIBLE"       # DISPONIBLE | EN_USO | MANTENIMIENTO (opcional, default: DISPONIBLE)
}

Response: 201 Created
{
  "id": 1,
  "plate": "GYE-1234",
  "type": "CARGA_LATERAL",
  "status": "DISPONIBLE",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### List Trucks
```bash
GET /trucks
GET /trucks?status=DISPONIBLE
GET /trucks?type=CARGA_LATERAL
GET /trucks?status=DISPONIBLE&type=CARGA_POSTERIOR

Response: 200 OK
{
  "trucks": [
    {
      "id": 1,
      "plate": "GYE-1234",
      "type": "CARGA_LATERAL",
      "status": "DISPONIBLE",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1
}
```

### Get Truck by ID
```bash
GET /trucks/1

Response: 200 OK
{
  "id": 1,
  "plate": "GYE-1234",
  "type": "CARGA_LATERAL",
  "status": "DISPONIBLE",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Update Truck
```bash
PUT /trucks/1
Content-Type: application/json

{
  "status": "MANTENIMIENTO"    # Actualiza solo el status
}

# También se puede actualizar:
{
  "plate": "GYE-5678",         # Cambiar placa
  "type": "CARGA_POSTERIOR",   # Cambiar tipo
  "status": "DISPONIBLE"       # Cambiar estado
}

Response: 200 OK
{
  "id": 1,
  "plate": "GYE-1234",
  "type": "CARGA_LATERAL",
  "status": "MANTENIMIENTO",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T14:25:00Z"
}
```

### Delete Truck
```bash
DELETE /trucks/1

Response: 200 OK
{
  "message": "Camión eliminado exitosamente"
}

# Error si tiene turnos activos
Response: 400 Bad Request
{
  "error": "no se puede eliminar el camión: tiene 1 turno(s) activo(s)"
}
```

---

## 👤 Shifts (Turnos)

### Clock In
```bash
POST /shifts/clock-in
Content-Type: application/json

{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "truck_plate": "GYE-1234"
}

Response: 200 OK
{
  "shift_id": "650e8400-e29b-41d4-a716-446655440099",
  "message": "Turno iniciado exitosamente"
}
```

### Clock Out
```bash
POST /shifts/clock-out
Content-Type: application/json

{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000"
}

Response: 200 OK
{
  "message": "Turno finalizado exitosamente"
}
```

---

## 📊 Error Responses

### 400 Bad Request
```json
{
  "error": "Datos inválidos: plate is required"
}
```

### 404 Not Found
```json
{
  "error": "Camión con ID 999 no encontrado"
}
```

### 409 Conflict
```json
{
  "error": "ya existe un camión con la placa 'GYE-1234'"
}
```

### 500 Internal Server Error
```json
{
  "error": "Error interno del servidor"
}
```

---

## 🔄 RabbitMQ Events

### Operator Registration (Consumed by IdentityConsumer)
```
Exchange: city.cleaning.identity
RoutingKey: identity.operator.created.v1

Payload:
{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "full_name": "Juan Pérez",
  "preferred_zone_id": 3,
  "can_drive_lateral": true,
  "can_drive_posterior": false
}

Result: Auto-creates Driver + OperatorProfile
```

### Fleet RPC Request (Consumed by FleetRPCConsumer)
```
Queue: q.fleet.rpc
RoutingKey: fleet.resource.request

Request:
{
  "zone_id": 3,
  "required_type": "CARGA_LATERAL"
}

Response (to ReplyTo queue):
{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "assistant_id": null,
  "truck_plate": "GYE-1234",
  "status": "ALLOCATED"
}
```

### Fleet Release (Compensation)
```
Queue: q.fleet.rpc
RoutingKey: fleet.resource.release

Payload:
{
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "truck_plate": "GYE-1234",
  "reason": "routing_failed"
}

Result: Driver status → DISPONIBLE, Truck status → DISPONIBLE
```

---

## 🧪 Testing Workflow

### 1. Create a Truck
```bash
curl -X POST http://localhost:8082/api/v1/trucks \
  -H "Content-Type: application/json" \
  -d '{"plate":"GYE-1234","type":"CARGA_LATERAL"}'
```

### 2. Register an Operator (auth-service)
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phone":"+593987654321",
    "full_name":"Juan Pérez",
    "role":"OPERATOR",
    "preferred_zone_id":3,
    "can_drive_lateral":true
  }'
```
**Note:** This publishes an event that fleet-service consumes to create Driver + OperatorProfile

### 3. Clock In
```bash
curl -X POST http://localhost:8082/api/v1/shifts/clock-in \
  -H "Content-Type: application/json" \
  -d '{
    "driver_id":"550e8400-e29b-41d4-a716-446655440000",
    "truck_plate":"GYE-1234"
  }'
```

### 4. Trigger Schedule Orchestrator
```bash
# Create incidents in zone 3 until score > threshold
# Orchestrator will automatically request fleet via RPC
# Fleet-service will allocate the driver and truck
```

### 5. Clock Out
```bash
curl -X POST http://localhost:8082/api/v1/shifts/clock-out \
  -H "Content-Type: application/json" \
  -d '{"driver_id":"550e8400-e29b-41d4-a716-446655440000"}'
```

---

## 📚 Documentation

- **Swagger UI:** http://localhost:8082/swagger/index.html
- **Health Check:** http://localhost:8082/health
- **Full Guide:** FLEET_VEHICLE_MANAGEMENT.md
- **RPC Implementation:** IMPLEMENTACION_RPC_COMPLETA.md

---

## 🐛 Common Issues

### "ya existe un camión con la placa..."
- Each plate must be unique
- Use different plates or delete the existing truck first

### "no se puede eliminar el camión: tiene turnos activos"
- Clock out the driver first
- Or wait until the shift ends

### "El camión no está disponible (Estado: EN_USO)"
- Truck is currently assigned to an active shift
- Use a different truck or wait until it's available

### "Conductor no encontrado"
- The driver hasn't been created yet
- Register the operator in auth-service first
- Wait for IdentityConsumer to process the event

---

**Last Updated:** January 2024
