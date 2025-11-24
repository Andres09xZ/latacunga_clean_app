# 🚗 Routing Service - Integración con OSRM

## 📋 Descripción

Microservicio que actúa como interfaz para OSRM (Open Source Routing Machine) para optimizar rutas de recolección de basura en Latacunga.

**Tecnologías:**
- **Go 1.21+**
- **PostgreSQL** (Neon)
- **RabbitMQ** (mensajería asíncrona)
- **OSRM** (optimización de rutas)

---

## 🏗️ Arquitectura

```
┌─────────────────┐
│ Schedule Service│
│  (Publisher)    │
└────────┬────────┘
         │ planning.run.requested.v1
         ▼
┌────────────────────────────────┐
│  RabbitMQ Exchange             │
│  city.cleaning.planning        │
└────────┬───────────────────────┘
         │
         ▼
┌────────────────────────────────┐
│  q.routing.plan-requests       │
│  (Queue)                       │
└────────┬───────────────────────┘
         │
         ▼
┌────────────────────────────────┐
│  Routing Service               │
│  1. Consume message            │
│  2. Call OSRM /trip            │
│  3. Save to PostgreSQL         │
│  4. Publish result             │
└────────┬───────────────────────┘
         │
         ▼
┌────────────────────────────────┐
│  OSRM Server                   │
│  http://localhost:5000/trip    │
└────────┬───────────────────────┘
         │
         ▼
┌────────────────────────────────┐
│  PostgreSQL (route_plans)      │
└────────────────────────────────┘
         │
         ▼
┌────────────────────────────────┐
│  RabbitMQ Exchange             │
│  city.cleaning.routes          │
│  routes.plan.created.v1        │
└────────────────────────────────┘
```

---

## 🚀 Instalación y Configuración de OSRM

### Opción 1: Docker (Recomendado)

1. **Descargar datos de Ecuador (OSM)**:
```bash
wget http://download.geofabrik.de/south-america/ecuador-latest.osm.pbf
```

2. **Preprocesar datos con OSRM**:
```bash
docker run -t -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-extract -p /opt/car.lua /data/ecuador-latest.osm.pbf
docker run -t -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-partition /data/ecuador-latest.osrm
docker run -t -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-customize /data/ecuador-latest.osrm
```

3. **Ejecutar servidor OSRM**:
```bash
docker run -t -i -p 5000:5000 -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-routed --algorithm mld /data/ecuador-latest.osrm
```

4. **Probar que funciona**:
```bash
# Probar ruta entre dos puntos en Latacunga
curl "http://localhost:5000/route/v1/driving/-78.614,-0.933;-78.600,-0.920?overview=full"
```

### Opción 2: Docker Compose

Crear `docker-compose.yml`:
```yaml
version: '3.8'
services:
  osrm:
    image: ghcr.io/project-osrm/osrm-backend:latest
    ports:
      - "5000:5000"
    volumes:
      - ./osrm-data:/data
    command: osrm-routed --algorithm mld /data/ecuador-latest.osrm
    restart: unless-stopped
```

Ejecutar:
```bash
docker-compose up -d
```

---

## ⚙️ Configuración del Routing Service

1. **Copiar archivo de entorno**:
```bash
cp .env.example .env
```

2. **Editar `.env`**:
```env
DB_URL=postgresql://user:password@ep-xxxxx.us-east-2.aws.neon.tech/latacunga_db?sslmode=require
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
OSRM_URL=http://localhost:5000
```

3. **Instalar dependencias**:
```bash
go mod download
```

4. **Ejecutar migraciones** (opcional, se ejecutan automáticamente):
```bash
# Si tienes psql instalado
psql $DB_URL -f migrations/001_create_route_plans.sql
```

5. **Ejecutar el servicio**:
```bash
go run cmd/server/main.go
```

---

## 📡 Flujo de Mensajería

### Input: `planning.run.requested.v1`

**Exchange**: `city.cleaning.planning`  
**Queue**: `q.routing.plan-requests`  
**Routing Key**: `planning.run.requested.v1`

**Payload**:
```json
{
  "request_id": "req-2024-11-22-001",
  "zone_id": 3,
  "points": [
    {"latitude": -0.933, "longitude": -78.614},
    {"latitude": -0.925, "longitude": -78.620},
    {"latitude": -0.940, "longitude": -78.605}
  ]
}
```

### Output: `routes.plan.created.v1`

**Exchange**: `city.cleaning.routes`  
**Routing Key**: `routes.plan.created.v1`

**Payload**:
```json
{
  "request_id": "req-2024-11-22-001",
  "zone_id": 3,
  "distance_meters": 2450.5,
  "duration_seconds": 420.2,
  "geometry": "}_p_Iv_~Kp@_@SENEd@c@...",
  "waypoint_order": [0, 2, 1],
  "optimized_at": "2024-11-22T10:30:00Z"
}
```

---

## 🧪 Pruebas

### 1. Probar OSRM directamente

```bash
# Endpoint /trip con 3 puntos en Latacunga
curl "http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.620,-0.925;-78.605,-0.940?source=first&geometries=polyline&overview=full"
```

