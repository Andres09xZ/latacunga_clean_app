# ETL Routes Importer - Latacunga Garbage Collection

Este script procesa archivos CSV de rutas de recolección de basura para Latacunga, Ecuador, obtiene coordenadas geográficas usando Nominatim y genera un archivo SQL para poblar la base de datos.

## 📋 Requisitos Previos

- Python 3.8 o superior
- pip (gestor de paquetes de Python)
- Conexión a Internet (para geocodificación)

## 🚀 Instalación

1. **Instalar dependencias de Python:**

```bash
cd schedule-service/scripts
pip install -r requirements.txt
```

O instalar manualmente:

```bash
pip install pandas requests
```

## 📁 Estructura de Archivos

```
schedule-service/
├── data/
│   └── raw/                    # Coloca aquí tus archivos CSV
│       ├── 1.- RUTAS LATERAL ORIENTAL...csv
│       ├── 2.- RUTAS LATERAL OCCIDENTAL...csv
│       ├── 3.- RUTAS LATERAL NOCTURNA...csv
│       ├── PLANIFICACION RUTA 1... - RUTA 1 LUNES.csv
│       ├── PLANIFICACION RUTA 1... - RUTA 1 MARTES.csv
│       └── ... (más archivos)
├── migrations/
│   └── seed_routes.sql         # Salida generada
└── scripts/
    ├── etl_routes_importer.py  # Script principal
    ├── requirements.txt         # Dependencias
    └── README.md               # Este archivo
```

## 📝 Formato de Archivos CSV

### Rutas Urbanas (Carga Lateral)

**Nombres esperados:**
- `RUTAS LATERAL ORIENTAL`
- `RUTAS LATERAL OCCIDENTAL`
- `RUTAS LATERAL NOCTURNA`

**Características:**
- Operan todos los días (Lunes a Domingo)
- Horario: 07:30-16:30 (Diurnas) o 21:00-05:00 (Nocturna)

### Rutas Rurales (Carga Posterior)

**Nombres esperados:**
- `RUTA [N] LUNES.csv`
- `RUTA [N] MARTES.csv`
- etc.

**Características:**
- Operan días específicos
- Horario: 07:00-17:00

### Columnas esperadas

Los CSV deben contener una columna con uno de estos nombres:
- `PARROQUIAS/BARRIOS`
- `LUGARES`
- `PARROQUIA`
- `BARRIO`
- `LUGAR`

## ▶️ Uso

1. **Coloca los archivos CSV en `data/raw/`:**

```bash
mkdir -p data/raw
# Copia tus archivos CSV aquí
```

2. **Ejecuta el script:**

```bash
cd schedule-service
python scripts/etl_routes_importer.py
```

3. **Revisa el output:**

El script generará:
- `migrations/seed_routes.sql` - Archivo SQL con todos los INSERT statements
- `migrations/fallback_coordinates.csv` - Reporte de lugares con coordenadas fallback
- `etl_routes_importer.log` - Log detallado de la ejecución

Ejemplo de consola:
```
2025-11-22 00:30:00 - INFO - Geocoding: Barrio Isimbo
2025-11-22 00:30:01 - INFO - [OK] Found: Barrio Isimbo -> (-0.923450, -78.612340)
2025-11-22 00:30:02 - INFO - Geocoding: Oficinas Regional Oriental
2025-11-22 00:30:03 - WARNING - [X] Not found: Oficinas Regional Oriental - using Latacunga center as fallback
2025-11-22 00:30:04 - INFO - [FALLBACK] Using Latacunga center for: Oficinas Regional Oriental
```

4. **Revisa el reporte de fallback:**

```bash
cat migrations/fallback_coordinates.csv
```

Este archivo lista lugares que necesitan corrección manual de coordenadas.

## 🔍 Proceso del ETL

1. **Lectura de CSV/Excel:**
   - Busca automáticamente la fila con los encabezados correctos
   - Ignora filas vacías y datos basura
   - Soporta múltiples hojas (sheets) en archivos Excel

