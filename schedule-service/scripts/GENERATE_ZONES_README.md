# Generate Zones - Generador de Polígonos de Zonas de Recolección

## 📋 Descripción

Script Python que lee archivos de planificación de rutas (Excel/CSV) y genera automáticamente polígonos geográficos (GeoJSON) representando el área de cobertura de cada ruta por día de la semana.

## 🎯 Características

- ✅ **Geocoding inteligente**: Usa Nominatim OSM para obtener coordenadas
- ✅ **Soporte multi-formato**: Lee archivos CSV y Excel (.xlsx, .xls)
- ✅ **Multi-sheet**: Procesa todas las hojas de archivos Excel
- ✅ **Detección dinámica**: Encuentra automáticamente la columna de barrios
- ✅ **Convex Hull**: Genera polígonos usando algoritmo de envolvente convexa
- ✅ **Rate limiting**: Respeta límites de la API de Nominatim (1.1s entre peticiones)
- ✅ **Salida GeoJSON**: Compatible con mapas (Leaflet, Mapbox, QGIS)

## 🔧 Requisitos

```bash
pip install pandas requests shapely geojson openpyxl
```

O instalar desde requirements.txt:
```bash
pip install -r scripts/requirements.txt
```

## 📂 Estructura de Entrada

El script busca archivos en `./data/raw/`:

```
data/raw/
├── 1.- RUTAS LATERAL ORIENTAL..xlsx
├── 2.- RUTAS LATERAL OCCIDENTAL..xlsx
├── 3.- RUTAS LATERAL NOCTURNA..xlsx
├── PLANIFICACION RUTA 1. BELISARIO_QUEVEDO...xlsx
├── PLANIFICACION RUTA 2. SAN BUENAVENTURA...xlsx
├── PLANIFICACION RUTA 3 PASTOCALLE TOACAZO.xlsx
├── PLANIFICACION RUTA 4. TANICUCHI...xlsx
└── PLANIFICACION RUTA 5. POALÓ...xlsx
```

### Formato de Archivos

- **Columnas**: Debe contener una columna con barrios/lugares/sectores
- **Headers**: Pueden estar en cualquier fila de las primeras 10
- **Sheets**: Excel con múltiples hojas (una por día) son soportados

## 🚀 Uso

```bash
cd schedule-service
python scripts/generate_zones.py
```

El script:
1. Lee todos los archivos CSV/Excel en `./data/raw/`
2. Identifica la ruta y día de cada archivo/hoja
3. Extrae nombres de barrios (máximo 8 por archivo)
4. Geocodifica cada barrio usando Nominatim
5. Genera polígonos usando Convex Hull
6. Guarda resultado en `./migrations/zonas_recoleccion.geojson`

## 📊 Salida

### Archivo GeoJSON

**Ubicación**: `./migrations/zonas_recoleccion.geojson`

**Formato**:
```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": {
        "type": "Polygon",
        "coordinates": [[[-78.61, -0.93], ...]]
      },
      "properties": {
        "zone_name": "RUTA 1 LUNES",
        "points_count": 3,
        "route_id": "RUTA_1",
        "day": "LUNES"
      }
    }
  ]
}
```

### Archivo de Log

**Ubicación**: `./scripts/generate_zones.log`

Contiene:
- Archivos procesados
- Barrios geocodificados
- Coordenadas encontradas
- Errores y advertencias

## 🗺️ Geometrías Generadas

El script crea diferentes tipos de geometrías según el número de puntos:

| Puntos | Geometría | Buffer |
|--------|-----------|--------|
| 1 punto | Círculo | ±1.1 km (0.01°) |
| 2 puntos | Línea + buffer | ±1.1 km (0.01°) |
| 3+ puntos | Convex Hull | ±550 m (0.005°) |

### Convex Hull (Envolvente Convexa)

Es el polígono más pequeño que contiene todos los puntos. Imagina poner tachuelas en un mapa y envolver una cuerda alrededor - esa forma es el Convex Hull.

