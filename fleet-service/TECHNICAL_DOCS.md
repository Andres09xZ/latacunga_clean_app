# Documentación Técnica - Fleet Service

## Arquitectura del Sistema

### Visión General
Fleet Service es un microservicio responsable de gestionar la flota de camiones, conductores, turnos y realizar asignaciones inteligentes de recursos bajo demanda.

### Componentes Principales

#### 1. **Modelos de Datos** (`internal/models`)
- **Truck**: Camiones con estados (DISPONIBLE, EN_USO, MANTENIMIENTO)
- **Driver**: Conductores con estados (OFFLINE, DISPONIBLE, OCUPADO)
- **OperatorProfile**: Perfil del operador (zona preferida, permisos de conducción)
- **ActiveShift**: Turnos activos de conductores

#### 2. **Handlers REST** (`internal/handlers`)
- **ShiftHandler**: Maneja clock-in y clock-out de operadores

#### 3. **Servicios** (`internal/services`)
- **AllocationService**: Motor de asignación inteligente con algoritmo de priorización
- **EventPublisher**: Publicador de eventos a RabbitMQ

#### 4. **Consumidores RabbitMQ** (`internal/consumers`)
- **IdentityConsumer**: Sincroniza operadores desde Identity Service
- **ResourceConsumer**: Procesa solicitudes de recursos desde Scheduler
- **WorkorderConsumer**: Libera conductores al completar órdenes de trabajo

## Flujos de Negocio

### Flujo 1: Clock-In (Inicio de Turno)
```
1. Operador envía solicitud con driver_id y truck_plate
2. Sistema valida:
   - Conductor existe
   - No tiene turno activo
   - Camión existe y está DISPONIBLE
3. Sistema crea ActiveShift
4. Actualiza Truck → EN_USO
5. Actualiza Driver → DISPONIBLE
6. Publica evento: resource.available.v1
```

### Flujo 2: Clock-Out (Fin de Turno)
```
1. Operador envía solicitud con driver_id
2. Sistema busca turno activo
3. Cierra turno (EndTime, IsActive=false)
4. Actualiza Driver → OFFLINE
5. Actualiza Truck → DISPONIBLE
```

### Flujo 3: Asignación Inteligente de Conductor
```
1. Scheduler solicita recurso (zona, tipo de camión)
2. AllocationService ejecuta query SQL con:
   - Filtros: turno activo, tipo correcto, estado DISPONIBLE
   - Priorización: zona preferida coincidente
   - Random para desempate
3. Bloqueo atómico: UPDATE driver SET status = 'OCUPADO'
4. Publica evento: resources.driver.allocated.v1
```

### Flujo 4: Liberación de Conductor
```
1. Operations Service completa orden de trabajo
2. Publica evento: workorder.completed.v1
3. WorkorderConsumer procesa evento
4. AllocationService libera conductor
5. Actualiza Driver → DISPONIBLE
```

## Motor de Asignación Inteligente

### Algoritmo de Búsqueda

```sql
WITH eligible_drivers AS (
    SELECT 
        d.id, d.full_name, d.status,
        t.id, t.plate, t.type,
        op.preferred_zone_id,
        a_s.id as shift_id,
        CASE 
            WHEN op.preferred_zone_id = :zone_id THEN 1 
            ELSE 2 
        END as priority
    FROM drivers d
    INNER JOIN active_shifts a_s ON d.id = a_s.driver_id AND a_s.is_active = true
    INNER JOIN trucks t ON a_s.truck_id = t.id
    INNER JOIN operator_profiles op ON d.id = op.driver_id
    WHERE 
        d.status = 'DISPONIBLE'
        AND t.type = :truck_type
        AND (
            (t.type = 'CARGA_LATERAL' AND op.can_drive_lateral = true)
            OR
            (t.type = 'CARGA_POSTERIOR' AND op.can_drive_posterior = true)
        )
)
SELECT * FROM eligible_drivers
ORDER BY priority ASC, RANDOM()
LIMIT 1
```

### Criterios de Priorización
1. **Prioridad 1**: Conductores con zona preferida = zona solicitada
2. **Prioridad 2**: Resto de conductores disponibles
3. **Desempate**: Random

### Mecanismo de Bloqueo (Locking)
Para evitar **race conditions** y doble asignación:

```go
// Dentro de una transacción
updateResult := tx.Model(&Driver{}).
    Where("id = ? AND status = ?", driverID, "DISPONIBLE").
    Update("status", "OCUPADO")

if updateResult.RowsAffected == 0 {
    // Conductor ya fue asignado por otro proceso
    return ErrAllocationFailed
}
```

## Eventos RabbitMQ

### Consumidos

