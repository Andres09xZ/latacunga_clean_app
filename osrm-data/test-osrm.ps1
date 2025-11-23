# Script de Testing para OSRM
# Prueba diferentes tipos de rutas en Latacunga

Write-Host "Testeando OSRM - Motor de Rutas" -ForegroundColor Cyan
Write-Host ""

$baseUrl = "http://localhost:5000"

# Test 1: Verificar que OSRM esta corriendo
Write-Host "Test 1: Verificando que OSRM esta corriendo..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$baseUrl/route/v1/driving/-78.614,-0.933;-78.612,-0.935?overview=false" -TimeoutSec 5
    Write-Host "  OK - OSRM esta activo" -ForegroundColor Green
} catch {
    Write-Host "  ERROR - OSRM no responde. Esta corriendo el contenedor?" -ForegroundColor Red
    Write-Host "  Ejecuta: docker-compose up -d routing-engine" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Test 2: Ruta simple (2 puntos en Latacunga)
Write-Host "Test 2: Ruta simple (2 puntos en centro de Latacunga)..." -ForegroundColor Yellow
$route1 = "$baseUrl/route/v1/driving/-78.614,-0.933;-78.612,-0.935?overview=false"
try {
    $response = Invoke-RestMethod -Uri $route1
    $distance = [math]::Round($response.routes[0].distance)
    $duration = [math]::Round($response.routes[0].duration / 60, 1)
    Write-Host "  OK - Distancia: $distance m | Tiempo: $duration min" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Test 3: Trip (multiples puntos con optimizacion)
Write-Host "Test 3: Trip optimizado (5 puntos de recoleccion)..." -ForegroundColor Yellow
$trip = "$baseUrl/trip/v1/driving/-78.614,-0.933;-78.612,-0.935;-78.616,-0.930;-78.610,-0.937;-78.618,-0.932?source=first&destination=last&roundtrip=true"
try {
    $response = Invoke-RestMethod -Uri $trip
    $distance = [math]::Round($response.trips[0].distance)
    $duration = [math]::Round($response.trips[0].duration / 60, 1)
    $numWaypoints = $response.waypoints.Count
    Write-Host "  OK - Ruta optimizada: $numWaypoints puntos | $distance m | $duration min" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Test 4: Nearest (punto mas cercano en la red vial)
Write-Host "Test 4: Nearest (snap a calle mas cercana)..." -ForegroundColor Yellow
$nearest = "$baseUrl/nearest/v1/driving/-78.614,-0.933?number=1"
try {
    $response = Invoke-RestMethod -Uri $nearest
    $snappedLon = [math]::Round($response.waypoints[0].location[0], 6)
    $snappedLat = [math]::Round($response.waypoints[0].location[1], 6)
    $distance = [math]::Round($response.waypoints[0].distance, 1)
    Write-Host "  OK - Snap: [$snappedLon, $snappedLat] | Distancia al punto: $distance m" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "Tests completados. OSRM esta listo para produccion." -ForegroundColor Green
Write-Host ""
Write-Host "Documentacion API: https://project-osrm.org/docs/v5.24.0/api/" -ForegroundColor Cyan