2. **Clasificación:**
   - **LATERAL**: Rutas urbanas que operan todos los días
   - **POSTERIOR**: Rutas rurales que operan días específicos

3. **Geocodificación Inteligente:**
   - Usa Nominatim API (OpenStreetMap)
   - **Estrategias de búsqueda múltiples:**
     1. "Lugar, Latacunga, Cotopaxi, Ecuador"
     2. Sin prefijos (Barrio, Sector, etc.): "Lugar, Latacunga, Ecuador"
     3. "Lugar, Cotopaxi, Ecuador"
     4. "Lugar, Ecuador"
   - Valida que las coordenadas estén en Ecuador
   - **Fallback automático**: Si no encuentra, usa el centro de Latacunga (-0.9346, -78.6156)
   - Rate limiting: 1.2 segundos entre peticiones
   - Caché local para evitar peticiones repetidas

4. **Generación SQL:**
   - Crea tablas `rutas.sectors` y `rutas.route_schedules`
   - Usa PostGIS para almacenar geometrías: `ST_Point(lon, lat)`
   - Maneja conflictos con `ON CONFLICT DO NOTHING`

5. **Reporte de Coordenadas Fallback:**
   - Genera `migrations/fallback_coordinates.csv`
   - Lista todos los lugares que usaron coordenadas del centro de Latacunga
   - Permite corrección manual posterior

## 📊 Ejemplo de Salida SQL

```sql
-- Sector Urbano
INSERT INTO rutas.sectors (id, name, type, days, schedule) 
VALUES ('LATERAL_ORIENTAL', 'Ruta Lateral Oriental', 'lateral', '{0,1,2,3,4,5,6}', '07:30-16:30');

-- Barrio Urbano (se inserta 7 veces, una por cada día)
INSERT INTO rutas.route_schedules (sector_id, landmark_name, landmark_geom, day_of_week) 
VALUES ('LATERAL_ORIENTAL', 'Barrio Isimbo', ST_SetSRID(ST_Point(-78.61234, -0.92345), 4326), 1);

-- Sector Rural
INSERT INTO rutas.sectors (id, name, type, days, schedule) 
VALUES ('RUTA_POST_1', 'Ruta Posterior 1', 'posterior', '{1}', '07:00-17:00');

-- Barrio Rural (solo un día)
INSERT INTO rutas.route_schedules (sector_id, landmark_name, landmark_geom, day_of_week) 
VALUES ('RUTA_POST_1', 'Sigsicalle Sur', ST_SetSRID(ST_Point(-78.65432, -0.95432), 4326), 1);
```

## 🔧 Configuración Avanzada

### Cambiar el delay de rate limiting

Edita la constante en el script:

```python
RATE_LIMIT_DELAY = 1.2  # segundos entre peticiones
```

### Cambiar el User-Agent

```python
USER_AGENT = 'TuApp/1.0'
```

### Cambiar directorios

```python
INPUT_DIR = Path('./data/raw')
OUTPUT_DIR = Path('./migrations')
```

## 🐛 Troubleshooting

### Error: "No CSV files found"

**Solución:** Asegúrate de que los archivos CSV estén en `data/raw/`

```bash
ls data/raw/*.csv
```

### Error: "Could not find header row"

**Problema:** El script no puede encontrar la columna con los nombres de lugares.

**Solución:** 
1. Abre el CSV y verifica que tenga una columna con "PARROQUIAS", "BARRIOS" o "LUGARES"
2. Revisa que no haya caracteres especiales en los encabezados

### Lugares no encontrados por Nominatim

**Comportamiento Actual:** El script usa el centro de Latacunga como fallback automático.

**Proceso de corrección:**

1. **Revisa el reporte de fallback:**
```bash
cat migrations/fallback_coordinates.csv
```

Este archivo contiene todos los lugares que no se pudieron geocodificar y están usando coordenadas del centro de Latacunga.

2. **Corrige las coordenadas manualmente:**

