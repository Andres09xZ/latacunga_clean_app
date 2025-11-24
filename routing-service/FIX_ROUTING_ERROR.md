# Solución: Error "se requieren al menos 2 puntos para optimizar una ruta"

## 🔍 Problema Identificado

El `routing-service` estaba recibiendo mensajes con solo 1 punto en la cola de RabbitMQ, causando el error:

```
❌ Error al optimizar ruta: error al optimizar con OSRM: se requieren al menos 2 puntos para optimizar una ruta
```

### Causa Raíz

Mensajes antiguos en la cola `q.routing.plan-requests` que fueron generados antes de que se implementara la lógica de agregar el punto DEPOT (EPAGAL) automáticamente en el `schedule-service`.

## ✅ Soluciones Implementadas

### 1. Validación en el Consumidor (rabbitmq.go)

Se agregó validación para descartar mensajes con menos de 2 puntos:

```go
// Validar que haya suficientes puntos
if len(req.Points) < 2 {
    log.Printf("⚠️  Mensaje con puntos insuficientes (%d puntos): RequestID=%s ZoneID=%d - Descartando mensaje antiguo",
        len(req.Points), req.RequestID, req.ZoneID)
    msg.Ack(false) // ACK para remover de la cola (mensaje inválido/antiguo)
    continue
}
```

**Beneficio:** Los mensajes antiguos/inválidos se descartan automáticamente sin causar errores.

### 2. Retrocompatibilidad en OptimizeAndSave (router.go)

Se agregó lógica para agregar el DEPOT automáticamente si solo hay 1 punto:

```go
// Validación: si solo hay 1 punto, agregar DEPOT automáticamente (retrocompatibilidad)
points := req.Points
if len(points) == 1 {
    log.Printf("⚠️  Solo 1 punto recibido, agregando DEPOT (EPAGAL) como punto de inicio...")
    depotPoint := models.Point{
        Latitude:  -0.9364043,
        Longitude: -78.6087099,
    }
    points = append([]models.Point{depotPoint}, points...)
    log.Printf("✅ DEPOT agregado. Total puntos: %d", len(points))
}
```

**Beneficio:** Mayor tolerancia a mensajes en diferentes formatos.

### 3. Script de Purga de Cola

Se creó `purge_routing_queue.ps1` para limpiar mensajes viejos de la cola:

```powershell
.\purge_routing_queue.ps1
```

**Uso:**
1. Detener el `routing-service`
2. Ejecutar el script de purga
3. Reiniciar el `routing-service`

## 🚀 Pasos para Resolver el Error

### Opción A: Purgar la Cola (Recomendado)

1. **Detener el routing-service**
   ```powershell
   # Presionar Ctrl+C en la terminal donde está corriendo
   ```

2. **Ejecutar el script de purga**
   ```powershell
   cd routing-service
   .\purge_routing_queue.ps1
   ```

3. **Reiniciar el routing-service**
   ```powershell
   go run .\cmd\server\main.go
   ```

### Opción B: Dejar que se Descarten Automáticamente

Los mensajes antiguos se descartarán automáticamente gracias a la validación agregada. Solo necesitas:

1. **Reiniciar el routing-service**
   ```powershell
   # Detener con Ctrl+C
   go run .\cmd\server\main.go
   ```

Los mensajes con menos de 2 puntos se marcarán como procesados y serán removidos de la cola.

## 📝 Prevención Futura

El `schedule-service` ahora siempre agrega el punto DEPOT en `logic.go` (líneas 100-112):

```go
// Insertar EPAGAL como primer punto (DEPOT_START)
depotPoint := GeoPoint{
    Lat:           -0.9364043,
    Lon:           -78.6087099,
    IncidentID:    "DEPOT_START",
    GravityPoints: 0,
}
points = append(points, depotPoint)
```

Esto garantiza que todos los mensajes nuevos tengan al menos 2 puntos (DEPOT + al menos 1 incidente).

## 🧪 Verificación

Para verificar que el problema está resuelto:

1. **Revisar los logs del routing-service:**
   ```
   ✅ DEPOT agregado. Total puntos: 2
   [INFO] Optimizando ruta para 2 puntos...
   ✅ Ruta procesada y publicada
   ```

2. **Verificar la cola está vacía:**
   - Abrir RabbitMQ Management: http://localhost:15672
   - Ir a "Queues"
   - Verificar que `q.routing.plan-requests` tenga 0 mensajes

## 🔧 Archivos Modificados

1. **routing-service/internal/messaging/rabbitmq.go**
   - Agregada validación de puntos mínimos
   - Cambio de `msg.Nack(false, true)` a `msg.Ack(false)` para mensajes inválidos

2. **routing-service/internal/service/router.go**
   - Agregada lógica de retrocompatibilidad para agregar DEPOT automáticamente

3. **routing-service/purge_routing_queue.ps1** (nuevo)
   - Script para purgar la cola de RabbitMQ

## 📊 Comportamiento Esperado

### Antes (Con error)
```
[RECEIVED] Mensaje con 1 punto
❌ Error: se requieren al menos 2 puntos
❌ Error: se requieren al menos 2 puntos
❌ Error: se requieren al menos 2 puntos (loop infinito)
```

### Después (Funcionando)
```
[RECEIVED] Mensaje con 1 punto
⚠️  Mensaje con puntos insuficientes - Descartando mensaje antiguo
[RECEIVED] Mensaje con 2+ puntos
✅ Ruta procesada y publicada
```

---

**Fecha:** 2025-11-24  
**Autor:** GitHub Copilot  
**Estado:** ✅ Resuelto