**Respuesta esperada**:
```json
{
  "code": "Ok",
  "trips": [{
    "geometry": "}_p_Iv_~Kp@_@...",
    "distance": 2450.5,
    "duration": 420.2,
    "waypoints": [
      {"waypoint_index": 0, "trips_index": 0, "location": [-78.614, -0.933]},
      {"waypoint_index": 2, "trips_index": 0, "location": [-78.605, -0.940]},
      {"waypoint_index": 1, "trips_index": 0, "location": [-78.620, -0.925]}
    ]
  }]
}
```

### 2. Probar el Routing Service con RabbitMQ

```bash
# Publicar mensaje de prueba en RabbitMQ
# (requiere tener rabbitmqadmin instalado)
rabbitmqadmin publish exchange=city.cleaning.planning routing_key=planning.run.requested.v1 payload='{"request_id":"test-001","zone_id":1,"points":[{"latitude":-0.933,"longitude":-78.614},{"latitude":-0.925,"longitude":-78.620}]}'
```

### 3. Verificar logs del servicio

```
[INFO] Optimizando ruta para 2 puntos (RequestID: test-001, ZoneID: 1)...
[SUCCESS] Ruta optimizada guardada: ID=123e4567-e89b-12d3-a456-426614174000, Distance=1234.56m, Duration=180.50s
[PUBLISHED] Evento routes.plan.created.v1 publicado: RequestID=test-001
✅ Ruta procesada y publicada: RequestID=test-001
```

### 4. Verificar en la base de datos

```sql
SELECT * FROM route_plans ORDER BY created_at DESC LIMIT 1;
```

---

## 🗂️ Estructura de Archivos

```
routing-service/
├── cmd/
│   └── server/
│       └── main.go                  # Punto de entrada
├── internal/
│   ├── models/
│   │   └── route.go                 # Structs: RouteRequest, RoutePlan, RouteResponse
│   ├── osrm/
│   │   └── client.go                # Cliente HTTP para OSRM
│   ├── service/
│   │   └── router.go                # Lógica de negocio
│   ├── messaging/
│   │   └── rabbitmq.go              # Consumer/Producer
│   └── database/
│       └── database.go              # Conexión y migraciones
├── migrations/
│   └── 001_create_route_plans.sql  # Schema SQL
├── docs/
│   └── OSRM_SETUP.md                # Documentación OSRM
├── .env.example                     # Template de variables
├── go.mod                           # Dependencias
└── README.md                        # Este archivo
```

---

## 🐛 Troubleshooting

### Problema: OSRM no responde

**Solución**:
```bash
# Verificar que el contenedor está corriendo
docker ps | grep osrm

# Ver logs
docker logs <container-id>

# Reiniciar contenedor
docker restart <container-id>
```

### Problema: Error "No route found"

**Causa**: Los puntos están muy alejados o no hay calles conectadas.

**Solución**:
- Verifica que los puntos estén dentro de Ecuador
- Asegúrate de que los puntos están cerca de calles (no en medio del campo)
- Prueba con coordenadas en el centro de Latacunga: `-78.614, -0.933`

### Problema: Geometría vacía

**Causa**: Falta el parámetro `overview=full` en la URL de OSRM.

**Solución**: Verificar que el cliente incluye `overview=full` en la petición.

---

## 📊 Ejemplo Completo de Uso

### 1. Iniciar OSRM
```bash
docker run -t -i -p 5000:5000 -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-routed --algorithm mld /data/ecuador-latest.osrm
```

### 2. Iniciar RabbitMQ
```bash
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

### 3. Configurar BD (Neon PostgreSQL)
```sql
-- Ejecutar migration
-- migrations/001_create_route_plans.sql
```

### 4. Iniciar Routing Service
```bash
go run cmd/server/main.go
```

### 5. Enviar solicitud (desde Schedule Service)
```go
// El Schedule Service publica este mensaje:
message := RouteRequest{
    RequestID: "daily-route-001",
    ZoneID:    3,
    Points: []Point{
        {Latitude: -0.933, Longitude: -78.614}, // Garaje
        {Latitude: -0.925, Longitude: -78.620}, // Punto 1
        {Latitude: -0.940, Longitude: -78.605}, // Punto 2
    },
}
```

### 6. Ver resultado
El Routing Service publicará:
```json
{
  "request_id": "daily-route-001",
  "zone_id": 3,
  "distance_meters": 3450.8,
  "duration_seconds": 520.5,
  "geometry": "}_p_Iv_~Kp@_@SENEd@c@...",
  "waypoint_order": [0, 2, 1],
  "optimized_at": "2024-11-22T15:30:00Z"
}
```

La geometría `}_p_Iv_~Kp@_@...` es el **dibujo exacto** de la ruta por las calles de Latacunga que se puede visualizar en un mapa usando bibliotecas como Leaflet o Mapbox.

---

## 🔗 Referencias

- [OSRM Documentation](http://project-osrm.org/)
- [OSRM API Reference](https://project-osrm.org/docs/v5.24.0/api/)
- [Polyline Encoding](https://developers.google.com/maps/documentation/utilities/polylinealgorithm)
- [Geofabrik OSM Data](https://download.geofabrik.de/)

---

**Autor**: Backend Team  
**Versión**: 1.0  
**Fecha**: 2024-11-22
