# OSRM Data - Routing Engine para Latacunga Clean

Esta carpeta contiene los datos procesados de OpenStreetMap para Ecuador, usados por el motor de rutas OSRM.

## 📋 Estructura de Archivos

Después de procesar, encontrarás:
- `ecuador-latest.osm.pbf` - Archivo original de OpenStreetMap
- `ecuador-latest.osrm*` - Archivos procesados por OSRM (extract, partition, customize)

## 🚀 Pasos de Configuración

### 1️⃣ Descargar el Mapa de Ecuador

```powershell
# Desde PowerShell en esta carpeta
Invoke-WebRequest -Uri "http://download.geofabrik.de/south-america/ecuador-latest.osm.pbf" -OutFile "ecuador-latest.osm.pbf"
```

O descarga manualmente desde: http://download.geofabrik.de/south-america/ecuador-latest.osm.pbf

### 2️⃣ Procesar los Datos (3 Comandos)

**Extraer la red vial:**
```powershell
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-extract -p /opt/car.lua /data/ecuador-latest.osm.pbf
```

**Particionar el grafo:**
```powershell
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-partition /data/ecuador-latest.osrm
```

**Personalizar pesos:**
```powershell
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-customize /data/ecuador-latest.osrm
```

### 3️⃣ Levantar el Servicio

Desde la carpeta raíz del proyecto:
```powershell
docker-compose up -d routing-engine
```

### 4️⃣ Probar el Servicio

```bash
# Ruta de prueba en Latacunga
http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.612,-0.935?source=first
```

## 📊 Tamaño Estimado

- `ecuador-latest.osm.pbf`: ~50-100 MB
- Archivos procesados: ~200-400 MB
- Total: ~500 MB en disco

## ⚠️ Notas

- Los archivos `.osrm` no deben subirse a Git (están en .gitignore)
- Actualizar el mapa: Re-descargar y re-procesar cada 1-3 meses
- Algoritmo MLD: Permite actualizaciones rápidas de tráfico/pesos
