# Guía de Pruebas - Operations Service

## 🎯 Objetivo

Probar el flujo completo de una orden de trabajo desde la asignación hasta la finalización, simulando el comportamiento de un conductor desde Postman.

## 📋 Prerrequisitos

1. **RabbitMQ corriendo**:
```bash
docker-compose up -d rabbitmq
# Verificar en http://localhost:15672 (user: tesis, pass: tesis)
```

2. **Base de datos PostgreSQL** con las tablas creadas

3. **Operations Service corriendo**:
```bash
cd operations-service
go run cmd/server/main.go
```

## 🔄 Flujo de Prueba Completo

### Paso 1: Crear una Orden de Trabajo (Simular Scheduler)

Publica un mensaje a RabbitMQ usando la Management UI o este script:

**Opción A: Via RabbitMQ Management UI**

1. Ir a http://localhost:15672
2. Login: `tesis` / `tesis`
3. Ir a pestaña "Exchanges"
4. Click en `city.cleaning.operations`
5. En "Publish message":
   - Routing key: `workorders.created.v1`
   - Payload:

```json
{
  "request_id": "test_001",
  "zone_id": 1,
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "truck_plate": "LAT-123",
  "geometry": "polyline_encoded_string_from_osrm",
  "stops": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "lat": -0.9346,
      "lon": -78.6158,
      "address": "Mercado Central"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440002",
      "lat": -0.9350,
      "lon": -78.6160,
      "address": "Parque Vicente León"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440003",
      "lat": -0.9355,
      "lon": -78.6165,
      "address": "Plaza de San Sebastián"
    }
  ]
}
```

6. Click "Publish message"

**Verificar en logs del service**:
```
[INFO] Received workorder.created event: request_id=test_001, zone=1, driver=550e8400...
[INFO] Work order created successfully: id=xxx, stops=3
```

### Paso 2: Conductor Consulta su Orden Activa

**Request:**
```bash
curl -X GET "http://localhost:8084/api/v1/driver/orders/active?driver_id=550e8400-e29b-41d4-a716-446655440000"
```

**Response Esperado (200 OK):**
```json
{
  "id": "9c24960c-a1f9-45cf-ae96-1337833134fd",
  "request_id": "test_001",
  "zone_id": 1,
  "driver_id": "550e8400-e29b-41d4-a716-446655440000",
  "truck_plate": "LAT-123",
  "status": "ASIGNADA",
  "route_polyline": "polyline_encoded_string_from_osrm",
  "total_stops": 3,
  "completed_stops": 0,
  "assigned_at": "2025-11-23T19:30:00Z",
  "stops": [
    {
      "id": 1,
      "work_order_id": "9c24960c-a1f9-45cf-ae96-1337833134fd",
      "incident_ref_id": "660e8400-e29b-41d4-a716-446655440001",
      "latitude": -0.9346,
      "longitude": -78.6158,
      "address": "Mercado Central",
      "sequence_order": 1,
      "status": "PENDIENTE"
    },
    {
      "id": 2,
      "work_order_id": "9c24960c-a1f9-45cf-ae96-1337833134fd",
      "incident_ref_id": "660e8400-e29b-41d4-a716-446655440002",
      "latitude": -0.9350,
      "longitude": -78.6160,
      "address": "Parque Vicente León",
      "sequence_order": 2,
      "status": "PENDIENTE"
    },
    {
      "id": 3,
      "work_order_id": "9c24960c-a1f9-45cf-ae96-1337833134fd",
      "incident_ref_id": "660e8400-e29b-41d4-a716-446655440003",
      "latitude": -0.9355,
      "longitude": -78.6165,
      "address": "Plaza de San Sebastián",
      "sequence_order": 3,
      "status": "PENDIENTE"
    }
  ]
}
```

**Copiar el `id` de la orden para los siguientes pasos.**

### Paso 3: Conductor Inicia la Orden

**Request:**
```bash
curl -X POST http://localhost:8084/api/v1/driver/orders/9c24960c-a1f9-45cf-ae96-1337833134fd/start \
  -H "Content-Type: application/json" \
  -d '{
    "start_latitude": -0.9346,
    "start_longitude": -78.6158
  }'
```

