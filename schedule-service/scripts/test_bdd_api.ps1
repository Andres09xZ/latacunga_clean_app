# Script de prueba BDD para API REST Scheduler Service
# Prerequisito: servicio corriendo en http://localhost:8083
# Prerequisito: migración 009 ejecutada (status column)
# Prerequisito: zona URBANO_CENTRAL con ID 1 y threshold 50

# Setup
$BASE_URL = "http://localhost:8083"
$JWT_SECRET = "mysecret"

Write-Host "`n=== Generando JWT ===" -ForegroundColor Cyan
$TOKEN = (go run scripts/gen_jwt.go -sub admin1 -role admin -email admin@municipio.ec -secret $JWT_SECRET).Trim()
Write-Host "Token: $TOKEN"

$HEADERS = @{ Authorization = "Bearer $TOKEN" }

Write-Host "`n=== ESCENARIO 1: GET /api/v1/zones (Listar zonas con geometría) ===" -ForegroundColor Yellow
try {
    $zonesResponse = Invoke-RestMethod -Method GET -Uri "$BASE_URL/api/v1/zones" -Headers $HEADERS
    Write-Host "✅ Status: 200 OK"
    Write-Host "Zonas encontradas: $($zonesResponse.Count)"
    if ($zonesResponse.Count -gt 0) {
        $zone1 = $zonesResponse[0]
        Write-Host "  - ID: $($zone1.id), Name: $($zone1.name), Type: $($zone1.type)"
        if ($zone1.geometry) {
            Write-Host "  - Geometría incluida: ✅"
        } else {
            Write-Host "  - ⚠️  Geometría no incluida (verificar migración/datos)"
        }
    }
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
}

Write-Host "`n=== ESCENARIO 2: GET /api/v1/zones/1/metrics (Ver métricas zona) ===" -ForegroundColor Yellow
try {
    $metricsResponse = Invoke-RestMethod -Method GET -Uri "$BASE_URL/api/v1/zones/1/metrics" -Headers $HEADERS
    Write-Host "✅ Status: 200 OK"
    Write-Host "Métricas:"
    Write-Host "  - zone_id: $($metricsResponse.zone_id)"
    Write-Host "  - name: $($metricsResponse.name)"
    Write-Host "  - current_score: $($metricsResponse.current_score)"
    Write-Host "  - threshold: $($metricsResponse.threshold)"
    Write-Host "  - status: $($metricsResponse.status)"
    Write-Host "  - last_updated: $($metricsResponse.last_updated)"
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
}

Write-Host "`n=== ESCENARIO 3: POST /api/v1/debug/simulate (Simular incidente con zone_id + points) ===" -ForegroundColor Yellow
try {
    $simulateBody = @{
        zone_id = 1
        type = "SENSOR_IOT_LLENO"
        points = 5
    } | ConvertTo-Json
    $simulateResponse = Invoke-RestMethod -Method POST -Uri "$BASE_URL/api/v1/debug/simulate" -Headers $HEADERS -ContentType "application/json" -Body $simulateBody
    Write-Host "✅ Status: 202 Accepted"
    Write-Host "Respuesta: $($simulateResponse | ConvertTo-Json -Compress)"
    
    # Verificar métricas nuevamente
    Start-Sleep -Seconds 1
    $metricsAfter = Invoke-RestMethod -Method GET -Uri "$BASE_URL/api/v1/zones/1/metrics" -Headers $HEADERS
    Write-Host "Puntaje después: $($metricsAfter.current_score) (esperado: incremento de 5)"
    Write-Host "Status después: $($metricsAfter.status)"
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
}

Write-Host "`n=== ESCENARIO 4: POST /api/v1/config/weights (Configurar pesos) ===" -ForegroundColor Yellow
try {
    $weightsBody = @{
        SENSOR_IOT_LLENO = 10
        REPORTE_CIUDADANO = 2
    } | ConvertTo-Json
    $weightsResponse = Invoke-RestMethod -Method POST -Uri "$BASE_URL/api/v1/config/weights" -Headers $HEADERS -ContentType "application/json" -Body $weightsBody
    Write-Host "✅ Status: 200 OK"
    Write-Host "Mensaje: $($weightsResponse.message)"
    if ($weightsResponse.message -eq "Pesos actualizados correctamente") {
        Write-Host "  ✅ Mensaje coincide con especificación Gherkin"
    } else {
        Write-Host "  ⚠️  Mensaje difiere del esperado"
    }
    Write-Host "Pesos actuales: $($weightsResponse.weights | ConvertTo-Json -Compress)"
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
}

Write-Host "`n=== ESCENARIO 5: POST /api/v1/zones/1/trigger (Botón de pánico / trigger manual) ===" -ForegroundColor Yellow
try {
    $triggerBody = @{
        reason = "Emergencia Sanitaria reportada por Alcaldía"
        operator_override = "CHOFER_01"
    } | ConvertTo-Json
    $triggerResponse = Invoke-RestMethod -Method POST -Uri "$BASE_URL/api/v1/zones/1/trigger" -Headers $HEADERS -ContentType "application/json" -Body $triggerBody
    Write-Host "✅ Status: 201 Created"
    Write-Host "Work Order ID: $($triggerResponse.work_order_id)"
    Write-Host "Zone ID: $($triggerResponse.zone_id)"
    Write-Host "Status: $($triggerResponse.status)"
    if ($triggerResponse.status -eq "EN_PROGRESO") {
        Write-Host "  ✅ Status cambió a EN_PROGRESO"
    } else {
        Write-Host "  ⚠️  Status no es EN_PROGRESO: $($triggerResponse.status)"
    }
    if ($triggerResponse.reason) {
        Write-Host "Reason: $($triggerResponse.reason)"
    }
    if ($triggerResponse.operator_override) {
        Write-Host "Operator Override: $($triggerResponse.operator_override)"
    }
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
}

Write-Host "`n=== Verificación final de estado zona 1 ===" -ForegroundColor Cyan
try {
    $finalMetrics = Invoke-RestMethod -Method GET -Uri "$BASE_URL/api/v1/zones/1/metrics" -Headers $HEADERS
    Write-Host "Status final: $($finalMetrics.status)"
    Write-Host "Current score final: $($finalMetrics.current_score)"
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
}

Write-Host "`n=== PRUEBAS COMPLETADAS ===" -ForegroundColor Green
Write-Host "Verifica que:"
Write-Host "1. GET /zones retornó geometría MultiPolygon"
Write-Host "2. GET /zones/1/metrics retornó status ACUMULANDO/LISTO_PARA_RECOLECCION/EN_PROGRESO"
Write-Host "3. POST /debug/simulate retornó 202 Accepted"
Write-Host "4. POST /config/weights retornó 'Pesos actualizados correctamente'"
Write-Host "5. POST /zones/1/trigger retornó 201 Created con work_order_id y status EN_PROGRESO"
