# Fleet Service - Vehicle Management & Operator Profiles

## 📋 Resumen de Cambios

Se ha implementado la gestión completa de vehículos (buses/camiones de recolección) en el **fleet-service** y se verificó que los perfiles de operador se crean automáticamente cuando los operadores se registran.

---

## ✅ Funcionalidades Implementadas

### 1. **CRUD de Camiones (Buses de Recolección)**

Se crearon endpoints REST para gestionar la flota de vehículos:

#### **Endpoints Disponibles:**

- **POST** `/api/v1/trucks` - Crear un nuevo camión
- **GET** `/api/v1/trucks` - Listar todos los camiones (con filtros opcionales)
- **GET** `/api/v1/trucks/:id` - Obtener un camión por ID
- **PUT** `/api/v1/trucks/:id` - Actualizar un camión existente
- **DELETE** `/api/v1/trucks/:id` - Eliminar un camión (solo si no tiene turnos activos)

#### **Tipos de Camiones:**
- `CARGA_LATERAL` - Camión de carga lateral
- `CARGA_POSTERIOR` - Camión de carga posterior

#### **Estados de Camiones:**
- `DISPONIBLE` - Listo para ser asignado
- `EN_USO` - Actualmente en un turno activo
- `MANTENIMIENTO` - Fuera de servicio

---

### 2. **Creación Automática de Perfiles de Operador**

El **IdentityConsumer** ya estaba implementado y funcional. Este consumidor:

- Escucha eventos `identity.operator.created.v1` en RabbitMQ
- Al recibir un evento de registro de operador, crea automáticamente:
  1. **Driver** - Registro del conductor con su UUID
  2. **OperatorProfile** - Perfil con zona preferida y permisos de conducción

#### **Campos del Perfil de Operador:**
```json
{
  "driver_id": "uuid",
  "user_id": "uuid",
  "preferred_zone_id": 1,
  "can_drive_lateral": true,
  "can_drive_posterior": false
}
```

---

### 3. **Integración RPC con Asignación Real**

El **FleetRPCConsumer** ahora usa el `AllocationService` para asignar conductores reales en lugar de datos simulados.

#### **Flujo de Asignación:**
1. **Schedule-service** solicita recursos vía RPC (Fleet Request)
2. **FleetRPCConsumer** recibe la solicitud
3. **AllocationService** busca el mejor conductor disponible:
   - Con turno activo
   - Conduciendo el tipo de camión requerido
   - Estado `DISPONIBLE`
   - **Prioridad:** Conductor con zona preferida = zona solicitada
4. Responde con `DriverID`, `TruckPlate`, `Status: ALLOCATED`

---

## 📂 Archivos Creados/Modificados

### **Nuevos Archivos:**

1. **`fleet-service/internal/services/truck_service.go`**
   - Lógica de negocio para gestión de camiones
   - Validaciones (placa única, campos obligatorios)
   - CRUD completo con verificación de turnos activos

2. **`fleet-service/internal/handlers/truck_handler.go`**
   - Handlers HTTP para endpoints de camiones
   - Validación de requests con Gin
   - Documentación Swagger inline

### **Archivos Modificados:**

3. **`fleet-service/cmd/fleet-service/main.go`**
   - Registradas rutas `/api/v1/trucks` con CRUD completo
   - Actualizado `NewFleetRPCConsumer` para recibir `AllocationService` y `EventPublisher`
   - Eliminados servicios placeholder (DriverService, TruckService, ResourceService)

4. **`fleet-service/internal/consumers/fleet_rpc_consumer.go`**
   - Refactorizado constructor para recibir `AllocationService`
   - Implementada asignación real en `handleResourceRequest()`:
     ```go
     allocation, err := c.allocationService.FindBestDriver(req.ZoneID, req.RequiredType)
     ```
   - Agregado logging detallado de asignaciones

---

## 🧪 Testing - Crear un Camión

### **Request (POST):**
```bash
curl -X POST http://localhost:8082/api/v1/trucks \
  -H "Content-Type: application/json" \
  -d '{
    "plate": "GYE-1234",
    "type": "CARGA_LATERAL",
    "status": "DISPONIBLE"
  }'
```

### **Response (201 Created):**
```json
{
  "id": 1,
  "plate": "GYE-1234",
  "type": "CARGA_LATERAL",
  "status": "DISPONIBLE",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

## 🔍 Testing - Listar Camiones con Filtros

### **Request (GET):**
```bash
# Todos los camiones
curl http://localhost:8082/api/v1/trucks

# Solo disponibles
curl http://localhost:8082/api/v1/trucks?status=DISPONIBLE

# Solo carga lateral
curl http://localhost:8082/api/v1/trucks?type=CARGA_LATERAL

# Disponibles de carga posterior
curl http://localhost:8082/api/v1/trucks?status=DISPONIBLE&type=CARGA_POSTERIOR
```

### **Response (200 OK):**
```json
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

---

## 🔄 Testing - Actualizar Estado de Camión

### **Request (PUT):**
```bash
curl -X PUT http://localhost:8082/api/v1/trucks/1 \
  -H "Content-Type: application/json" \
  -d '{
    "status": "MANTENIMIENTO"
  }'
```

