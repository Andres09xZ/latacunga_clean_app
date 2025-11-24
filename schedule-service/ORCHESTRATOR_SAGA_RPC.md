# Schedule Service - Orquestador de Sagas con Patrón RPC

## 📋 Resumen de Implementación

Se ha refactorizado el **schedule-service** para convertirlo en un **Orquestador de Sagas** que coordina la planificación de rutas de recolección usando el patrón **RPC (Request-Reply)** sobre RabbitMQ.

---

## 🏗️ Arquitectura Implementada

### Componentes Principales

1. **RPCClient** (`internal/messaging/rpc_client.go`)
   - Cliente RabbitMQ para comunicación síncrona
   - Genera `CorrelationId` único por solicitud
   - Establece `ReplyTo` en cola exclusiva `q.scheduler.replies`
   - Timeout de 10 segundos por solicitud
   - Métodos:
     - `RequestFleet()`: Solicita recursos (chofer + camión) a fleet-service
     - `RequestRouting()`: Solicita optimización de ruta a routing-service
     - `ReleaseFleet()`: Compensación para liberar recursos en caso de fallo

2. **Orchestrator** (`internal/service/orchestrator.go`)
   - Coordina el flujo completo de planificación (Saga)
   - Implementa compensaciones (rollback) en caso de fallos
   - Método principal: `TriggerZone(zoneID, zoneName)`

3. **DTOs** (`internal/models/rpc_dtos.go`)
   - `FleetRequest/Response`: Comunicación con fleet-service
   - `RoutingRequest/Response`: Comunicación con routing-service
   - `WorkOrderEvent`: Evento final para operations-service

4. **OrchestratorAdapter** (`internal/service/orchestrator_adapter.go`)
   - Adapta el Orchestrator a la interfaz de TriggerLogic
   - Permite integración con PlanningService existente

---

## 🔄 Flujo de Orquestación (Saga)

Cuando una zona supera el umbral de gravedad (Score > 50):

### Paso 0: Bloqueo de Incidentes
```
Estado inicial → PROCESSING
```
- Marca todos los incidentes de la zona como `PROCESSING`
- Evita procesamiento duplicado

### Paso 1: Solicitar Recursos (Fleet)
```
Schedule → [RPC] → Fleet Service
Request: { zone_id, required_type }
Response: { driver_id, assistant_id, truck_plate, status }
```
**Si falla**: Desbloquea incidentes → ❌ Abortar

### Paso 2: Solicitar Ruta Optimizada (Routing)
```
Schedule → [RPC] → Routing Service
Request: { points: [DEPOT + incidents] }
Response: { polyline, distance_km, duration_min, optimized_order }
```
**Si falla**: 
1. Envía `FleetReleaseRequest` (compensación) ➡️ Libera recursos
2. Desbloquea incidentes → ❌ Abortar

### Paso 3: Publicar Work Order (Operations)
```
Schedule → [Fire & Forget] → Operations Service
Event: WorkOrderEvent { order_id, driver, truck, route, incidents }
```
**Si falla**: Compensación Fleet + Desbloquear incidentes

### Paso 4: Resetear Score de Zona
```
UPDATE cleaning_zones SET score = 0 WHERE id = zone_id
```

---

## 📡 Configuración RabbitMQ

### Exchanges Utilizados
- `city.cleaning.planning` (Topic) - Para solicitudes RPC
- `city.cleaning.operations` (Topic) - Para Work Orders

### Routing Keys
- `fleet.resource.request` - Solicitud de recursos
- `fleet.resource.release` - Liberación de recursos (compensación)
- `routing.optimize` - Solicitud de optimización de ruta
- `workorder.created` - Creación de orden de trabajo

### Colas
- `q.scheduler.replies` - Cola de respuestas RPC (exclusiva del scheduler)
- Los servicios fleet y routing deben escuchar sus routing keys correspondientes

---

## 🛠️ Cambios Realizados

### Archivos Nuevos
```
schedule-service/
├── internal/
│   ├── messaging/
│   │   └── rpc_client.go                  ✅ Cliente RPC
│   ├── models/
│   │   └── rpc_dtos.go                    ✅ DTOs para RPC
│   └── service/
│       ├── orchestrator.go                ✅ Lógica de orquestación
│       └── orchestrator_adapter.go        ✅ Adaptador de interfaz
```

### Archivos Modificados
```
schedule-service/
├── internal/
│   ├── server/
│   │   └── server.go                      🔧 Inicializa Orchestrator
│   ├── services/
│   │   └── planning_service.go            🔧 Integra Orchestrator
│   └── models/
│       └── zone.go                        🔧 PendingItem con punteros
```

