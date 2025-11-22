# 🗺️ API de Zonas de Recolección - Schedule Service

## 📋 Descripción

API REST para consultar zonas geográficas de recolección de basura con soporte PostGIS.

**Base URL:** `http://localhost:8083`

---

## 🚀 Endpoints Disponibles

### 1. **Listar todas las zonas**

```http
GET /api/zones
```

**Respuesta:**
```json
[
  {
    "id": 1,
    "zone_name": "RUTA 1 LUNES",
    "route_name": "RUTA_1",
    "schedule_day": 1,
    "points_count": 2,
    "area_km2": 1.1140,
    "created_at": "2025-11-22T01:48:09Z",
    "updated_at": "2025-11-22T01:48:09Z"
  }
]
```

---

### 2. **Obtener zona por ID**

```http
GET /api/zones/{id}
```

**Ejemplo:**
```bash
curl http://localhost:8083/api/zones/1
```

---

### 3. **Buscar zonas por ruta**

```http
GET /api/zones/route/{route_name}
```

**Ejemplo:**
```bash
curl http://localhost:8083/api/zones/route/RUTA_1
```

**Respuesta:** Array de zonas de la RUTA_1 (Lunes, Martes, Miércoles, Jueves, Viernes)

---

### 4. **Buscar zonas por día**

```http
GET /api/zones/day/{day}
```

**Parámetros:**
- `day`: 0=Domingo, 1=Lunes, 2=Martes, 3=Miércoles, 4=Jueves, 5=Viernes, 6=Sábado

**Ejemplo:**
```bash
curl http://localhost:8083/api/zones/day/1
```

**Respuesta:**
```json
{
  "day": 1,
  "day_name": "Lunes",
  "zones": [
    { "id": 1, "zone_name": "RUTA 1 LUNES", ... },
    { "id": 6, "zone_name": "RUTA 2 LUNES", ... }
  ]
}
```

---

### 5. **🔍 Buscar zona por coordenadas (IMPORTANTE)**

```http
GET /api/zones/search?lat={latitude}&lon={longitude}
```

**Ejemplo:**
```bash
# Buscar qué zona cubre el centro de Latacunga
curl "http://localhost:8083/api/zones/search?lat=-0.933&lon=-78.614"
```

**Respuesta:**
```json
{
  "zone_id": 12,
  "zone_name": "RUTA 2 LUNES",
  "route_name": "RUTA_2",
  "day_name": "Lunes",
  "distance_meters": 34.5
}
```

**Caso de uso:** 
- Un ciudadano reporta basura en coordenadas específicas
- El sistema determina automáticamente qué ruta debe recogerla

---

### 6. **📍 Buscar zona más cercana**

```http
GET /api/zones/nearest?lat={latitude}&lon={longitude}
```

**Ejemplo:**
```bash
# Encontrar la zona más cercana (aunque el punto no esté dentro)
curl "http://localhost:8083/api/zones/nearest?lat=-0.950&lon=-78.600"
```

**Respuesta:**
```json
{
  "zone_id": 8,
  "zone_name": "RUTA 2 MIERCOLES",
  "route_name": "RUTA_2",
  "day_name": "Miércoles",
  "distance_meters": 245.8
}
```

---

### 7. **📊 Resumen por rutas**

```http
GET /api/zones/summary/routes
```

**Respuesta:**
```json
[
  {
    "route_name": "RUTA_1",
    "total_zones": 5,
    "total_area_km2": 5.57,
    "avg_points_per_zone": 2.0,
    "zones": "RUTA 1 LUNES, RUTA 1 MARTES, ..."
  }
]
```

---

### 8. **📊 Resumen por días**

```http
GET /api/zones/summary/days
```

**Respuesta:**
```json
[
  {
    "schedule_day": 1,
    "day_name": "Lunes",
    "total_zones": 5,
    "total_area_km2": 4.08,
    "routes": "RUTA_1, RUTA_2, RUTA_3, RUTA_4, RUTA_5"
  }
]
```

---

### 9. **🗺️ Exportar como GeoJSON**

```http
GET /api/zones/geojson
```