Edita el archivo CSV y actualiza las coordenadas con las correctas. Puedes obtenerlas de:
- Google Maps: Click derecho → "¿Qué hay aquí?"
- OpenStreetMap: https://www.openstreetmap.org
- Coordenadas GPS directas

Ejemplo de `fallback_coordinates_corrected.csv`:
```csv
place_name,latitude,longitude,notes
"Oficinas Regional Oriental",-0.9234,-78.6145,"Coordenada corregida manualmente"
"GAD Parroquial",-0.9156,-78.6234,"Obtenida de Google Maps"
"UPC",-0.9445,-78.6153,"Coordenada verificada"
```

3. **Actualiza el SQL con las coordenadas corregidas:**
```bash
python scripts/update_coordinates.py migrations/fallback_coordinates_corrected.csv
```

Este script:
- Lee las coordenadas corregidas del CSV
- Actualiza el archivo `seed_routes.sql`
- Crea un backup automático: `seed_routes.sql.backup`
- Reporta cuántas coordenadas se actualizaron

4. **Aplica el SQL actualizado a la base de datos**

### Rate limiting por OpenStreetMap

**Síntoma:** "HTTP 429 - Too Many Requests"

**Solución:** Aumenta el `RATE_LIMIT_DELAY` a 2.0 segundos:

```python
RATE_LIMIT_DELAY = 2.0
```

## 📈 Performance

- **Tiempo estimado:** 1-2 minutos por cada 100 lugares únicos
- **API calls:** Máximo 1 por lugar (con caché)
- **Rate limit:** ~50 peticiones por minuto con delay de 1.2s

## 🗄️ Aplicar los Seeds a la Base de Datos

Una vez generado el archivo SQL:

```bash
# Opción 1: Con psql
psql -h localhost -U postgres -d nombre_db -f migrations/seed_routes.sql

# Opción 2: Desde Go migrations
# Copia seed_routes.sql a la carpeta de migrations del proyecto
# Se ejecutará automáticamente al iniciar el servicio
```

## 📝 Log Output

El script genera un log detallado:

```
2025-11-14 10:30:00 - INFO - Processing LATERAL file: 1.- RUTAS LATERAL ORIENTAL.csv
2025-11-14 10:30:01 - INFO - Found header row at index 3
2025-11-14 10:30:01 - INFO - Using column: 'PARROQUIAS/BARRIOS' for place names
2025-11-14 10:30:01 - INFO - Found 45 unique places
2025-11-14 10:30:02 - INFO - Geocoding: Barrio Isimbo
2025-11-14 10:30:03 - INFO - ✓ Found: Barrio Isimbo -> (-0.923450, -78.612340)
...
```

## 🎯 Características Especiales

1. **Geocodificación Multi-estrategia**: 
   - Intenta 4 estrategias diferentes antes de usar fallback
   - Remueve prefijos automáticamente (Barrio, Sector, etc.)
   - Valida que coordenadas estén en Ecuador
   
2. **Fallback Inteligente**: 
   - Nunca falla por coordenadas faltantes
   - Usa centro de Latacunga para lugares no encontrados
   - Genera reporte CSV para corrección manual posterior

3. **Caché de Geocodificación**: Evita peticiones duplicadas

4. **Rate Limiting Automático**: Respeta los límites de Nominatim

5. **Robusto ante errores**: Continúa procesando aunque falle la geocodificación

6. **Duplicados permitidos**: Un barrio puede aparecer en múltiples días

7. **Logging completo**: Cada operación se registra para auditoría

8. **Soporte Multi-hoja**: Procesa archivos Excel con múltiples días

9. **Herramienta de Actualización**: Script para corregir coordenadas después del ETL

## 📞 Soporte

Para problemas o preguntas, revisa:
1. El archivo de log: `etl_routes_importer.log`
2. La documentación de Nominatim: https://nominatim.org/
3. El código fuente comentado en `etl_routes_importer.py`

---

**Autor:** Data Engineering Team  
**Fecha:** 2025-11-14  
**Versión:** 1.0.0
