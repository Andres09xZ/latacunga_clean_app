# Implementación Feature BDD - API REST Scheduler Service

## Resumen de cambios

Esta implementación cumple con la especificación Gherkin para el servicio de planificación:
- **5 escenarios** implementados según feature BDD
- **Persistencia completa** de incidentes y órdenes de trabajo
- **Estados de zona** (ACUMULANDO, LISTO_PARA_RECOLECCION, EN_PROGRESO)
- **Geometría incluida** en listado de zonas
- **Códigos de respuesta** según especificación (200, 201, 202)

## Migración requerida

Antes de probar, ejecuta la migración 009 en tu base de datos Neon:

```sql
-- d:/Octavo Semestre/Tesis/backend_latacunga_clean/schedule-service/migrations/009_add_zone_status.sql

ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'ACUMULANDO';
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS last_updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP;
UPDATE cleaning_zones SET last_updated = updated_at WHERE last_updated IS NULL;
CREATE INDEX IF NOT EXISTS idx_cleaning_zones_status ON cleaning_zones(status);
```

## Endpoints implementados

### 1. GET /api/v1/zones (Listar zonas con geometría)
**Status:** 200 OK  
**Headers:** `Authorization: Bearer <token>`  
**Respuesta:**
```json
[
  {
    "id": 1,
    "name": "URBANO_CENTRAL",
    "type": "URBANO",
    "zone_name": "URBANO_CENTRAL",
    "route_type": "URBANO",
    "schedule_time": "08:00:00",
    "threshold": 50,
    "current_score": 45,
    "status": "ACUMULANDO",
    "last_updated": "2025-11-22T12:00:00Z",
    "geometry": "{\"type\":\"Feature\",\"geometry\":{...},\"properties\":{...}}"
  }
]
```

### 2. GET /api/v1/zones/{id}/metrics (Ver métricas)
**Status:** 200 OK  
**Headers:** `Authorization: Bearer <token>`  
**Respuesta:**
```json
{
  "zone_id": 1,
  "name": "URBANO_CENTRAL",
  "current_score": 45,
  "threshold": 50,
  "status": "ACUMULANDO",
  "last_updated": "2025-11-22T12:00:00Z"
}
```

### 3. POST /api/v1/debug/simulate (Simular incidente)
**Status:** 202 Accepted  
**Headers:** `Authorization: Bearer <token>`, `Content-Type: application/json`  
**Body (opción explícita):**
```json
{
  "zone_id": 1,
  "type": "SENSOR_IOT_LLENO",
  "points": 5
}
```
**Body (opción geo):**
```json
{
  "type": "SENSOR_IOT_LLENO",
  "lat": -0.933,
  "lon": -78.614
}
```
**Respuesta:**
```json
{
  "message": "Incidente simulado",
  "zone_id": 1,
  "points_added": 5
}
```

### 4. POST /api/v1/config/weights (Configurar pesos)
**Status:** 200 OK  
**Headers:** `Authorization: Bearer <token>`, `Content-Type: application/json`  
**Body:**
```json
{
  "SENSOR_IOT_LLENO": 10,
  "REPORTE_CIUDADANO": 2
}
```
**Respuesta:**
```json
{
  "message": "Pesos actualizados correctamente",
  "weights": {
    "SENSOR_IOT_LLENO": 10,
    "REPORTE_CIUDADANO": 2
  }
}
```

### 5. POST /api/v1/zones/{id}/trigger (Botón de pánico)
**Status:** 201 Created  
**Headers:** `Authorization: Bearer <token>`, `Content-Type: application/json`  
**Body:**
```json
{
  "reason": "Emergencia Sanitaria reportada por Alcaldía",
  "operator_override": "CHOFER_01"
}
```
**Respuesta:**
```json
{
  "work_order_id": 42,
  "zone_id": 1,
  "status": "EN_PROGRESO",
  "reason": "Emergencia Sanitaria reportada por Alcaldía",
  "operator_override": "CHOFER_01",
  "work_order": {
    "ID": 42,
    "ZoneName": "URBANO_CENTRAL",
    "GeneratedAt": "2025-11-22T12:30:00Z",
    "TotalPoints": 10,
    "Threshold": 50,
    "Incidents": [...]
  }
}
```

## Estados de zona

- **ACUMULANDO**: Puntaje < umbral, acumulando incidentes
- **LISTO_PARA_RECOLECCION**: Puntaje >= umbral, generó orden automáticamente
- **EN_PROGRESO**: Orden forzada manualmente (trigger), operación en curso

## Archivos modificados