**Response Esperado (200 OK):**
```json
{
  "message": "Work order started successfully",
  "data": {
    "id": "9c24960c-a1f9-45cf-ae96-1337833134fd",
    "status": "EN_PROGRESO",
    "started_at": "2025-11-23T19:31:00Z",
    ...
  }
}
```

**Verificar en DB:**
```sql
SELECT id, status, started_at FROM work_orders 
WHERE id = '9c24960c-a1f9-45cf-ae96-1337833134fd';
```

### Paso 4: Completar Primera Parada (Stop ID 1)

**Request:**
```bash
curl -X POST http://localhost:8084/api/v1/driver/stops/1/complete \
  -H "Content-Type: application/json" \
  -d '{
    "status": "RECOGIDO",
    "comment": "Recolección exitosa en Mercado Central"
  }'
```

**Response Esperado (200 OK):**
```json
{
  "message": "Stop completed successfully",
  "data": {
    "id": 1,
    "status": "RECOGIDO",
    "serviced_at": "2025-11-23T19:32:00Z",
    ...
  }
}
```

**Verificar contador en DB:**
```sql
SELECT completed_stops FROM work_orders 
WHERE id = '9c24960c-a1f9-45cf-ae96-1337833134fd';
-- Debe mostrar: 1
```

### Paso 5: Completar Segunda Parada (Stop ID 2)

**Request:**
```bash
curl -X POST http://localhost:8084/api/v1/driver/stops/2/complete \
  -H "Content-Type: application/json" \
  -d '{
    "status": "RECOGIDO"
  }'
```

**Contador debe incrementar a 2**

### Paso 6: Completar Tercera Parada (Stop ID 3)

**Request:**
```bash
curl -X POST http://localhost:8084/api/v1/driver/stops/3/complete \
  -H "Content-Type: application/json" \
  -d '{
    "status": "NO_RECOGIDO",
    "comment": "Contenedor ya vacío"
  }'
```

**Nota:** El contador NO incrementa porque es `NO_RECOGIDO`

### Paso 7: Finalizar la Orden

**Request:**
```bash
curl -X POST http://localhost:8084/api/v1/driver/orders/9c24960c-a1f9-45cf-ae96-1337833134fd/finish \
  -H "Content-Type: application/json" \
  -d '{
    "end_latitude": -0.9355,
    "end_longitude": -78.6165
  }'
```

**Response Esperado (200 OK):**
```json
{
  "message": "Work order finished successfully",
  "data": {
    "id": "9c24960c-a1f9-45cf-ae96-1337833134fd",
    "status": "COMPLETADA",
    "completed_at": "2025-11-23T19:35:00Z",
    "total_stops": 3,
    "completed_stops": 2
  }
}
```

**Logs Esperados:**
```
[INFO] Work order completed: id=9c24960c..., driver=550e8400..., zone=1
[INFO] Published workorder.completed.v1 event for zone 1, driver 550e8400...
```

### Paso 8: Verificar Evento Publicado

**En RabbitMQ Management UI:**

1. Ir a pestaña "Exchanges"
2. Click en `city.cleaning.operations`
3. Ver estadísticas de mensajes publicados

**O consumir desde otro servicio:**

En Fleet Service o Scheduler Service deberían ver:
```
[INFO] Received workorder.completed event: zone=1, driver=550e8400...
[INFO] Driver 550e8400-e29b-41d4-a716-446655440000 liberado
```

## 🧪 Casos de Prueba Adicionales

### Caso 1: Intentar Iniciar Orden Ya Iniciada

```bash
curl -X POST http://localhost:8084/api/v1/driver/orders/{ID}/start \
  -H "Content-Type: application/json" \
  -d '{"start_latitude": -0.9346, "start_longitude": -78.6158}'
```

**Respuesta Esperada (400):**
```json
{
  "error": "invalid status",
  "message": "Work order must be in ASIGNADA status to start"
}
```

### Caso 2: Intentar Finalizar con Paradas Pendientes

```bash
# No completar todas las paradas antes de finalizar
curl -X POST http://localhost:8084/api/v1/driver/orders/{ID}/finish \
  -H "Content-Type: application/json" \
  -d '{"end_latitude": -0.9355, "end_longitude": -78.6165}'
```

