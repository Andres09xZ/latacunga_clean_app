# 🚀 Quick Start - Prueba E2E

## Ejecución Rápida (3 comandos)

```powershell
# 1. Levantar todos los servicios
docker-compose up -d

# 2. Esperar 30 segundos para inicialización
Start-Sleep -Seconds 30

# 3. Ejecutar prueba E2E con verificación automática
.\run_e2e_test.ps1
```

## Alternativa: Ejecución Manual

```powershell
# Ejecutar directamente el script Go
go run e2e_test.go
```

## Resultados Esperados

✅ **ÉXITO** - Verás este mensaje al final:
```
╔════════════════════════════════════════════════════════════════╗
║                  ✓ PRUEBA E2E EXITOSA ✓                       ║
╚════════════════════════════════════════════════════════════════╝

🎉 ¡TESIS FUNCIONAL! 🎉
```

❌ **FALLO** - El script se detendrá en el paso donde ocurrió el error con un mensaje en rojo.

## Verificación de Servicios

```powershell
# Verificar que todos los servicios estén corriendo
docker-compose ps

# Ver logs de un servicio específico
docker-compose logs -f <servicio>

# Ejemplos:
docker-compose logs -f auth-service
docker-compose logs -f operations-service
docker-compose logs -f scheduler-service
```

## Acceso a Herramientas

| Herramienta | URL | Credenciales |
|-------------|-----|--------------|
| RabbitMQ Management | http://localhost:15672 | tesis / tesis |
| Swagger Auth | http://localhost:8080/swagger/index.html | - |
| Swagger Fleet | http://localhost:8081/swagger/index.html | - |
| Swagger Operations | http://localhost:8085/swagger/index.html | - |

## Troubleshooting Rápido

### Problema: "Service NOT AVAILABLE"
```powershell
# Solución: Reiniciar servicios
docker-compose down
docker-compose up -d
Start-Sleep -Seconds 30
```

### Problema: "No se encontró orden activa"
```powershell
# Verificar que Scheduler haya procesado incidentes
curl http://localhost:8083/api/v1/zones/1/metrics

# Verificar RabbitMQ
# Ir a: http://localhost:15672 -> Queues
# Debe haber mensajes en las colas
```

### Problema: Datos duplicados
```powershell
# Limpiar datos de prueba
.\cleanup_e2e_data.ps1
```

## Limpieza de Datos de Prueba

```powershell
# Eliminar todos los datos generados por pruebas E2E
.\cleanup_e2e_data.ps1
```

Este script elimina:
- Operadores con username `test_driver_*`
- Incidentes con descripción `E2E Test`
- Órdenes de trabajo asociadas
- Turnos relacionados

## Personalización Rápida

### Cambiar número de incidentes
Edita `e2e_test.go`, línea ~258:
```go
for i := 0; i < 20; i++ {  // Cambiar de 15 a 20
```

### Cambiar zona de prueba
Edita `e2e_test.go`, línea ~252:
```go
baseLatitude := -0.9400   // Nueva latitud
baseLongitude := -78.6100 // Nueva longitud
```

### Cambiar timeouts
Edita `e2e_test.go`:
```go
time.Sleep(10 * time.Second)  // Aumentar si tu sistema es lento
```

## Logs en Tiempo Real

```powershell
# Ver todos los logs
docker-compose logs -f

# Ver solo errores
docker-compose logs | Select-String "error" -CaseSensitive

# Ver logs de RabbitMQ
docker-compose logs -f rabbitmq
```

## Ejecución Continua (CI/CD)

```powershell
# Ejecutar sin confirmaciones interactivas
.\run_e2e_test.ps1 -NonInteractive

# En un pipeline CI/CD
go run e2e_test.go
if ($LASTEXITCODE -ne 0) { exit 1 }
```

## Verificación Post-Prueba

```powershell
# Ver datos generados en la BD
psql "postgresql://neondb_owner:npg_jnw3bVupEP5i@ep-gentle-pond-adcmrdsv-pooler.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require"

# Queries útiles:
SELECT * FROM users WHERE username LIKE 'test_driver_%';
SELECT * FROM work_orders ORDER BY created_at DESC LIMIT 5;
SELECT * FROM incidents WHERE description LIKE '%E2E Test%';
```

## Métricas de Rendimiento

El script E2E te mostrará:
- ⏱️ Tiempo de respuesta de cada endpoint
- 📊 Estado de cada paso del flujo
- 🔢 IDs generados (operator, shift, work order)
- 📍 Coordenadas de incidentes creados

## Preguntas Frecuentes

**¿Puedo ejecutar la prueba múltiples veces?**
Sí, cada ejecución genera datos únicos con timestamp.

**¿Los datos persisten después de la prueba?**
Sí, usa `cleanup_e2e_data.ps1` para limpiarlos.

**¿Qué pasa si falla en el paso 3?**
Revisa los logs de Scheduler y Validation Service.

**¿Cuánto tiempo toma la prueba completa?**
Aproximadamente 30-40 segundos.

## Recursos Adicionales

- 📘 [Documentación completa](./E2E_README.md)
- 🏗️ [Arquitectura del sistema](./ARCHITECTURE_DIAGRAMS.md)
- 📚 [Índice de documentación](./CENTRAL_DOCUMENTATION_INDEX.md)

---

**Última actualización:** 2025-11-23  
**Versión:** 1.0.0