### **Response (200 OK):**
```json
{
  "id": 1,
  "plate": "GYE-1234",
  "type": "CARGA_LATERAL",
  "status": "MANTENIMIENTO",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T14:25:00Z"
}
```

---

## 🗑️ Testing - Eliminar Camión

### **Request (DELETE):**
```bash
curl -X DELETE http://localhost:8082/api/v1/trucks/1
```

### **Response (200 OK):**
```json
{
  "message": "Camión eliminado exitosamente"
}
```

### **Error si tiene turnos activos (400 Bad Request):**
```json
{
  "error": "no se puede eliminar el camión: tiene 1 turno(s) activo(s)"
}
```

---

## 📊 Flujo Completo End-to-End

### **Escenario: Operador se registra → Se asigna a una ruta**

1. **Registro de Operador (auth-service):**
   ```json
   POST /api/v1/auth/register
   {
     "phone": "+593987654321",
     "full_name": "Juan Pérez",
     "role": "OPERATOR",
     "preferred_zone_id": 3,
     "can_drive_lateral": true,
     "can_drive_posterior": false
   }
   ```

2. **Publicación de Evento (auth-service → RabbitMQ):**
   ```
   Exchange: city.cleaning.identity
   RoutingKey: identity.operator.created.v1
   Body: {
     "driver_id": "550e8400-e29b-41d4-a716-446655440000",
     "full_name": "Juan Pérez",
     "preferred_zone_id": 3,
     "can_drive_lateral": true,
     "can_drive_posterior": false
   }
   ```

3. **IdentityConsumer (fleet-service) crea automáticamente:**
   - **Driver** con ID `550e8400-e29b-41d4-a716-446655440000`
   - **OperatorProfile** vinculado al Driver

4. **Operador hace Clock-In:**
   ```json
   POST /api/v1/shifts/clock-in
   {
     "driver_id": "550e8400-e29b-41d4-a716-446655440000",
     "truck_plate": "GYE-1234"
   }
   ```
   - Se crea **ActiveShift**
   - Estado del camión cambia a `EN_USO`
   - Estado del conductor cambia a `DISPONIBLE`

5. **Zona supera umbral de gravedad (schedule-service):**
   - **Orchestrator** ejecuta `RequestFleet()`
   - **FleetRPCConsumer** recibe solicitud RPC
   - **AllocationService** busca conductor:
     ```sql
     SELECT driver_id, truck_plate
     FROM drivers d
     JOIN active_shifts a ON d.id = a.driver_id
     JOIN trucks t ON a.truck_id = t.id
     WHERE d.status = 'DISPONIBLE'
       AND t.type = 'CARGA_LATERAL'
       AND a.is_active = true
     ORDER BY (CASE WHEN preferred_zone_id = 3 THEN 1 ELSE 2 END), RANDOM()
     LIMIT 1
     ```
   - Responde con `DriverID`, `TruckPlate`
   - **Estado del conductor** cambia a `OCUPADO`

6. **Routing-service optimiza ruta** (RPC)

7. **Operations-service recibe Work Order**

---

## 🎯 Ventajas de la Implementación

✅ **Separación de responsabilidades:**
- `TruckService` - Lógica de negocio
- `TruckHandler` - HTTP y validación
- `AllocationService` - Asignación inteligente

✅ **Validaciones robustas:**
- Placas únicas
- No eliminar camiones con turnos activos
- Tipos y estados válidos

✅ **Integración RPC funcional:**
- Asignación real de recursos
- Priorización por zona preferida
- Manejo de errores con compensación

✅ **Auto-creación de perfiles:**
- Sin intervención manual
- Event-driven con RabbitMQ
- Transaccional (Driver + OperatorProfile en una TX)

---

## 📝 Próximos Pasos Sugeridos

1. **Agregar endpoints de gestión de conductores:**
   - `GET /api/v1/drivers` - Listar conductores
   - `GET /api/v1/drivers/:id` - Ver perfil completo
   - `PUT /api/v1/drivers/:id/profile` - Actualizar preferencias

2. **Dashboard de disponibilidad:**
   - `GET /api/v1/fleet/availability` - Ver recursos disponibles por zona

3. **Testing end-to-end:**
   - Crear script que simule registro → clock-in → asignación automática

4. **Métricas:**
   - Tiempo promedio de asignación
   - Tasa de utilización de flota por zona

---

## 🔗 Documentación Relacionada

- **IMPLEMENTACION_RPC_COMPLETA.md** - Detalles del patrón RPC
- **ORCHESTRATOR_SAGA_RPC.md** - Saga Orchestrator
- **Swagger UI:** `http://localhost:8082/swagger/index.html`

---

## ✅ Checklist de Validación

- [x] Crear camiones vía API
- [x] Listar/filtrar camiones
- [x] Actualizar estado de camiones
- [x] Eliminar camiones (con validación)
- [x] Auto-creación de perfiles de operador
- [x] Asignación real en FleetRPCConsumer
- [x] Integración con AllocationService
- [ ] Testing end-to-end completo
- [ ] Probar con múltiples zonas y conductores
- [ ] Verificar rollback en caso de falla de routing

---

**Fecha de Implementación:** Enero 2024  
**Versión:** 1.0.0  
**Autor:** Sistema de Gestión de Residuos Latacunga