## ⚙️ Configuración

Puedes modificar estas constantes en el script:

```python
RAW_DATA_DIR = "./data/raw"              # Carpeta con archivos
OUTPUT_FILE = "./migrations/zonas_recoleccion.geojson"
RATE_LIMIT_DELAY = 1.1                   # Segundos entre peticiones
MAX_BARRIOS_PER_FILE = 8                 # Límite de barrios por archivo
ZONE_BUFFER = 0.005                      # Buffer (~500m)
SMALL_ZONE_BUFFER = 0.01                 # Buffer para 1-2 puntos (~1km)
```

## 📈 Ejemplo de Ejecución

```
============================================================
Generate Zones Script - Starting
============================================================
Found 8 data files (CSV and Excel)

============================================================
Processing: PLANIFICACION RUTA 1. BELISARIO_QUEVEDO...xlsx
============================================================

--- Processing sheet: RUTA 1 LUNES ---
Zone identified as: RUTA_1_LUNES
Processing 8 barrios: ['Total', '____________________________________', ...]...
[OK] Geocoded: Total -> (-0.800542, -78.604106)
[OK] Geocoded: ____________________________________ -> (-0.934031, -78.614576)
[RESULT] Geocoded 2 of 8 barrios

============================================================
Generating zone geometries for 29 zones
============================================================

Zone: RUTA_1_LUNES - 2 points
[ZONE] Created buffered line from 2 points (buffer=0.01)
[OK] Created geometry for RUTA_1_LUNES

============================================================
[SUCCESS] Generated 26 zone polygons
[SUCCESS] Output saved to: ./migrations/zonas_recoleccion.geojson
============================================================
```

## 🗺️ Visualización

### Opción 1: QGIS
1. Abrir QGIS
2. Layer → Add Layer → Add Vector Layer
3. Seleccionar `zonas_recoleccion.geojson`
4. Estilizar según `zone_name` o `day`

### Opción 2: geojson.io
1. Ir a [geojson.io](https://geojson.io)
2. Arrastrar el archivo `zonas_recoleccion.geojson`
3. Ver y editar el mapa interactivo

### Opción 3: Code con Leaflet
```javascript
fetch('zonas_recoleccion.geojson')
  .then(res => res.json())
  .then(data => {
    L.geoJSON(data, {
      style: feature => ({
        color: getColorByRoute(feature.properties.route_id),
        weight: 2,
        fillOpacity: 0.3
      })
    }).addTo(map);
  });
```

## 🔍 Limitaciones Actuales

1. **Barrios genéricos**: Los archivos Excel contienen muchas filas como "Chofer:", "Total", etc. que no son barrios reales
2. **Geocoding limitado**: Solo se obtienen 1-3 coordenadas reales por archivo debido a los datos de entrada
3. **Polígonos aproximados**: Con pocos puntos, los polígonos son aproximaciones (círculos o líneas expandidas)

## 💡 Mejoras Sugeridas

Para obtener mejores polígonos:

1. **Usar archivo ETL diferente**: El script `etl_routes_importer.py` ya procesa correctamente los barrios
2. **Extraer coordenadas del SQL**: Leer las coordenadas ya geocodificadas de `seed_routes.sql`
3. **Agrupar por sector**: Usar la columna `sector` para crear polígonos
4. **Manual**: Dibujar polígonos manualmente en QGIS basándose en conocimiento local

## 📞 Soporte

Para más información sobre el procesamiento de rutas, ver:
- `scripts/etl_routes_importer.py` - ETL principal con geocoding completo
- `scripts/README.md` - Documentación del ETL

## 📝 Notas Técnicas

- **CRS**: WGS84 (EPSG:4326) - lat/lon
- **Formato**: GeoJSON (RFC 7946)
- **Encoding**: UTF-8
- **API**: Nominatim OpenStreetMap
- **Librería geométrica**: Shapely 2.0+

---

**Autor**: Desarrollador Senior  
**Fecha**: 2025-11-22  
**Versión**: 1.0
