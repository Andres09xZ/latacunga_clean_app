# ✅ Checklist de Verificación - Sistema E2E

## 📋 Pre-Ejecución

### Servicios Requeridos

- [ ] Docker Desktop está corriendo
- [ ] Docker Compose está instalado
- [ ] Go 1.21+ está instalado
- [ ] PostgreSQL client (psql) está instalado (opcional)

### Servicios del Sistema

```powershell
# Verificar servicios activos
docker-compose ps
```

- [ ] Auth Service (puerto 8080)
- [ ] Fleet Service (puerto 8081)
- [ ] Incident Service (puerto 8082)
- [ ] Scheduler Service (puerto 8083)
- [ ] Operations Service (puerto 8085)
- [ ] RabbitMQ (puertos 5672, 15672)
- [ ] PostgreSQL (Neon Cloud)

### Health Checks

```powershell
# Verificar manualmente cada servicio
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
curl http://localhost:8085/health
curl http://localhost:15672
```

- [ ] Auth Service: HTTP 200 OK
- [ ] Fleet Service: HTTP 200 OK
- [ ] Incident Service: HTTP 200 OK
- [ ] Scheduler Service: HTTP 200 OK
- [ ] Operations Service: HTTP 200 OK
- [ ] RabbitMQ Management: HTTP 200 OK

### Variables de Entorno

Verificar que cada servicio tenga su `.env` configurado:

- [ ] auth-service/.env
- [ ] fleet-service/.env
- [ ] incident-service/.env
- [ ] operations-service/.env
- [ ] routing-service/.env
- [ ] schedule-service/.env

### Base de Datos

```powershell
# Verificar conexión a Neon PostgreSQL
psql "postgresql://neondb_owner:npg_jnw3bVupEP5i@ep-gentle-pond-adcmrdsv-pooler.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require" -c "SELECT version();"
```

- [ ] Conexión a PostgreSQL exitosa
- [ ] Esquemas de tablas creados
- [ ] Migraciones aplicadas

### RabbitMQ