**Respuesta:** GeoJSON FeatureCollection completo
```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": 1,
      "geometry": {
        "type": "MultiPolygon",
        "coordinates": [[[-78.61, -0.93], ...]]
      },
      "properties": {
        "zone_name": "RUTA 1 LUNES",
        "route_name": "RUTA_1",
        "schedule_day": 1,
        "points_count": 2,
        "area_km2": 1.1140
      }
    }
  ]
}
```

**Uso:** Cargar directamente en Leaflet, Mapbox, QGIS, etc.

---

## 🧪 Ejemplos de Uso con cURL

### **Ejemplo 1: Buscar qué día recogen en mi casa**

```bash
# Supongamos que vives en lat=-0.925, lon=-78.620
curl "http://localhost:8083/api/zones/search?lat=-0.925&lon=-78.620"

# Respuesta:
# { "zone_name": "RUTA 3 JUEVES", "day_name": "Jueves", ... }
# → Tu basura se recoge los Jueves
```

### **Ejemplo 2: Ver todas las zonas que trabajan el Viernes**

```bash
curl http://localhost:8083/api/zones/day/5

# Respuesta: Lista de 5 rutas que trabajan el viernes
```

### **Ejemplo 3: Obtener mapa completo de la RUTA_2**

```bash
curl http://localhost:8083/api/zones/route/RUTA_2 > ruta2.json
```

---

## 🔌 Integración con Frontend (JavaScript)

### **React / Next.js - Buscar zona por clic en mapa**

```javascript
import { useState } from 'react';

const ZoneSearchMap = () => {
  const [zone, setZone] = useState(null);

  const handleMapClick = async (lat, lon) => {
    const res = await fetch(
      `http://localhost:8083/api/zones/search?lat=${lat}&lon=${lon}`
    );
    const data = await res.json();
    setZone(data);
  };

  return (
    <div>
      {zone && (
        <div className="zone-info">
          <h3>{zone.zone_name}</h3>
          <p>Ruta: {zone.route_name}</p>
          <p>Día de recolección: {zone.day_name}</p>
          <p>Distancia: {zone.distance_meters.toFixed(0)} metros</p>
        </div>
      )}
    </div>
  );
};
```

### **Leaflet - Mostrar zonas en mapa**

```javascript
// Cargar GeoJSON de zonas
fetch('http://localhost:8083/api/zones/geojson')
  .then(response => response.json())
  .then(geojson => {
    L.geoJSON(geojson, {
      style: (feature) => ({
        color: getColorByRoute(feature.properties.route_name),
        weight: 2,
        fillOpacity: 0.3
      }),
      onEachFeature: (feature, layer) => {
        layer.bindPopup(`
          <b>${feature.properties.zone_name}</b><br>
          Ruta: ${feature.properties.route_name}<br>
          Día: ${getDayName(feature.properties.schedule_day)}<br>
          Área: ${feature.properties.area_km2} km²
        `);
      }
    }).addTo(map);
  });

function getColorByRoute(routeName) {
  const colors = {
    'RUTA_1': '#FF5733',
    'RUTA_2': '#33FF57',
    'RUTA_3': '#3357FF',
    'RUTA_4': '#F333FF',
    'RUTA_5': '#FF33A8'
  };
  return colors[routeName] || '#999999';
}
```

---

## 🐘 Consultas SQL Directas (PostgreSQL)

### **Consulta 1: Zonas que intersectan con un círculo de 500m**

```sql
SELECT zone_name, route_name
FROM cleaning_zones
WHERE ST_DWithin(
  geom::geography,
  ST_SetSRID(ST_MakePoint(-78.614, -0.933), 4326)::geography,
  500 -- metros
);
```

### **Consulta 2: Calcular superposición entre zonas**

```sql
SELECT 
  a.zone_name as zona_a,
  b.zone_name as zona_b,
  ST_Area(ST_Intersection(a.geom, b.geom)::geography) / 1000000.0 as overlap_km2
FROM cleaning_zones a, cleaning_zones b
WHERE a.id < b.id
  AND ST_Intersects(a.geom, b.geom)
  AND ST_Area(ST_Intersection(a.geom, b.geom)::geography) > 0
