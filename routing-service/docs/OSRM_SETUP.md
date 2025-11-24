# 🗺️ Guía de Configuración de OSRM para Latacunga

## 📍 Contexto

OSRM (Open Source Routing Machine) es un motor de enrutamiento que calcula rutas óptimas en redes viales usando datos de OpenStreetMap.

Para el proyecto Latacunga Clean App, lo usamos para:
- **Optimizar el orden de visitas** (Traveling Salesman Problem)
- **Calcular distancias y tiempos reales** por calles
- **Obtener geometrías de rutas** (polylines) para visualización en mapas

---

## 🚀 Instalación Rápida (Docker)

### Paso 1: Descargar datos de OSM

Opciones según cobertura:

#### Opción A: Solo Ecuador (Recomendado)
```bash
wget http://download.geofabrik.de/south-america/ecuador-latest.osm.pbf
```
**Tamaño**: ~100 MB  
**Tiempo de procesamiento**: 2-5 minutos

#### Opción B: Toda Sudamérica (si planeas expandir)
```bash
wget http://download.geofabrik.de/south-america-latest.osm.pbf
```
**Tamaño**: ~1.5 GB  
**Tiempo de procesamiento**: 15-30 minutos

#### Opción C: Solo Latacunga (archivo personalizado)
Si tienes un `.osm` o `.pbf` personalizado de Latacunga, úsalo directamente.

---

### Paso 2: Preprocesar los datos

OSRM requiere 3 pasos de preprocesamiento:

```bash
# 1. Extract: Convierte OSM a formato OSRM
docker run -t -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-extract -p /opt/car.lua /data/ecuador-latest.osm.pbf

# 2. Partition: Prepara para algoritmo MLD (rápido)
docker run -t -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-partition /data/ecuador-latest.osrm

# 3. Customize: Optimiza para consultas
docker run -t -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-customize /data/ecuador-latest.osrm
```

**Nota Windows PowerShell**: Usar `${PWD}` funciona, o reemplaza con ruta absoluta:
```powershell
docker run -t -v "D:/path/to/data:/data" ...
```

---

### Paso 3: Ejecutar servidor OSRM

```bash
docker run -t -i -p 5000:5000 -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-routed --algorithm mld /data/ecuador-latest.osrm
```

**Logs esperados**:
```
[info] starting up engines, v5.27.1
[info] Threads: 8
[info] IP address: 0.0.0.0
[info] IP port: 5000
[info] http 1.1 compression handled by zlib version 1.2.11
[info] running and waiting for requests
```

---

## ✅ Verificación

### Test 1: Health check
```bash
curl http://localhost:5000/
```

**Respuesta esperada**:
```
OSRM routing engine
```

### Test 2: Ruta simple entre 2 puntos en Latacunga

```bash
# Centro de Latacunga → Parque Vicente León
curl "http://localhost:5000/route/v1/driving/-78.6174,-0.9346;-78.6150,-0.9335?overview=full&geometries=polyline"
```

**Respuesta esperada** (ejemplo simplificado):
```json
{
  "code": "Ok",
  "routes": [{
    "geometry": "ypkxErxbwK...",
    "distance": 245.7,
    "duration": 45.2
  }]
}
```

### Test 3: Trip con 3 puntos (TSP)

```bash
curl "http://localhost:5000/trip/v1/driving/-78.6174,-0.9346;-78.6100,-0.9250;-78.6200,-0.9400?source=first&geometries=polyline&overview=full"
```

**Interpretación**:
- `source=first`: El primer punto es fijo (ej: garaje del camión)
- `geometries=polyline`: Respuesta codificada (compacta)
- `overview=full`: Incluye toda la geometría de la ruta

---

## 🐳 Docker Compose (Alternativa)

Crear `docker-compose.yml` en la raíz del proyecto:

```yaml
version: '3.8'

services:
  osrm:
    image: ghcr.io/project-osrm/osrm-backend:latest
    container_name: osrm-latacunga
    ports:
      - "5000:5000"
    volumes:
      - ./osrm-data:/data
    command: osrm-routed --algorithm mld /data/ecuador-latest.osrm
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:5000/"]
      interval: 30s
      timeout: 10s
      retries: 3
```

**Uso**:
```bash
# Iniciar
docker-compose up -d

# Ver logs
docker-compose logs -f osrm

# Detener
docker-compose down
```

---

## 🔧 Configuración Avanzada

### Usar perfil de camión de basura

Por defecto, OSRM usa `car.lua`. Para camiones, puedes personalizar:

```bash
# Descargar perfil de camión
wget https://raw.githubusercontent.com/Project-OSRM/osrm-backend/master/profiles/truck.lua

# Usar en extract
docker run -t -v "${PWD}:/data" ghcr.io/project-osrm/osrm-backend osrm-extract -p /data/truck.lua /data/ecuador-latest.osm.pbf
```

**Cambios clave en `truck.lua`**:
- Restricciones de peso
- Altura máxima
- Evitar calles residenciales estrechas

### Optimizar para zona específica

Si solo trabajas en Latacunga, puedes extraer un subconjunto de OSM con `osmium`:

```bash
# Instalar osmium
sudo apt-get install osmium-tool

# Extraer bbox de Latacunga (aproximado)
osmium extract -b -78.70,-1.00,-78.55,-0.90 ecuador-latest.osm.pbf -o latacunga.osm.pbf
```

