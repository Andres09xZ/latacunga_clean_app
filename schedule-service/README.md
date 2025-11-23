# Schedule Service - Planning Core

Servicio de planificación mínima para Latacunga Clean: geolocaliza incidentes, acumula puntaje por zona y dispara eventos cuando se supera el umbral.

## Componentes Clave

- `CleaningZone`: definición geográfica + configuración de horario (`schedule_config`).
- `ZoneMetrics`: puntaje acumulado + umbral + marca de último disparo.
- `PlanningService`: lógica de procesamiento de incidentes y disparo de eventos.
- `RabbitMQ` publisher: emite `planning.resource.requested.v1` al superar umbral o trigger manual.
- BDD (Godog) en `internal/bdd` con escenarios en español.

## Requisitos

- Go >= 1.23
- PostgreSQL + PostGIS (Neon soporta extensiones PostGIS; habilita `CREATE EXTENSION postgis;` si no está activa).
- RabbitMQ (opcional; si no está disponible el publisher se ignora).
- Python (para scripts de carga de zonas si se requiere).

## Configuración `.env`

Ejemplo:
```
DB_URL=postgresql://usuario:password@host/db?sslmode=require
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
PORT=8083
```
Copia el archivo `.env` en la raíz del servicio y ajusta credenciales.

## Migraciones

Las migraciones SQL en `migrations/` se ejecutan automáticamente al iniciar el servicio. Para pruebas puedes saltarlas definiendo `SKIP_MIGRATIONS=1`.

## Ejecutar el servidor

```powershell
# Cargar variables (opcional si usas TestMain que ya invoca godotenv)
$env:DB_URL="postgresql://..."
$env:RABBITMQ_URL="amqp://guest:guest@localhost:5672/"

# Ejecutar
go run ./cmd/server
```

El servidor iniciará en el puerto 8083 y mostrará:
- ✅ Migraciones aplicadas
- 📚 Swagger UI disponible en: **http://localhost:8083/swagger/index.html**
- 🚀 Endpoints Planning Core listos

### Documentación Swagger

Accede a la interfaz interactiva de Swagger en: `http://localhost:8083/swagger/index.html`

Para regenerar la documentación después de cambios en los comentarios:
```powershell
swag init -g cmd/server/main.go -o docs
```

### Endpoints Principales

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/v1/planning/zones` | Lista zonas |
| GET | `/api/v1/planning/zones/{id}/metrics` | Puntaje y umbral |
| PUT | `/api/v1/planning/config/thresholds` | Actualiza umbrales (JSON `{ "URBANO_CENTRAL": 60 }`) |
| POST | `/api/v1/planning/simulate` | Simula incidente `{lat, lon, type}` |
| POST | `/api/v1/planning/zones/{id}/trigger` | Disparo manual `{reason}` |

## Simulación de Incidente

```powershell
Invoke-RestMethod -Method POST -Uri http://localhost:8083/api/v1/planning/simulate -Body '{"lat": -0.930, "lon": -78.615, "type": "SENSOR_LLENO"}' -ContentType 'application/json'
```

## BDD (Godog)

Los escenarios están en `internal/bdd/planning_core.feature`.

Ejecutar pruebas BDD (requiere DB con PostGIS y datos mínimos):
```powershell
$env:DB_URL="postgresql://..."
# Opcional para acelerar: saltar migraciones si ya están aplicadas
$env:SKIP_MIGRATIONS="1"
go test ./internal/bdd -count=1
```

## Pruebas Unitarias

```powershell
# Parser y repositorio (saltará si falta DB_URL)
$env:DB_URL="postgresql://..."; go test ./internal/schedule ./internal/repository -count=1
```

## Carga Inicial de Zonas (Opcional)

Usa scripts en `scripts/`:
```powershell
python .\scripts\seed_geojson.py
```
Asegúrate de tener `requirements.txt` instalado:
```powershell
pip install -r .\scripts\requirements.txt
```

## Eventos RabbitMQ

- Exchange: `planning` (tipo `topic`)
- Routing Key: `planning.resource.requested.v1`
- Payload ejemplo:
```json
{
  "zone": "URBANO_CENTRAL",
  "scheduled_time": "2025-11-22T21:00:00Z",
  "reason": "threshold_exceeded"
}
```

Si RabbitMQ no está accesible se registra una advertencia y el servicio continúa sin publicar.

## Extensiones / Próximos Pasos
- Agregar consumidor de eventos para estadísticas históricas.
- Persistir histórico de triggers.
- Ajustar pesos dinámicos vía endpoint de configuración.

## Troubleshooting

| Problema | Causa | Solución |
|----------|-------|----------|
| `DB_URL no establecido` en pruebas BDD | Falta variable ambiente | Exportar `DB_URL` o definir en `.env` |
| Error de geometría/migración | PostGIS no habilitado | Ejecutar `CREATE EXTENSION postgis;` |
| Evento no publicado | RabbitMQ caído | Verificar `RABBITMQ_URL`, logs de conexión |
| Pruebas lentas | Migraciones repetidas | Usar `SKIP_MIGRITIONS=1` (typo intencional: usar correcto `SKIP_MIGRATIONS`) |

## Licencia
Uso interno académico / municipal. Ajustar según necesidades.