**Respuesta Esperada (400):**
```json
{
  "error": "incomplete stops",
  "message": "All stops must be processed before finishing the work order"
}
```

### Caso 3: Completar Parada Ya Completada

```bash
# Intentar completar la misma parada dos veces
curl -X POST http://localhost:8084/api/v1/driver/stops/1/complete \
  -H "Content-Type: application/json" \
  -d '{"status": "RECOGIDO"}'
```

**Respuesta Esperada (400):**
```json
{
  "error": "invalid status",
  "message": "Stop must be in PENDIENTE status"
}
```

### Caso 4: Consultar Orden de Conductor Sin Órdenes

```bash
curl -X GET "http://localhost:8084/api/v1/driver/orders/active?driver_id=00000000-0000-0000-0000-000000000000"
```

**Respuesta Esperada (204 No Content):**
Sin body

## 📊 Verificaciones en Base de Datos

### Verificar Estado de Orden

```sql
SELECT 
    id,
    request_id,
    driver_id,
    status,
    total_stops,
    completed_stops,
    assigned_at,
    started_at,
    completed_at
FROM work_orders
WHERE driver_id = '550e8400-e29b-41d4-a716-446655440000'
ORDER BY assigned_at DESC;
```

### Verificar Paradas

```sql
SELECT 
    s.id,
    s.sequence_order,
    s.address,
    s.status,
    s.serviced_at
FROM work_order_stops s
JOIN work_orders w ON s.work_order_id = w.id
WHERE w.id = '9c24960c-a1f9-45cf-ae96-1337833134fd'
ORDER BY s.sequence_order;
```

### Verificar Progreso

```sql
SELECT 
    w.id,
    w.status as order_status,
    w.total_stops,
    w.completed_stops,
    COUNT(CASE WHEN s.status = 'RECOGIDO' THEN 1 END) as recogidos,
    COUNT(CASE WHEN s.status = 'NO_RECOGIDO' THEN 1 END) as no_recogidos,
    COUNT(CASE WHEN s.status = 'PENDIENTE' THEN 1 END) as pendientes
FROM work_orders w
LEFT JOIN work_order_stops s ON s.work_order_id = w.id
WHERE w.id = '9c24960c-a1f9-45cf-ae96-1337833134fd'
GROUP BY w.id;
```

## 🔍 Troubleshooting

### Problema: No se crea la orden

**Verificar:**
1. RabbitMQ está corriendo
2. El servicio tiene logs de conexión a RabbitMQ
3. El mensaje se publicó correctamente en la cola

**Comandos útiles:**
```bash
# Ver colas en RabbitMQ
curl -u tesis:tesis http://localhost:15672/api/queues

# Ver mensajes en cola
curl -u tesis:tesis http://localhost:15672/api/queues/%2F/q.operations.workorders
```

### Problema: Error 500 al completar parada

**Verificar:**
- El ID de la parada existe
- La parada pertenece a una orden válida
- El estado de la parada es PENDIENTE

### Problema: No se publica evento de finalización

**Verificar logs:**
```
ERROR: Failed to publish workorder.completed event: connection closed
```

**Solución:** Reiniciar RabbitMQ o el servicio

## ✅ Checklist de Prueba Exitosa

- [ ] Orden creada desde RabbitMQ
- [ ] GET /active retorna la orden con 3 paradas
- [ ] POST /start cambia estado a EN_PROGRESO
- [ ] POST /stops/1/complete incrementa completed_stops
- [ ] POST /stops/2/complete incrementa completed_stops
- [ ] POST /stops/3/complete NO incrementa (NO_RECOGIDO)
- [ ] POST /finish cambia estado a COMPLETADA
- [ ] Evento workorder.completed.v1 publicado
- [ ] GET /active retorna 204 (sin órdenes activas)

## 🎓 Conclusión

Si todos los pasos se completaron exitosamente:

✅ El servicio está funcionando correctamente
✅ La integración con RabbitMQ funciona
✅ Las transacciones de BD son consistentes
✅ Los eventos se publican correctamente
✅ Listo para integrar con la App Móvil
