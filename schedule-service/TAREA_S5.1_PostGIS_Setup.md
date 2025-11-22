# 📋 Tarea S5.1: Infraestructura de Datos Espaciales (PostGIS)

## 🎯 Objetivo

Crear la infraestructura de base de datos para almacenar las zonas de recolección con geometrías espaciales usando PostGIS.

---

## 📦 Prerequisitos

### 1. PostgreSQL y PostGIS instalados

**Windows:**
```powershell
# Descargar e instalar desde:
# https://www.postgresql.org/download/windows/
# Durante la instalación, asegúrate de incluir PostGIS
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib postgis
```

**Mac:**
```bash
brew install postgresql postgis
```

### 2. Verificar instalación

```powershell
# Verificar que PostgreSQL está corriendo
# Windows: Servicios -> postgresql-x64-XX debe estar "Ejecutándose"

# O desde línea de comandos:
psql --version
# Debe mostrar: psql (PostgreSQL) 15.x o superior
```

### 3. Python con dependencias

```powershell
# Instalar dependencias
pip install -r scripts/requirements.txt

# Verificar psycopg2
python -c "import psycopg2; print('psycopg2 OK')"
```

---

## 🚀 Pasos de Ejecución

### **Opción A: Ejecución Automatizada (Recomendado)**

El script `load_zones_to_db.py` hace TODO automáticamente:

```powershell
# 1. Editar configuración de base de datos
notepad scripts\load_zones_to_db.py
# Líneas 35-41: Ajusta DB_CONFIG con tus credenciales

# 2. Ejecutar script completo
python scripts\load_zones_to_db.py
```

El script realizará:
- ✅ Conexión a PostgreSQL
- ✅ Verificación de PostGIS
- ✅ Ejecución de migración 005
- ✅ Carga de datos desde GeoJSON
- ✅ Verificación de resultados

---

### **Opción B: Ejecución Manual (Paso a Paso)**

Si prefieres ejecutar cada paso manualmente:

#### **Paso 1: Crear la base de datos**

```powershell
# Conectar a PostgreSQL como superusuario
psql -U postgres

# En el prompt de psql:
CREATE DATABASE city_cleaning;
\c city_cleaning

# Habilitar PostGIS
CREATE EXTENSION IF NOT EXISTS postgis;

# Verificar versión
SELECT PostGIS_Version();

# Salir
\q
```

#### **Paso 2: Ejecutar migración SQL**

```powershell
# Opción 1: Desde línea de comandos
psql -U postgres -d city_cleaning -f migrations\005_create_cleaning_zones_postgis.sql

# Opción 2: Desde psql interactivo
psql -U postgres -d city_cleaning
\i migrations/005_create_cleaning_zones_postgis.sql
\q
```

#### **Paso 3: Verificar tabla creada**

```powershell
psql -U postgres -d city_cleaning -c "\d cleaning_zones"
```

Deberías ver:

```
Column       |          Type           | Nullable |
-------------+-------------------------+----------+
id           | integer                 | not null |
zone_name    | character varying(100)  | not null |
route_name   | character varying(50)   | not null |
schedule_day | integer                 | not null |
points_count | integer                 |          |
area_km2     | numeric(10,4)           |          |
geom         | geometry(MultiPolygon)  | not null |
created_at   | timestamp               |          |
updated_at   | timestamp               |          |
```

#### **Paso 4: Cargar datos desde GeoJSON**

```powershell
# Usando el script Python
python scripts\load_zones_to_db.py
```

---

## 🔍 Verificación de Resultados

### **1. Contar zonas cargadas**

```sql
-- Conectar a la base de datos
psql -U postgres -d city_cleaning

-- Contar zonas
SELECT COUNT(*) as total_zones FROM cleaning_zones;
-- Debe mostrar: 26 zonas
```

### **2. Ver resumen por ruta**

```sql
SELECT * FROM zones_by_route ORDER BY route_name;
```

**Resultado esperado:**
```
route_name | total_zones | total_area_km2 | zones
-----------+-------------+----------------+---------------------------
RUTA_1     |           5 |         0.2512 | RUTA 1 LUNES, RUTA 1 ...
RUTA_2     |           5 |         0.2009 | RUTA 2 LUNES, RUTA 2 ...
RUTA_3     |           5 |         0.2009 | RUTA 3 LUNES, RUTA 3 ...
RUTA_4     |           6 |         0.2411 | RUTA 4 TODOS, RUTA 4 ...
RUTA_5     |           5 |         0.2009 | RUTA 5 LUNES, RUTA 5 ...
```

### **3. Ver resumen por día**

```sql
SELECT * FROM zones_by_day ORDER BY schedule_day;
```

### **4. Probar búsqueda espacial**

```sql
-- Encontrar zona para un punto específico (Centro de Latacunga)
SELECT * FROM find_zone_by_point(-0.933, -78.614);
```

**Resultado esperado:**
```
zone_id | zone_name     | route_name | day_name | distance_meters
--------+---------------+------------+----------+-----------------
12      | RUTA 2 LUNES  | RUTA_2     | Lunes    |            34.5
```

---

## 📊 Consultas Útiles PostGIS

### **Calcular área total cubierta**

```sql
SELECT 
    SUM(ST_Area(geom::geography)) / 1000000.0 as total_km2
FROM cleaning_zones;
```

### **Encontrar zonas superpuestas**

```sql
SELECT 
    a.zone_name as zone_a,
    b.zone_name as zone_b,
    ST_Area(ST_Intersection(a.geom, b.geom)::geography) / 1000000.0 as overlap_km2
FROM cleaning_zones a
JOIN cleaning_zones b ON a.id < b.id
WHERE ST_Intersects(a.geom, b.geom)
ORDER BY overlap_km2 DESC;
```

