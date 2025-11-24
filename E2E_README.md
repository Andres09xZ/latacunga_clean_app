# 🧪 Prueba E2E - Sistema EPAGAL Latacunga

## 📋 Descripción

Script de prueba End-to-End (E2E) automatizado que valida el flujo completo del sistema de gestión de residuos de Latacunga, desde el registro de un operador hasta la finalización de una orden de trabajo.

## 🎯 Objetivo

Ejecutar una **prueba de humo** (smoke test) que verifica:
- ✅ Comunicación entre todos los microservicios
- ✅ Flujo asíncrono a través de RabbitMQ
- ✅ Integración completa del sistema

## 🏗️ Arquitectura Probada

```
┌─────────────┐     ┌──────────────┐     ┌────────────────┐
│Auth Service │────▶│Fleet Service │────▶│Incident Service│
│   :8080     │     │    :8081     │     │     :8082      │
└─────────────┘     └──────────────┘     └────────────────┘
                            │                     │
                            ▼                     ▼
                    ┌──────────────┐     ┌────────────────┐
                    │Scheduler Svc │◀────│Validation Svc  │
                    │    :8083     │     │   (interno)    │
                    └──────────────┘     └────────────────┘
                            │
                            ▼
                    ┌──────────────┐     ┌────────────────┐
                    │Operations Svc│◀────│Routing Service │
                    │    :8085     │     │   (interno)    │
                    └──────────────┘     └────────────────┘
```

## 📝 Flujo de Prueba (Happy Path)

### **PASO 1: Onboarding del Operador** 🚶
- Registra un nuevo operador con credenciales de prueba
- Publica evento `identity.operator.created.v1` a RabbitMQ
- Fleet Service consume el evento y crea el Driver
- **Espera:** 3 segundos para sincronización

### **PASO 2: Inicio de Turno** 🚛
- Realiza clock-in del operador con camión LAA-1020
- Marca al conductor como DISPONIBLE en Fleet
- **Resultado:** Shift activo y camión asignado

### **PASO 3: Generación de Demanda** 📍
- Crea 15 incidentes en Zona 1 (Centro de Latacunga)
- Validation Service valida y calcula gravedad
- Scheduler Service acumula puntos (score)
- **Espera:** 8 segundos para procesamiento
- **Umbral:** Score ≥ 50 → Zona TRIGGERED

### **PASO 4: Asignación y Ruteo** 🗺️
- Scheduler detecta zona triggered
- Fleet asigna conductor disponible
- Routing Service calcula ruta óptima con OSRM
- Operations Service crea Work Order
- **Espera:** 10 segundos para cálculo de ruta
- **Validación:** Orden ASIGNADA con stops y polyline

### **PASO 5: Ejecución y Cierre** ✅
- Inicia la orden (status → EN_PROGRESO)
- Completa todas las paradas secuencialmente
- Finaliza la orden
- Publica evento `workorder.completed.v1`
- Fleet libera conductor (DISPONIBLE)
- Scheduler reinicia score de zona (→ 0)

## 🚀 Cómo Ejecutar

### Prerrequisitos

1. **Todos los servicios levantados:**
```powershell
docker-compose up -d
```

2. **Verificar servicios activos:**
```powershell
# Auth Service
curl http://localhost:8080/health

# Fleet Service
curl http://localhost:8081/health

# Incident Service
curl http://localhost:8082/health

# Scheduler Service
curl http://localhost:8083/health

# Operations Service
curl http://localhost:8085/health
```

3. **RabbitMQ funcionando:**
- Management UI: http://localhost:15672
- Credentials: `tesis / tesis`

### Ejecutar la Prueba

```powershell
# Desde el directorio raíz del proyecto
go run e2e_test.go
```

## 📊 Salida Esperada

```
╔════════════════════════════════════════════════════════════════╗
║          PRUEBA E2E - SISTEMA EPAGAL LATACUNGA               ║
║          Smart City Waste Management Platform                 ║
╚════════════════════════════════════════════════════════════════╝

→ Verificando conectividad de servicios...
✓ Auth Service: OK
✓ Fleet Service: OK
✓ Incident Service: OK
✓ Scheduler Service: OK
✓ Operations Service: OK

═══════════════════════════════════════════════════════
PASO 1: Onboarding del Operador (Auth -> Fleet)
═══════════════════════════════════════════════════════
→ Request: POST http://localhost:8080/api/v1/auth/operators
✓ Registro de operador - Status: 201 ✓
✓ Operador registrado con ID: 550e8400-e29b-41d4-a716...
✓ PASO 1 COMPLETADO ✓

... (continúa con todos los pasos)

╔════════════════════════════════════════════════════════════════╗
║                  ✓ PRUEBA E2E EXITOSA ✓                       ║
╚════════════════════════════════════════════════════════════════╝

🎉 ¡TESIS FUNCIONAL! 🎉
```

