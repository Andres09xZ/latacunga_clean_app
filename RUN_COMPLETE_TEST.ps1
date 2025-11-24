# ============================================================
# GUÍA DE EJECUCIÓN - PRUEBA E2E
# ============================================================
# Ejecuta estos comandos en PowerShell para probar tu sistema

# PASO 1: Levantar todos los servicios
Write-Host "`n🚀 Levantando servicios..." -ForegroundColor Cyan
docker-compose up -d

# PASO 2: Esperar inicialización
Write-Host "`n⏳ Esperando 30 segundos para inicialización..." -ForegroundColor Yellow
Start-Sleep -Seconds 30

# PASO 3: Verificar servicios
Write-Host "`n✓ Verificando servicios..." -ForegroundColor Green

$services = @(
    "http://localhost:8080/health",  # Auth
    "http://localhost:8081/health",  # Fleet
    "http://localhost:8082/health",  # Incident
    "http://localhost:8083/health",  # Scheduler
    "http://localhost:8085/health"   # Operations
)

foreach ($url in $services) {
    try {
        $response = Invoke-WebRequest -Uri $url -Method Get -TimeoutSec 5
        Write-Host "  ✓ $url : OK" -ForegroundColor Green
    }
    catch {
        Write-Host "  ✗ $url : FAIL" -ForegroundColor Red
    }
}

# PASO 4: Ejecutar prueba E2E
Write-Host "`n🧪 Ejecutando prueba E2E..." -ForegroundColor Cyan
go run e2e_test.go

# PASO 5: Resultado
if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ ¡PRUEBA EXITOSA!" -ForegroundColor Green
    Write-Host "El sistema está funcionando correctamente`n" -ForegroundColor Green
} else {
    Write-Host "`n❌ PRUEBA FALLIDA" -ForegroundColor Red
    Write-Host "Revisa los logs para más detalles`n" -ForegroundColor Yellow
}

# OPCIONAL: Limpiar datos de prueba
# Write-Host "`n🧹 ¿Deseas limpiar los datos de prueba? (S/N)" -ForegroundColor Yellow
# $response = Read-Host
# if ($response -eq "S" -or $response -eq "s") {
#     .\cleanup_e2e_data.ps1
# }
