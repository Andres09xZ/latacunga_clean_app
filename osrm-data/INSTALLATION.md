# 🗺️ Guía de Instalación OSRM - Motor de Rutas

## 📋 Requisitos Previos

- ✅ Docker Desktop instalado y corriendo
- ✅ PowerShell (Windows)
- ✅ ~500 MB de espacio en disco
- ✅ Conexión a Internet (~80 MB de descarga)

## 🚀 Instalación Rápida (3 minutos)

### Opción A: Script Automático (Recomendado)

```powershell
# 1. Ve a la carpeta osrm-data
cd "D:\Octavo Semestre\Tesis\backend_latacunga_clean\osrm-data"

# 2. Ejecuta el script de setup
.\setup-osrm.ps1

# 3. Levanta el servicio Docker
cd ..
docker-compose up -d routing-engine

# 4. Espera 10 segundos y prueba
.\osrm-data\test-osrm.ps1
```

### Opción B: Paso a Paso Manual

#### 1️⃣ Descargar el Mapa de Ecuador

```powershell
cd "D:\Octavo Semestre\Tesis\backend_latacunga_clean\osrm-data"

Invoke-WebRequest -Uri "http://download.geofabrik.de/south-america/ecuador-latest.osm.pbf" -OutFile "ecuador-latest.osm.pbf"
```

#### 2️⃣ Extraer la Red Vial (2-5 minutos)

```powershell
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-extract -p /opt/car.lua /data/ecuador-latest.osm.pbf
```

**¿Qué hace?** Convierte el archivo `.pbf` de OpenStreetMap en un grafo optimizado para rutas.

#### 3️⃣ Particionar el Grafo (1-2 minutos)

```powershell
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-partition /data/ecuador-latest.osrm
```

**¿Qué hace?** Divide el mapa en celdas jerárquicas para el algoritmo MLD (Multi-Level Dijkstra).

#### 4️⃣ Personalizar Pesos (30 segundos)

```powershell
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-customize /data/ecuador-latest.osrm
```

**¿Qué hace?** Calcula los tiempos de viaje y distancias para cada segmento de calle.

#### 5️⃣ Levantar el Servicio

```powershell
cd ..
docker-compose up -d routing-engine
```

#### 6️⃣ Verificar que Funciona

Espera 10 segundos y abre en el navegador:

```
http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.612,-0.935?source=first
```

Si ves un JSON con `waypoints` y `trips`, ¡Funciona! 🎉

## 🧪 Pruebas

### Test Automático

```powershell
cd "D:\Octavo Semestre\Tesis\backend_latacunga_clean\osrm-data"
.\test-osrm.ps1
```

### Tests Manuales

**Health Check:**
```
http://localhost:5000/health
```

**Ruta simple (2 puntos):**
```
http://localhost:5000/route/v1/driving/-78.614,-0.933;-78.612,-0.935?overview=false
```

**Trip optimizado (5 puntos, orden automático):**
```
http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.612,-0.935;-78.616,-0.930;-78.610,-0.937;-78.618,-0.932?source=first&roundtrip=true
```

**Nearest (snap a calle más cercana):**
```
http://localhost:5000/nearest/v1/driving/-78.614,-0.933
```

## 📚 Endpoints Principales

| Endpoint | Descripción | Uso en el Proyecto |
|----------|-------------|-------------------|
| `/route/v1/driving/` | Ruta más rápida entre 2+ puntos | Navegar camión a zona |
| `/trip/v1/driving/` | TSP - Optimiza orden de visita | **Optimizar rutas de recolección** |
| `/nearest/v1/driving/` | Snap coordenada a calle | Validar puntos de contenedores |
| `/table/v1/driving/` | Matriz de distancias/tiempos | Análisis de cobertura |
| `/match/v1/driving/` | Map matching GPS | Tracking en tiempo real |

## 🔧 Comandos Útiles

**Ver logs del contenedor:**
```powershell
docker logs latacunga_routing -f
```

**Reiniciar el servicio:**
```powershell
docker-compose restart routing-engine
```

**Detener el servicio:**
```powershell
docker-compose down routing-engine
```

**Ver uso de recursos:**
```powershell
docker stats latacunga_routing
```

## ⚠️ Troubleshooting

### Error: "Cannot connect to Docker daemon"
```powershell
# Verifica que Docker Desktop esté corriendo
docker ps
```

### Error: "address already in use"
```powershell
# El puerto 5000 está ocupado, cámbialo en docker-compose.yaml
ports:
  - "5001:5000"  # Usa 5001 externamente
```

### El servicio no responde después de levantar
```powershell
# Espera 10-15 segundos, OSRM carga el grafo en RAM
# Verifica logs:
docker logs latacunga_routing
```

### Quiero actualizar el mapa
```powershell
# Re-descarga y re-procesa
cd osrm-data
Remove-Item ecuador-latest.* -Force
.\setup-osrm.ps1
```

## 📊 Uso de Recursos

- **RAM**: ~500 MB (con mapa de Ecuador)
- **CPU**: Bajo en idle, picos al calcular rutas
- **Disco**: ~500 MB (mapa + archivos procesados)
- **Red**: Solo al descargar mapa inicial

## 🔗 Referencias

- **Documentación OSRM**: https://project-osrm.org/docs/v5.24.0/api/
- **OpenStreetMap Ecuador**: https://www.openstreetmap.org/relation/108089
- **Geofabrik Downloads**: http://download.geofabrik.de/south-america/ecuador.html

## 🎯 Próximos Pasos

1. **Integrar con tu backend Go**: Crear cliente HTTP para llamar OSRM
2. **Implementar TSP**: Usar `/trip` para optimizar rutas diarias
3. **Persistir resultados**: Guardar rutas calculadas en PostgreSQL
4. **Monitorear**: Agregar métricas de uso (rutas/min, tiempo promedio)