ORDER BY overlap_km2 DESC;
```

### **Consulta 3: Densidad de zonas por kilómetro cuadrado**

```sql
SELECT 
  route_name,
  COUNT(*) as zonas,
  SUM(area_km2) as area_total,
  COUNT(*) / SUM(area_km2) as densidad_zonas_por_km2
FROM cleaning_zones
GROUP BY route_name
ORDER BY densidad_zonas_por_km2 DESC;
```

---

## 🏗️ Estructura de la Base de Datos

### **Tabla: `cleaning_zones`**

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | SERIAL | ID único |
| `zone_name` | VARCHAR(100) | Nombre de la zona |
| `route_name` | VARCHAR(50) | ID de la ruta |
| `schedule_day` | INTEGER | Día de la semana (0-6) |
| `points_count` | INTEGER | Puntos usados para crear geometría |
| `area_km2` | DECIMAL(10,4) | Área en km² |
| `geom` | GEOMETRY(MULTIPOLYGON, 4326) | Geometría espacial |
| `created_at` | TIMESTAMP | Fecha de creación |
| `updated_at` | TIMESTAMP | Última actualización |

### **Vistas Disponibles**

- `zones_by_route` - Resumen agrupado por ruta
- `zones_by_day` - Resumen agrupado por día

---

## 🧪 Testing

### **Probar el servicio**

```bash
# 1. Verificar que el servicio está corriendo
curl http://localhost:8083/health

# 2. Listar zonas
curl http://localhost:8083/api/zones | jq

# 3. Buscar zona en el centro de Latacunga
curl "http://localhost:8083/api/zones/search?lat=-0.933&lon=-78.614" | jq

# 4. Ver resumen de rutas
curl http://localhost:8083/api/zones/summary/routes | jq
```

---

## 📦 Archivos Generados

```
schedule-service/
├── internal/
│   ├── models/
│   │   └── zone.go                    # Modelo CleaningZone
│   ├── repository/
│   │   └── zone_repository.go         # Repositorio con queries PostGIS
│   ├── handlers/
│   │   └── zone_handler.go            # Controladores HTTP
│   └── server/
│       └── server.go                  # Rutas configuradas
├── migrations/
│   ├── 005_create_cleaning_zones_postgis.sql  # Migración
│   └── zonas_recoleccion.geojson              # Datos
└── scripts/
    └── load_zones_to_db.py            # Script de carga
```

---

## 🎯 Casos de Uso

### **1. Ciudadano reporta basura**
- Usuario clic en mapa → coordenadas (lat, lon)
- `GET /api/zones/search?lat=-0.933&lon=-78.614`
- Sistema dice: "Esta ubicación corresponde a RUTA 2 - Recolección los Lunes"

### **2. Operador planifica ruta**
- `GET /api/zones/route/RUTA_3`
- Obtiene todas las zonas de RUTA_3 para visualizar en mapa

### **3. Dashboard de estadísticas**
- `GET /api/zones/summary/routes`
- Muestra gráfico de barras con área cubierta por cada ruta

### **4. Optimización de rutas**
- `GET /api/zones/geojson`
- Exporta todas las zonas para análisis en QGIS o Python

---

## 🔧 Dependencias Requeridas

```bash
go get github.com/paulmach/orb          # Manejo de geometrías
go get gorm.io/driver/postgres          # Driver PostgreSQL
go get gorm.io/gorm                     # ORM
go get github.com/gin-gonic/gin         # Web framework
```

---

## ✅ Checklist de Implementación

- ✅ Tabla `cleaning_zones` creada en Neon PostgreSQL
- ✅ 25 zonas cargadas con geometrías MultiPolygon
- ✅ Índice espacial GIST creado
- ✅ Modelos Go (`zone.go`)
- ✅ Repositorio con queries PostGIS (`zone_repository.go`)
- ✅ Handlers HTTP (`zone_handler.go`)
- ✅ Rutas API configuradas (`server.go`)
- ⏳ Documentación Swagger (TODO)
- ⏳ Tests unitarios (TODO)
- ⏳ Frontend de visualización (TODO)

---

**Autor:** Schedule Service Team  
**Fecha:** 2025-11-22  
**Versión:** 1.0
