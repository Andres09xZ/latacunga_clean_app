# Operations Service

Microservicio para gestionar órdenes de trabajo y el ciclo de vida de recolección de residuos.

## 🎯 Funcionalidad

Este servicio:
- Recibe rutas planificadas del Scheduler vía RabbitMQ
- Convierte rutas en Órdenes de Trabajo (Work Orders)
- Gestiona el progreso de recolección a través de la API REST
- Publica eventos de finalización para Fleet y Scheduler

## 🚀 Inicio Rápido

### Variables de Entorno

```env
DB_URL=postgresql://user:password@host:port/database
PORT=8084
RABBITMQ_URL=amqp://user:password@localhost:5672/
```

### Instalación

```bash
cd operations-service
go mod tidy
go run cmd/server/main.go
```

## 📡 API Endpoints

### Driver Endpoints

#### 1. GET /api/v1/driver/orders/active
Obtiene la orden activa para un conductor.

**Query Parameters:**
- `driver_id` (UUID, required): ID del conductor

**Response 200:**
```json
{
  "id": "uuid",
  "request_id": "scheduler_request_id",
  "zone_id": 1,
  "driver_id": "uuid",
  "truck_plate": "ABC-123",
  "status": "ASIGNADA",
  "route_polyline": "encoded_polyline_string",
  "total_stops": 5,
  "completed_stops": 0,
  "assigned_at": "2025-11-23T10:00:00Z",
  "stops": [
    {
      "id": 1,
      "incident_ref_id": "uuid",
      "latitude": -0.9346,
      "longitude": -78.6158,
      "address": "Calle Principal 123",
      "sequence_order": 1,
      "status": "PENDIENTE"
    }
  ]
}
```

**Response 204:** Sin órdenes activas

#### 2. POST /api/v1/driver/orders/:id/start
Inicia una orden de trabajo.

**Request Body:**
```json
{
  "start_latitude": -0.9346,
  "start_longitude": -78.6158
}
```

**Response 200:**
```json
{
  "message": "Work order started successfully",
  "data": { ... }
}
```

#### 3. POST /api/v1/driver/stops/:id/complete
Marca una parada como completada.

**Request Body:**
```json
{
  "status": "RECOGIDO",
  "comment": "Recolección exitosa"
}
```

**Valores válidos para status:** `RECOGIDO`, `NO_RECOGIDO`

**Response 200:**
```json
{
  "message": "Stop completed successfully",
  "data": { ... }
}
```

#### 4. POST /api/v1/driver/orders/:id/finish
Finaliza una orden de trabajo.

**Request Body:**
```json
{
  "end_latitude": -0.9346,
  "end_longitude": -78.6158
}
```

**Validaciones:**
- Todas las paradas deben estar procesadas
- Estado debe ser EN_PROGRESO

**Response 200:**
```json
{
  "message": "Work order finished successfully",
  "data": { ... }
}
```

**Evento publicado:** `workorder.completed.v1`

## 🔄 Integración RabbitMQ

### Consumer (Entrada)

**Exchange:** `city.cleaning.operations` (topic)
**Queue:** `q.operations.workorders`
**Routing Key:** `workorders.created.v1`

**Payload:**
```json
{
  "request_id": "scheduler_123",
  "zone_id": 1,
  "driver_id": "uuid",
  "truck_plate": "ABC-123",
  "geometry": "encoded_polyline",
  "stops": [
    {
      "id": "incident_uuid",
      "lat": -0.9346,
      "lon": -78.6158,
      "address": "Dirección opcional"
    }
  ]
}
```

**Acción:** Crea WorkOrder + WorkOrderStops con estado inicial `ASIGNADA`

### Producer (Salida)

**Exchange:** `city.cleaning.operations`
**Routing Key:** `workorder.completed.v1`

**Payload:**
```json
{
  "event_type": "workorder.completed",
  "work_order_id": "uuid",
  "driver_id": "uuid",
  "zone_id": 1,
  "status": "COMPLETED",
  "total_stops": 5,
  "completed_stops": 5,
  "event_timestamp": "2025-11-23T15:30:00Z"
}
```

**Consumidores esperados:**
- Fleet Service: Libera al conductor
- Scheduler Service: Resetea el score de la zona

**Routing Key (opcional):** `stop.serviced.v1` - Se publica al completar cada parada

## 🗄️ Modelo de Datos

