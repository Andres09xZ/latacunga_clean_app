# Script de Configuracion OSRM para Latacunga Clean
# Ejecutar desde PowerShell en la carpeta osrm-data

Write-Host "Configurando OSRM - Motor de Rutas para Latacunga Clean" -ForegroundColor Cyan
Write-Host ""

# Verificar que estamos en la carpeta correcta
if (-not (Test-Path "README.md")) {
    Write-Host "Error: Ejecuta este script desde la carpeta osrm-data" -ForegroundColor Red
    exit 1
}

# Paso 1: Descargar el mapa de Ecuador
Write-Host "Paso 1/4: Descargando mapa de Ecuador (~80MB)..." -ForegroundColor Yellow
if (Test-Path "ecuador-latest.osm.pbf") {
    Write-Host "  OK - ecuador-latest.osm.pbf ya existe, omitiendo descarga" -ForegroundColor Green
} else {
    Invoke-WebRequest -Uri "http://download.geofabrik.de/south-america/ecuador-latest.osm.pbf" -OutFile "ecuador-latest.osm.pbf"
    Write-Host "  OK - Descarga completada" -ForegroundColor Green
}

Write-Host ""
Write-Host "Paso 2/4: Extrayendo red vial (esto toma 2-5 min)..." -ForegroundColor Yellow
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-extract -p /opt/car.lua /data/ecuador-latest.osm.pbf
if ($LASTEXITCODE -eq 0) {
    Write-Host "  OK - Extraccion completada" -ForegroundColor Green
} else {
    Write-Host "  ERROR en extraccion" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Paso 3/4: Particionando grafo..." -ForegroundColor Yellow
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-partition /data/ecuador-latest.osrm
if ($LASTEXITCODE -eq 0) {
    Write-Host "  OK - Particion completada" -ForegroundColor Green
} else {
    Write-Host "  ERROR en particion" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Paso 4/4: Personalizando pesos..." -ForegroundColor Yellow
docker run -t -v "${PWD}:/data" osrm/osrm-backend osrm-customize /data/ecuador-latest.osrm
if ($LASTEXITCODE -eq 0) {
    Write-Host "  OK - Personalizacion completada" -ForegroundColor Green
} else {
    Write-Host "  ERROR en personalizacion" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "OSRM configurado exitosamente!" -ForegroundColor Green
Write-Host ""
Write-Host "Proximos pasos:" -ForegroundColor Cyan
Write-Host "  1. cd .." -ForegroundColor White
Write-Host "  2. docker-compose up -d routing-engine" -ForegroundColor White
Write-Host "  3. Espera 10 segundos" -ForegroundColor White
Write-Host "  4. Prueba en el navegador:" -ForegroundColor White
Write-Host "     http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.612,-0.935?source=first" -ForegroundColor Gray
Write-Host ""