| Exchange | Routing Key | Payload | Acción |
|----------|-------------|---------|--------|
| `identity.management` | `identity.operator.created.v1` | OperatorCreatedEvent | Crea Driver y OperatorProfile |
| `planning.scheduler` | `planning.resource.requested.v1` | ResourceRequestedEvent | Asigna conductor con FindBestDriver |
| `operations.workorders` | `workorder.completed.v1` | WorkorderCompletedEvent | Libera conductor (DISPONIBLE) |

### Publicados

| Exchange | Routing Key | Payload | Cuándo |
|----------|-------------|---------|--------|
| `city.cleaning.resources` | `resource.available.v1` | ResourceAvailableEvent | Clock-in exitoso |
| `city.cleaning.resources` | `resources.driver.allocated.v1` | DriverAllocatedEvent | Asignación exitosa |

## Estados y Transiciones

### Estados de Truck
```
DISPONIBLE → EN_USO (clock-in)
EN_USO → DISPONIBLE (clock-out)
DISPONIBLE ⇄ MANTENIMIENTO (manual)
```

### Estados de Driver
```
OFFLINE → DISPONIBLE (clock-in)
DISPONIBLE → OCUPADO (asignación)
OCUPADO → DISPONIBLE (completar workorder)
DISPONIBLE → OFFLINE (clock-out)
```

## Validaciones Críticas

### Clock-In
- ✅ Conductor debe existir
- ✅ Conductor NO debe tener turno activo
- ✅ Camión debe existir
- ✅ Camión debe estar DISPONIBLE

### Clock-Out
- ✅ Debe existir turno activo del conductor

### Asignación
- ✅ Conductor debe tener turno activo
- ✅ Conductor debe estar DISPONIBLE (no OCUPADO)
- ✅ Tipo de camión debe coincidir con lo solicitado
- ✅ Conductor debe tener permiso para ese tipo de camión
- ✅ Bloqueo atómico para evitar doble asignación

## Configuración

### Variables de Entorno Críticas

```env
# Database (Neon PostgreSQL)
DB_HOST=ep-xxx.neon.tech
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=xxxxx
DB_NAME=fleet_db
DB_SSLMODE=require

# RabbitMQ
RABBITMQ_URL=amqp://user:pass@host:5672/

# Server
SERVER_PORT=8082
GIN_MODE=release
```

## Optimizaciones

### Índices de Base de Datos
```sql
-- Índices automáticos por GORM
CREATE INDEX idx_drivers_status ON drivers(status);
CREATE INDEX idx_trucks_status ON trucks(status);
CREATE INDEX idx_active_shifts_driver_active ON active_shifts(driver_id, is_active);
CREATE UNIQUE INDEX idx_operator_profiles_driver_id ON operator_profiles(driver_id);
```

### Pool de Conexiones
```go
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
```

## Monitoreo y Logs

### Logs Importantes
```
✓ Clock-in exitoso - Driver: {uuid}, Truck: {plate}
✓ Clock-out exitoso - Driver: {uuid}
✓ Conductor asignado: {name} - Zona {id} ⭐ CON PRIORIDAD
✓ Conductor liberado: {uuid}
⚠ No hay conductores disponibles para zona {id}
❌ Error al bloquear conductor (race condition)
```

### Health Check
```bash
GET /health
Response: {"service": "fleet-service", "status": "healthy"}
```

## Testing

### Tests Unitarios
```bash
go test ./...
```

### Tests de Integración
Requiere:
- PostgreSQL en ejecución
- RabbitMQ en ejecución

### Carga de Datos de Prueba
```bash
psql $DATABASE_URL -f scripts/init_data.sql
```

## Troubleshooting

### Problema: Conductores no se asignan
**Causa**: No hay conductores con turno activo o todos están OCUPADOS
**Solución**: Verificar `SELECT * FROM active_shifts WHERE is_active = true`

### Problema: Error de conexión a RabbitMQ
**Causa**: URL incorrecta o RabbitMQ no está corriendo
**Solución**: Verificar RABBITMQ_URL y `docker ps | grep rabbitmq`

### Problema: Migraciones no se aplican
**Causa**: Error de conexión a PostgreSQL
**Solución**: Verificar DB_HOST, DB_USER, DB_PASSWORD en .env

## Roadmap Futuro

- [ ] Métricas con Prometheus
- [ ] Tracing distribuido con OpenTelemetry
- [ ] Cache de conductores disponibles con Redis
- [ ] API de consulta de disponibilidad
- [ ] Dashboard en tiempo real
- [ ] Notificaciones push a operadores móviles
- [ ] Historial de turnos y reportes

## Referencias

- [GORM Documentation](https://gorm.io/docs/)
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [RabbitMQ Go Client](https://github.com/rabbitmq/amqp091-go)
- [PostgreSQL Best Practices](https://www.postgresql.org/docs/current/index.html)