### **Obtener centro (centroide) de cada zona**

```sql
SELECT 
    zone_name,
    ST_Y(ST_Centroid(geom)) as latitude,
    ST_X(ST_Centroid(geom)) as longitude
FROM cleaning_zones
ORDER BY zone_name;
```

### **Exportar a GeoJSON desde PostgreSQL**

```sql
-- Exportar todas las zonas
COPY (
    SELECT jsonb_build_object(
        'type', 'FeatureCollection',
        'features', jsonb_agg(feature)
    )
    FROM (
        SELECT jsonb_build_object(
            'type', 'Feature',
            'geometry', ST_AsGeoJSON(geom)::jsonb,
            'properties', jsonb_build_object(
                'zone_name', zone_name,
                'route_name', route_name,
                'schedule_day', schedule_day,
                'area_km2', area_km2
            )
        ) as feature
        FROM cleaning_zones
    ) features
) TO 'C:/temp/zones_export.geojson';
```

---

## 🗺️ Visualización

### **Opción 1: pgAdmin 4**

1. Abrir pgAdmin
2. Conectar a `city_cleaning`
3. Click derecho en `cleaning_zones` → View/Edit Data → Geometry Viewer
4. Se mostrará un mapa interactivo con las zonas

### **Opción 2: QGIS**

1. Abrir QGIS
2. Layer → Add Layer → Add PostGIS Layers
3. Configurar conexión:
   - Name: `city_cleaning`
   - Host: `localhost`
   - Database: `city_cleaning`
   - User: `postgres`
4. Connect → Seleccionar `cleaning_zones` → Add

### **Opción 3: Web (Leaflet)**

```javascript
// Ejemplo de código para mostrar las zonas en un mapa web
fetch('/api/zones')
  .then(r => r.json())
  .then(geojson => {
    L.geoJSON(geojson, {
      style: {color: '#ff7800', weight: 2},
      onEachFeature: (feature, layer) => {
        layer.bindPopup(`
          <b>${feature.properties.zone_name}</b><br>
          Ruta: ${feature.properties.route_name}<br>
          Área: ${feature.properties.area_km2} km²
        `);
      }
    }).addTo(map);
  });
```

---

## ❌ Solución de Problemas

### **Error: "database 'city_cleaning' does not exist"**

```powershell
# Crear la base de datos
psql -U postgres -c "CREATE DATABASE city_cleaning;"
```

### **Error: "extension 'postgis' does not exist"**

```powershell
# Instalar PostGIS
# Windows: Reinstalar PostgreSQL con Stack Builder → PostGIS
# Linux: sudo apt-get install postgis postgresql-XX-postgis-3
```

### **Error: "authentication failed for user 'postgres'"**

```powershell
# Editar scripts/load_zones_to_db.py
# Línea 39: Cambiar 'password' por tu contraseña real
```

### **Error: "psycopg2 not found"**

```powershell
pip install psycopg2-binary
```

### **Error: "permission denied to create extension"**

```powershell
# Conectar como superusuario
psql -U postgres -d city_cleaning -c "CREATE EXTENSION postgis;"
```

---

## 📁 Archivos Generados

Después de ejecutar esta tarea, tendrás:

```
schedule-service/
├── migrations/
│   ├── 005_create_cleaning_zones_postgis.sql  ✅ Migración SQL
│   └── zonas_recoleccion.geojson              ✅ Datos fuente
├── scripts/
│   ├── load_zones_to_db.py                    ✅ Script de carga
│   └── load_zones.log                         📄 Log de ejecución
```

---

## ✅ Criterios de Éxito

Tarea S5.1 completada si:

- ✅ Extensión PostGIS habilitada en `city_cleaning`
- ✅ Tabla `cleaning_zones` creada con tipo `GEOMETRY(MULTIPOLYGON, 4326)`
- ✅ Índice espacial GIST creado en columna `geom`
- ✅ 26 zonas cargadas desde el GeoJSON
- ✅ Funciones `find_zone_by_point()` funcionando
- ✅ Vistas `zones_by_route` y `zones_by_day` creadas
- ✅ Trigger `calculate_zone_area` activado

---

## 🔜 Próximos Pasos

Una vez completada esta tarea:

1. **S5.2**: Implementar API REST en Go para consultar zonas
2. **S5.3**: Integrar búsqueda espacial en schedule-service
3. **S5.4**: Crear endpoint `/api/zones/search?lat={lat}&lon={lon}`
4. **S5.5**: Dashboard de visualización de rutas

---

## 📞 Comandos Rápidos de Referencia

```powershell
# Conectar a la base de datos
psql -U postgres -d city_cleaning

# Ver tablas
\dt

# Describir tabla
\d cleaning_zones

# Consultar datos
SELECT zone_name, route_name, area_km2 FROM cleaning_zones LIMIT 5;

# Contar registros
SELECT COUNT(*) FROM cleaning_zones;

# Salir
\q
```

---

## 📝 Notas Importantes

- **SRID 4326**: Sistema de coordenadas WGS84 (GPS estándar)
- **MultiPolygon**: Permite geometrías con múltiples polígonos fusionados
- **GIST Index**: Acelera consultas espaciales (ST_Contains, ST_Intersects, etc.)
- **Geography vs Geometry**: Geography considera curvatura terrestre (más preciso)
- **Buffer orgánico**: Zonas generadas con buffer de 400m por punto

---

**Autor:** Sistema de Zonificación Automática  
**Fecha:** 2025-11-22  
**Versión:** 1.0