**Ventajas**:
- Preprocesamiento más rápido
- Menor uso de RAM
- Consultas ligeramente más rápidas

---

## 🧪 Testing desde Routing Service

Crear archivo `test_osrm.go`:

```go
package main

import (
	"fmt"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/routing-service/internal/osrm"
)

func main() {
	client := osrm.NewOSRMClient("http://localhost:5000")

	points := []models.Point{
		{Latitude: -0.9346, Longitude: -78.6174}, // Centro
		{Latitude: -0.9250, Longitude: -78.6100}, // Norte
		{Latitude: -0.9400, Longitude: -78.6200}, // Sur
	}

	resp, err := client.OptimizeRoute(points)
	if err != nil {
		panic(err)
	}

	trip := resp.Trips[0]
	fmt.Printf("Distancia: %.2f metros\n", trip.Distance)
	fmt.Printf("Duración: %.2f segundos\n", trip.Duration)
	fmt.Printf("Geometría: %s\n", trip.Geometry[:50]+"...")
	fmt.Printf("Orden: %v\n", osrm.ExtractWaypointOrder(trip))
}
```

Ejecutar:
```bash
go run test_osrm.go
```

**Output esperado**:
```
Distancia: 3450.80 metros
Duración: 520.50 segundos
Geometría: }_p_Iv_~Kp@_@SENEd@c@...
Orden: [0 2 1]
```

---

## 📊 Interpretación de Resultados

### Waypoint Order

Si envías puntos `[A, B, C]` y el resultado es `[0, 2, 1]`:
- Empezar en `A` (índice 0)
- Luego ir a `C` (índice 2)
- Finalmente `B` (índice 1)

**Ejemplo práctico**:
```
Puntos originales:
0: Garaje (-0.933, -78.614)
1: Reporte 1 (-0.925, -78.620)
2: Reporte 2 (-0.940, -78.605)

Orden optimizado [0, 2, 1]:
1. Salir del garaje
2. Ir a Reporte 2 (más cercano)
3. Después a Reporte 1
```

### Geometría (Polyline)

La cadena `}_p_Iv_~Kp@_@...` es una **polyline codificada**.

**Decodificar** (para visualización):
- **Frontend**: Usar `polyline` (npm) o `L.Polyline.fromEncoded()` (Leaflet)
- **Backend**: Usar biblioteca como `github.com/twpayne/go-polyline`

**Ejemplo con Leaflet**:
```javascript
const encoded = "}_p_Iv_~Kp@_@SENEd@c@...";
const decoded = L.Polyline.fromEncoded(encoded);
map.addLayer(decoded);
```

---

## 🐛 Problemas Comunes

### Error: "No route found"

**Causas**:
1. Puntos están fuera del área de datos OSM
2. Puntos no están cerca de calles
3. No hay conexión vial entre puntos

**Solución**:
```bash
# Verificar que los puntos estén en Ecuador
# Verificar con mapa: https://www.openstreetmap.org/
# Buscar -0.933, -78.614 (centro Latacunga)
```

### Error: "Cannot open file"

**Causa**: El preprocesamiento no se completó.

**Solución**: Verifica que existan estos archivos:
```bash
ls -lh ecuador-latest.osrm*
# Deberías ver:
# ecuador-latest.osrm
# ecuador-latest.osrm.ebg
# ecuador-latest.osrm.icd
# ...
```

### Error: "Out of memory"

**Causa**: RAM insuficiente para procesar el archivo.

**Solución**:
1. Aumentar memoria de Docker (Settings → Resources → Memory: 4GB+)
2. Usar archivo más pequeño (solo Latacunga en lugar de todo Ecuador)

### Latencia alta (>2 segundos)

**Causa**: Algoritmo CH en lugar de MLD.

**Solución**: Asegúrate de usar `--algorithm mld` al ejecutar `osrm-routed`.

---

## 📈 Métricas de Rendimiento

**Hardware típico** (laptop i5, 8GB RAM):

| Operación | Tiempo |
|-----------|--------|
| Extract (Ecuador) | ~3 min |
| Partition | ~1 min |
| Customize | ~30 seg |
| Consulta /trip (3 puntos) | ~50 ms |
| Consulta /trip (10 puntos) | ~150 ms |

**Producción** (servidor dedicado):
- Consultas: <20 ms
- Throughput: >1000 req/s

---

## 🔗 Referencias

- [OSRM GitHub](https://github.com/Project-OSRM/osrm-backend)
- [OSRM API Docs](https://project-osrm.org/docs/v5.24.0/api/)
- [Geofabrik Downloads](https://download.geofabrik.de/)
- [Polyline Encoding](https://developers.google.com/maps/documentation/utilities/polylinealgorithm)
- [OSM Ecuador](https://www.openstreetmap.org/#map=8/-1.000/-78.500)

---

**Próximos pasos**:
1. ✅ Configurar OSRM localmente
2. ✅ Probar endpoints /route y /trip
3. ✅ Integrar con Routing Service
4. 🔄 Configurar en producción (Railway/Render con OSRM prebuilds)
5. 📊 Visualizar rutas en frontend con Leaflet