### work_orders
```sql
CREATE TABLE work_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id VARCHAR NOT NULL,
    zone_id INTEGER NOT NULL,
    driver_id UUID NOT NULL,
    truck_plate VARCHAR NOT NULL,
    status VARCHAR(20) DEFAULT 'ASIGNADA',
    route_polyline TEXT,
    total_stops INTEGER DEFAULT 0,
    completed_stops INTEGER DEFAULT 0,
    assigned_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE INDEX idx_work_orders_driver ON work_orders(driver_id);
CREATE INDEX idx_work_orders_status ON work_orders(status);
CREATE INDEX idx_work_orders_zone ON work_orders(zone_id);
```

### work_order_stops
```sql
CREATE TABLE work_order_stops (
    id SERIAL PRIMARY KEY,
    work_order_id UUID NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    incident_ref_id UUID NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    address VARCHAR,
    sequence_order INTEGER NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDIENTE',
    serviced_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE INDEX idx_stops_work_order ON work_order_stops(work_order_id);
CREATE INDEX idx_stops_status ON work_order_stops(status);
```

## 🧪 Testing con Postman/cURL

### 1. Simular creación de orden (vía RabbitMQ)

Publica un mensaje a la cola `q.operations.workorders`:

```json
{
  "request_id": "test_request_001",
  "zone_id": 1,
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "truck_plate": "ABC-123",
  "geometry": "encoded_polyline_from_osrm",
  "stops": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "lat": -0.9346,
      "lon": -78.6158,
      "address": "Calle Principal 123"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440002",
      "lat": -0.9350,
      "lon": -78.6160,
      "address": "Av. Central 456"
    }
  ]
}
```

### 2. Consultar orden activa

```bash
curl -X GET "http://localhost:8084/api/v1/driver/orders/active?driver_id=550e8400-e29b-41d4-a716-446655440000"
```

### 3. Iniciar orden

```bash
curl -X POST http://localhost:8084/api/v1/driver/orders/{ORDER_ID}/start \
  -H "Content-Type: application/json" \
  -d '{
    "start_latitude": -0.9346,
    "start_longitude": -78.6158
  }'
```

### 4. Completar paradas

```bash
curl -X POST http://localhost:8084/api/v1/driver/stops/1/complete \
  -H "Content-Type: application/json" \
  -d '{
    "status": "RECOGIDO",
    "comment": "Recolección exitosa"
  }'

curl -X POST http://localhost:8084/api/v1/driver/stops/2/complete \
  -H "Content-Type: application/json" \
  -d '{
    "status": "RECOGIDO"
  }'
```

### 5. Finalizar orden

```bash
curl -X POST http://localhost:8084/api/v1/driver/orders/{ORDER_ID}/finish \
  -H "Content-Type: application/json" \
  -d '{
    "end_latitude": -0.9350,
    "end_longitude": -78.6160
  }'
```

**Resultado esperado en logs:**
```
[INFO] Work order completed: id=xxx, driver=xxx, zone=1
[INFO] Published workorder.completed.v1 event for zone 1, driver xxx
```

**En Fleet Service:**
```
[INFO] Driver 550e8400-e29b-41d4-a716-446655440000 liberado
```

**En Scheduler Service:**
```
[INFO] Zona 1 reseteada (Score 0)
```

## 📊 Estados del Sistema

### Work Order Status
- `ASIGNADA`: Orden creada, esperando inicio
- `EN_PROGRESO`: Conductor comenzó la ruta
- `COMPLETADA`: Todas las paradas procesadas

### Stop Status
- `PENDIENTE`: Parada no visitada
- `RECOGIDO`: Residuos recolectados exitosamente
- `NO_RECOGIDO`: Parada visitada pero no se recolectó (ej: contenedor vacío)

## 🔧 Desarrollo

### Generar documentación Swagger

```bash
swag init -g cmd/server/main.go -o docs
```

### Ejecutar tests

```bash
go test ./...
```

### Build

```bash
go build -o bin/operations-service cmd/server/main.go
```

## 📝 Notas Importantes

1. **Transacciones**: Las actualizaciones de paradas y contadores usan transacciones para garantizar consistencia

2. **Eventos Asíncronos**: Los eventos RabbitMQ se publican en goroutines para no bloquear la respuesta HTTP

3. **Validaciones Críticas**:
   - Solo se puede iniciar una orden en estado ASIGNADA
   - Solo se puede finalizar si todas las paradas están procesadas
   - Solo se pueden completar paradas en estado PENDIENTE

4. **Manejo de Errores**: Todos los errores de RabbitMQ se loguean pero no fallan la operación HTTP

## 🔗 Dependencias

- **Scheduler Service**: Crea las órdenes de trabajo
- **Fleet Service**: Recibe notificaciones de finalización para liberar conductores
- **Incident Service**: Proporciona las referencias de incidentes para las paradas

## 🌐 Puerto por Defecto

`8084`

## 📚 Swagger UI

http://localhost:8084/swagger/index.html