Verificar en Management UI (http://localhost:15672):

- [ ] Exchange: city.cleaning.identity
- [ ] Exchange: city.cleaning.operations
- [ ] Exchange: city.cleaning.incidents
- [ ] Queue: q.operations.workorders
- [ ] Queue: q.fleet.drivers
- [ ] Queue: q.scheduler.incidents

### OSRM (Opcional)

Si tienes OSRM configurado localmente:

- [ ] OSRM Service está corriendo
- [ ] Datos de Ecuador cargados
- [ ] Endpoint de routing responde

## 🧪 Durante la Ejecución

### Paso 1: Onboarding del Operador

- [ ] ✅ Status 201 en registro
- [ ] ✅ ID de operador generado
- [ ] ✅ Evento RabbitMQ publicado
- [ ] ✅ Fleet Service consumió el evento
- [ ] ✅ Login exitoso (opcional)

### Paso 2: Inicio de Turno

- [ ] ✅ Status 200 en clock-in
- [ ] ✅ Shift ID generado
- [ ] ✅ Camión LAA-1020 asignado
- [ ] ✅ Conductor marcado como DISPONIBLE

### Paso 3: Generación de Demanda

- [ ] ✅ 15 incidentes creados (Status 201)
- [ ] ✅ Coordenadas en Zona 1 (Latacunga Centro)
- [ ] ✅ Validation Service procesó incidentes
- [ ] ✅ Scheduler acumuló puntos (score)
- [ ] ✅ Score ≥ 50 (umbral alcanzado)
- [ ] ✅ Zona con status TRIGGERED o LISTO

### Paso 4: Asignación y Ruteo

- [ ] ✅ Scheduler detectó zona triggered
- [ ] ✅ Fleet asignó conductor disponible
- [ ] ✅ Routing Service calculó ruta
- [ ] ✅ Operations Service recibió evento
- [ ] ✅ Work Order creada con status ASIGNADA
- [ ] ✅ Paradas (stops) > 0
- [ ] ✅ Route polyline generado

### Paso 5: Ejecución y Cierre

- [ ] ✅ Orden iniciada (Status EN_PROGRESO)
- [ ] ✅ Todas las paradas completadas
- [ ] ✅ Orden finalizada exitosamente
- [ ] ✅ Evento workorder.completed.v1 publicado
- [ ] ✅ Fleet liberó conductor
- [ ] ✅ Scheduler reinició score de zona (→ 0)

## 📊 Post-Ejecución

### Verificación de Datos

```powershell
# Verificar datos generados en BD
psql "postgresql://..." -c "SELECT * FROM users WHERE username LIKE 'test_driver_%';"
psql "postgresql://..." -c "SELECT * FROM work_orders ORDER BY created_at DESC LIMIT 5;"
psql "postgresql://..." -c "SELECT * FROM incidents WHERE description LIKE '%E2E Test%';"
```

- [ ] Operador de prueba en tabla `users`
- [ ] Perfil de operador en `operator_profiles`
- [ ] Turno en tabla `shifts`
- [ ] 15 incidentes en tabla `incidents`
- [ ] Orden de trabajo en `work_orders`
- [ ] Paradas en `work_order_stops`

### Verificación de RabbitMQ

En Management UI (http://localhost:15672):

- [ ] Mensaje `identity.operator.created.v1` publicado
- [ ] Mensaje `workorders.created.v1` consumido
- [ ] Mensaje `workorder.completed.v1` publicado
- [ ] Colas vacías (mensajes procesados)
- [ ] Sin mensajes en Dead Letter Queue

### Verificación de Logs

```powershell
# Verificar logs de cada servicio
docker-compose logs auth-service | Select-String "error" -CaseSensitive
docker-compose logs fleet-service | Select-String "error" -CaseSensitive
docker-compose logs operations-service | Select-String "error" -CaseSensitive
```

- [ ] Sin errores críticos en logs
- [ ] Eventos RabbitMQ loggeados
- [ ] Timestamps coherentes
- [ ] Trace IDs correlacionados

## 🎉 Resultado Final

### Script E2E

- [ ] ✅ PASO 1 COMPLETADO
- [ ] ✅ PASO 2 COMPLETADO
- [ ] ✅ PASO 3 COMPLETADO
- [ ] ✅ PASO 4 COMPLETADO
- [ ] ✅ PASO 5 COMPLETADO

### Mensaje de Éxito

```
╔════════════════════════════════════════════════════════════════╗
║                  ✓ PRUEBA E2E EXITOSA ✓                       ║
╚════════════════════════════════════════════════════════════════╝

🎉 ¡TESIS FUNCIONAL! 🎉
```

- [ ] ✅ Mensaje final en VERDE
- [ ] ✅ Sin errores ROJOS
- [ ] ✅ Tiempo de ejecución: 30-40 segundos
- [ ] ✅ IDs generados mostrados al final

## 🧹 Limpieza (Opcional)

### Después de la Prueba

```powershell
# Limpiar datos de prueba
.\cleanup_e2e_data.ps1
```

- [ ] Confirmación solicitada
- [ ] Datos de prueba eliminados
- [ ] Reporte de eliminaciones mostrado
- [ ] Base de datos limpia

### Detener Servicios

```powershell
# Detener todos los servicios
docker-compose down

# O detener sin eliminar volúmenes
docker-compose stop
```

- [ ] Servicios detenidos
- [ ] Contenedores removidos (opcional)
- [ ] Volúmenes preservados (opcional)

## 📝 Notas de Ejecución

### Información Generada

Registra los IDs generados durante la prueba:

```
Username: test_driver_TIMESTAMP
Email: test_TIMESTAMP@test.com
Operator ID: ____________________________________
Shift ID: ____________________________________
Work Order ID: ____________________________________
```

### Tiempo de Ejecución

```
Inicio: ______________ (HH:MM:SS)
Fin: ______________ (HH:MM:SS)
Duración Total: ______________ segundos
```

### Observaciones

```
_____________________________________________________________
_____________________________________________________________
_____________________________________________________________
_____________________________________________________________
```

## ❌ Troubleshooting

### Si algo falló, verifica:

- [ ] Todos los servicios están corriendo
- [ ] RabbitMQ está activo
- [ ] PostgreSQL (Neon) está accesible
- [ ] No hay conflictos de puerto
- [ ] Variables de entorno correctas
- [ ] Logs de servicios no muestran errores

### Comandos de Diagnóstico

```powershell
# Ver estado de contenedores
docker-compose ps

# Ver logs en tiempo real
docker-compose logs -f

# Reiniciar un servicio específico
docker-compose restart operations-service

# Ver uso de recursos
docker stats

# Verificar red
docker network ls
docker network inspect backend_latacunga_clean_default
```

## ✅ Aprobación Final

### Criterios de Éxito Cumplidos

- [ ] ✅ Todos los servicios funcionan
- [ ] ✅ Flujo E2E completo sin errores
- [ ] ✅ Datos generados correctamente
- [ ] ✅ Eventos RabbitMQ procesados
- [ ] ✅ Score de zona se reinició
- [ ] ✅ Conductor liberado correctamente
- [ ] ✅ Documentación completa
- [ ] ✅ Scripts auxiliares funcionan

### Firma de Aprobación

```
Ejecutado por: _______________________________
Fecha: _______________________________
Resultado: [  ] EXITOSO  [  ] FALLIDO
Observaciones: _______________________________
_____________________________________________
_____________________________________________
```

---

**🎓 Este checklist valida que tu tesis está funcionalmente completa.**

Si todos los checks están en ✅, tu sistema está listo para:
- Presentación de tesis
- Demo en vivo
- Documentación académica
- Deployment en producción (con ajustes)

**¡Felicitaciones! 🎉**