1. **migrations/009_add_zone_status.sql** - Nueva columna status y last_updated
2. **internal/models/zone.go** - Agregados Status y LastUpdated
3. **internal/models/work_order.go** - Sin cambios (ya tenía ID)
4. **internal/repository/zone_repo.go** - Métodos actualizados:
   - ListZonesWithMetrics: incluye geometría y status
   - GetZoneMetrics: formato exacto Gherkin
   - IncrementZoneScore: actualiza status automáticamente
   - ResetZoneScore: reinicia a ACUMULANDO
   - UpdateZoneStatus (nuevo): cambio manual de estado
5. **internal/handlers/v1_admin_handlers.go** - Endpoints extendidos:
   - SimulateIncident: acepta zone_id+points o lat+lon+type
   - UpdateWeights: mensaje "Pesos actualizados correctamente"
   - ForceTrigger: acepta reason/operator, retorna 201, actualiza status a EN_PROGRESO
6. **internal/service/incident_trigger.go** - WorkOrder DTO con ID

## Probar manualmente

### 1. Ejecutar migración
Copia contenido de `migrations/009_add_zone_status.sql` y ejecuta en Neon.

### 2. Inicializar datos de prueba (si necesario)
```sql
UPDATE cleaning_zones SET current_score = 45, status = 'ACUMULANDO' WHERE id = 1;
```

### 3. Levantar servicio
```powershell
go run cmd/server/main.go
```

### 4. Generar token
```powershell
$env:JWT_SECRET = "mysecret"
$TOKEN = go run scripts/gen_jwt.go -sub admin1 -role admin -email admin@municipio.ec -secret $env:JWT_SECRET
```

### 5. Ejecutar script de prueba BDD
```powershell
.\scripts\test_bdd_api.ps1
```

O probar endpoints individualmente:

```powershell
$HEADERS = @{ Authorization = "Bearer $TOKEN" }

# Listar zonas
Invoke-RestMethod -Method GET -Uri http://localhost:8083/api/v1/zones -Headers $HEADERS

# Ver métricas
Invoke-RestMethod -Method GET -Uri http://localhost:8083/api/v1/zones/1/metrics -Headers $HEADERS

# Simular incidente
Invoke-RestMethod -Method POST -Uri http://localhost:8083/api/v1/debug/simulate -Headers $HEADERS -ContentType application/json -Body (@{ zone_id = 1; type = "SENSOR_IOT_LLENO"; points = 5 } | ConvertTo-Json)

# Configurar pesos
Invoke-RestMethod -Method POST -Uri http://localhost:8083/api/v1/config/weights -Headers $HEADERS -ContentType application/json -Body (@{ SENSOR_IOT_LLENO = 10; REPORTE_CIUDADANO = 2 } | ConvertTo-Json)

# Trigger manual
Invoke-RestMethod -Method POST -Uri http://localhost:8083/api/v1/zones/1/trigger -Headers $HEADERS -ContentType application/json -Body (@{ reason = "Emergencia Sanitaria"; operator_override = "CHOFER_01" } | ConvertTo-Json)
```

## Validación de cumplimiento Gherkin

| Escenario | Endpoint | Status esperado | Validación |
|-----------|----------|----------------|-----------|
| 1. Listar zonas | GET /api/v1/zones | 200 OK | ✅ Incluye id, name, type, geometry |
| 2. Ver métricas | GET /api/v1/zones/1/metrics | 200 OK | ✅ Formato exacto: zone_id, name, current_score, threshold, status, last_updated |
| 3. Simular incidente | POST /api/v1/debug/simulate | 202 Accepted | ✅ Acepta zone_id+points, incrementa score |
| 4. Configurar pesos | POST /api/v1/config/weights | 200 OK | ✅ Mensaje "Pesos actualizados correctamente" |
| 5. Botón pánico | POST /api/v1/zones/1/trigger | 201 Created | ✅ Retorna work_order_id, actualiza status a EN_PROGRESO |

## Tests automatizados

Los tests unitarios existentes en `internal/service/incident_trigger_test.go` siguen pasando:
```powershell
go test ./...
# ok github.com/Andres09xZ/latacunga_clean_app/schedule-service/internal/service 0.791s
```

## Próximos pasos (opcional)

1. Agregar endpoint GET /api/v1/work_orders para listar órdenes generadas
2. Agregar endpoint GET /api/v1/work_orders/{id} para detalle de orden
3. Agregar filtros en ListZones por status
4. Webhook notification cuando se genera orden automáticamente
5. Dashboard frontend consumiendo estos endpoints

## Soporte

Si encuentras errores:
1. Verifica migración 009 ejecutada
2. Confirma que zona ID 1 existe con threshold > 0
3. Verifica JWT_SECRET en .env
4. Revisa logs del servicio para errores de DB

Build status: ✅ PASS  
Tests status: ✅ PASS  
Endpoints: ✅ 5/5 implementados según Gherkin