---

## 🚀 Uso

### Inicialización del Servicio

El orchestrator se inicializa automáticamente al arrancar el schedule-service si:
1. `RABBITMQ_URL` está configurado
2. La conexión a RabbitMQ es exitosa

```bash
cd schedule-service
go run ./cmd/server/main.go
```

Logs esperados:
```
✅ RPCClient inicializado con cola de respuestas: q.scheduler.replies
🔄 Consumidor de respuestas RPC iniciado en cola: q.scheduler.replies
✅ Orchestrator initialized - Saga RPC pattern enabled for zone triggers
```

### Flujo de Prueba

1. **Crear incidentes en una zona** hasta superar umbral (Score > 50)
2. **Observar logs del orchestrator**:
   ```
   🚀 [ORCHESTRATOR] Iniciando planificación para Zona 2 (URBANO_NORTE)
   🔒 5 incidentes marcados como PROCESSING
   📤 FleetRequest enviado: ZoneID=2, CorrelationId=xxx
   ✅ FleetResponse recibido: Driver=xxx, Truck=ABC-123
   📤 RoutingRequest enviado: ZoneID=2, Points=6, CorrelationId=yyy
   ✅ RoutingResponse recibido: Distance=15.06km, Duration=25min
   ✅ Work Order publicado: OrderID=zzz
   🔄 Score de zona 2 reseteado a 0
   🎉 [ORCHESTRATOR] Planificación completada exitosamente para Zona 2
   ```

### Escenarios de Fallo

#### Fallo en Fleet Service
```
❌ Error en RequestFleet: timeout esperando respuesta
🔓 Incidentes de zona 2 desbloqueados
```

#### Fallo en Routing Service
```
❌ Error en RequestRouting: OSRM no disponible
🔄 Compensación enviada: Driver=xxx, Truck=ABC-123, Reason=ROUTING_FAILED
🔓 Incidentes de zona 2 desbloqueados
```

---

## 🔧 Configuración Requerida

### Variables de Entorno
```env
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
DB_URL=postgresql://user:pass@localhost:5432/schedule_db
PORT=8083
```

### Dependencias Go
```bash
go get github.com/rabbitmq/amqp091-go
go get github.com/google/uuid
```

---

## 📝 Notas Importantes

1. **DEPOT Automático**: El orchestrator añade automáticamente el punto DEPOT (EPAGAL) al inicio de cada ruta
   ```go
   DEPOT_LATITUDE  = -0.9364043
   DEPOT_LONGITUDE = -78.6087099
   ```

2. **Timeout RPC**: Todas las solicitudes RPC tienen un timeout de 10 segundos
   ```go
   const RPCTimeout = 10 * time.Second
   ```

3. **Compensaciones**: Si una operación falla, el orchestrator ejecuta automáticamente las compensaciones necesarias (rollback)

4. **Estado de Incidentes**: 
   - `PENDING` → Esperando procesamiento
   - `PROCESSING` → En proceso de planificación
   - Si falla → Vuelven a `PENDING`

5. **Integración con Fleet/Operations**: Los servicios fleet-service y operations-service deben implementar los endpoints correspondientes para responder a las solicitudes RPC

---

## 🧪 Testing

Para probar el flujo completo:

```bash
# 1. Iniciar todos los servicios
docker-compose up -d

# 2. Crear incidentes hasta superar umbral
curl -X POST http://localhost:8082/api/v1/incidents \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": -0.935,
    "longitude": -78.618,
    "type": "punto_acopio",
    "description": "Test incident"
  }'

# 3. Verificar logs del schedule-service
docker logs -f schedule-service
```

---

## ✅ Estado de Implementación

- [x] DTOs para RPC definidos
- [x] RPCClient implementado con Request-Reply
- [x] Consumidor de respuestas RPC configurado
- [x] Orchestrator con lógica de Saga implementado
- [x] Compensaciones (rollback) configuradas
- [x] Integración con PlanningService completada
- [x] Logging detallado añadido
- [ ] Testing end-to-end con fleet-service real
- [ ] Testing end-to-end con operations-service real
- [ ] Métricas y observabilidad avanzada

---

## 📚 Referencias

- [Patrón Saga](https://microservices.io/patterns/data/saga.html)
- [RabbitMQ RPC Tutorial](https://www.rabbitmq.com/tutorials/tutorial-six-go.html)
- [Compensating Transactions](https://docs.microsoft.com/en-us/azure/architecture/patterns/compensating-transaction)