## 🎨 Códigos de Color

- 🟢 **Verde**: Operación exitosa
- 🔴 **Rojo**: Error crítico (detiene la prueba)
- 🟡 **Amarillo**: Información
- 🔵 **Azul**: Encabezados de pasos
- 🟣 **Púrpura**: Datos JSON

## ❌ Troubleshooting

### Error: "Service NOT AVAILABLE"
**Causa:** Servicio no está corriendo
**Solución:**
```powershell
docker-compose up -d
docker-compose ps  # Verificar estado
docker-compose logs <servicio>  # Ver logs
```

### Error: "No se encontró orden activa" (Paso 4)
**Posibles causas:**
1. **Score insuficiente:** Menos de 15 incidentes creados
2. **RabbitMQ no está corriendo:** Eventos no se propagan
3. **Routing Service falló:** No pudo calcular ruta
4. **OSRM no está disponible:** Servicio de rutas caído

**Diagnóstico:**
```powershell
# Ver métricas de zona
curl http://localhost:8083/api/v1/zones/1/metrics

# Ver logs del Scheduler
docker-compose logs scheduler-service

# Ver logs de Routing
docker-compose logs routing-service

# Verificar RabbitMQ
# http://localhost:15672 -> Queues
```

### Error: "Timeout" o "Connection Refused"
**Causa:** Puerto incorrecto o firewall
**Solución:**
```powershell
# Verificar puertos en uso
netstat -ano | findstr "8080 8081 8082 8083 8085"

# Verificar variables de entorno
cat operations-service/.env
```

## 📈 Métricas Clave

| Métrica | Valor Esperado |
|---------|----------------|
| Tiempo total de ejecución | ~30-40 segundos |
| Incidentes creados | 15 |
| Score final de zona | ≥ 50 (para trigger) |
| Paradas en Work Order | ≥ 1 |
| Status final conductor | DISPONIBLE |
| Score final zona | 0 (reiniciado) |

## 🔧 Personalización

### Cambiar zona de prueba
```go
// En paso3_GenerarDemanda()
baseLatitude := -0.9346   // Centro de Latacunga (Zona 1)
baseLongitude := -78.6156

// Para Zona 2 (La Laguna):
baseLatitude := -0.9400
baseLongitude := -78.6100
```

### Cambiar número de incidentes
```go
// En paso3_GenerarDemanda()
for i := 0; i < 20; i++ {  // Aumentar a 20
    ...
}
```

### Cambiar camión
```go
// En paso2_IniciarTurno()
TruckPlate: "LAA-1021",  // Usar otro camión
```

## 📚 Contexto de la Tesis

Este script valida la **hipótesis central** de la tesis:

> *"Un sistema de microservicios basado en eventos puede coordinar eficientemente la recolección de residuos utilizando datos de reportes ciudadanos, asignación inteligente y ruteo optimizado."*

### Validaciones Técnicas

✅ **Arquitectura de Microservicios:** 7 servicios independientes
✅ **Event-Driven:** RabbitMQ como message broker
✅ **Asincronía:** Procesamiento sin bloqueos
✅ **Escalabilidad:** Servicios pueden desplegarse independientemente
✅ **Resiliencia:** Reintentos y timeouts configurables
✅ **Ruteo Optimizado:** Integración con OSRM
✅ **Gestión de Recursos:** Asignación dinámica de conductores

## 🤝 Contribución

Para agregar nuevas pruebas:

1. Crear nueva función `pasoN_Descripcion()`
2. Agregar validaciones con `assertStatus()`
3. Usar `printSuccess()` / `printError()` para feedback
4. Llamar desde `main()`

## 📞 Soporte

Si la prueba falla:

1. Revisa logs de cada servicio
2. Verifica RabbitMQ Management UI
3. Consulta variables de entorno (.env)
4. Valida conexión a PostgreSQL (Neon)

## 📝 Notas

- El script crea datos de prueba únicos cada vez (timestamp)
- Los IDs generados se imprimen al final
- Puedes ejecutar múltiples veces sin conflictos
- Los datos persisten en la BD (no se limpian automáticamente)

## ✅ Checklist de Éxito

- [ ] Todos los servicios responden HTTP 200 en `/health`
- [ ] RabbitMQ está corriendo (puerto 5672 y 15672)
- [ ] PostgreSQL (Neon) está accesible
- [ ] OSRM está configurado con datos de Ecuador
- [ ] Script ejecuta sin errores rojos
- [ ] Mensaje final: "🎉 ¡TESIS FUNCIONAL! 🎉"

---

**Autor:** Andrés Sebastián  
**Proyecto:** Sistema de Gestión Inteligente de Residuos - EPAGAL Latacunga  
**Universidad:** [Tu Universidad]  
**Año:** 2025
