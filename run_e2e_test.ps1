# ============================================================
# Script de Prueba E2E - Sistema EPAGAL Latacunga
# ============================================================

Write-Host "`n╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║     PRUEBA E2E - SISTEMA EPAGAL LATACUNGA                    ║" -ForegroundColor Cyan
Write-Host "║     Pre-Verificación y Ejecución                              ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝`n" -ForegroundColor Cyan

# Colores
$SUCCESS = "Green"
$ERROR = "Red"
$INFO = "Yellow"
$STEP = "Cyan"

# Configuración de servicios
$services = @{
    "Fleet Service"      = "http://localhost:8081/health"
    "Scheduler Service"  = "http://localhost:8083/health"
    "Operations Service" = "http://localhost:8085/health"
}

# Función para verificar servicio
function Test-Service {
    param (
        [string]$Name,
        [string]$Url
    )
    
    try {
        $response = Invoke-WebRequest -Uri $Url -Method Get -TimeoutSec 5 -ErrorAction Stop -UseBasicParsing
        if ($response.StatusCode -eq 200) {
            Write-Host "✓ $Name : " -NoNewline -ForegroundColor $SUCCESS
            Write-Host "OK" -ForegroundColor $SUCCESS
            return $true
        }
    }
    catch {
        Write-Host "✗ $Name : " -NoNewline -ForegroundColor $ERROR
        Write-Host "NOT AVAILABLE" -ForegroundColor $ERROR
        return $false
    }
}

# ============================================================
# PASO 1: Verificar servicios
# ============================================================
Write-Host "═══ PASO 1: Verificando servicios ═══`n" -ForegroundColor $STEP

$allHealthy = $true
foreach ($service in $services.GetEnumerator()) {
    $result = Test-Service -Name $service.Key -Url $service.Value
    if (-not $result) {
        $allHealthy = $false
    }
    Start-Sleep -Milliseconds 200
}

if (-not $allHealthy) {
    Write-Host "`n❌ ERROR: Algunos servicios no están disponibles" -ForegroundColor $ERROR
    Write-Host "→ Ejecuta: docker-compose up -d" -ForegroundColor $INFO
    Write-Host "→ Espera 30 segundos y vuelve a intentar`n" -ForegroundColor $INFO
    exit 1
}

Write-Host "`n✓ Todos los servicios están disponibles`n" -ForegroundColor $SUCCESS

# ============================================================
# PASO 2: Verificar RabbitMQ
# ============================================================
Write-Host "═══ PASO 2: Verificando RabbitMQ ═══`n" -ForegroundColor $STEP

try {
    $rabbitResponse = Invoke-WebRequest -Uri "http://localhost:15672" -Method Get -TimeoutSec 5 -ErrorAction Stop -UseBasicParsing
    Write-Host "✓ RabbitMQ Management UI: OK" -ForegroundColor $SUCCESS
    Write-Host "→ URL: http://localhost:15672 (tesis/tesis)`n" -ForegroundColor $INFO
}
catch {
    Write-Host "✗ RabbitMQ Management UI: NOT AVAILABLE" -ForegroundColor $ERROR
    Write-Host "→ Verifica que RabbitMQ esté corriendo`n" -ForegroundColor $INFO
    exit 1
}

# ============================================================
# PASO 3: Verificar Go
# ============================================================
Write-Host "═══ PASO 3: Verificando Go ═══`n" -ForegroundColor $STEP

try {
    $goVersion = go version 2>&1
    Write-Host "✓ Go instalado: $goVersion`n" -ForegroundColor $SUCCESS
}
catch {
    Write-Host "✗ Go no está instalado o no está en PATH" -ForegroundColor $ERROR
    Write-Host "→ Instala Go desde: https://golang.org/dl/`n" -ForegroundColor $INFO
    exit 1
}

# ============================================================
# PASO 4: Verificar archivo de prueba
# ============================================================
Write-Host "═══ PASO 4: Verificando archivo e2e_test.go ═══`n" -ForegroundColor $STEP

if (Test-Path "e2e_test.go") {
    Write-Host "✓ Archivo e2e_test.go encontrado`n" -ForegroundColor $SUCCESS
}
else {
    Write-Host "✗ Archivo e2e_test.go no encontrado" -ForegroundColor $ERROR
    Write-Host "→ Asegúrate de estar en el directorio correcto`n" -ForegroundColor $INFO
    exit 1
}

# ============================================================
# PASO 5: Mostrar información del sistema
# ============================================================
Write-Host "═══ INFORMACIÓN DEL SISTEMA ═══`n" -ForegroundColor $STEP

Write-Host "Servicios probados:" -ForegroundColor $INFO
Write-Host "  • Auth Service       → http://localhost:8080" -ForegroundColor Gray
Write-Host "  • Fleet Service      → http://localhost:8081" -ForegroundColor Gray
Write-Host "  • Incident Service   → http://localhost:8082" -ForegroundColor Gray
Write-Host "  • Scheduler Service  → http://localhost:8083" -ForegroundColor Gray
Write-Host "  • Operations Service → http://localhost:8085" -ForegroundColor Gray
Write-Host ""
Write-Host "RabbitMQ:" -ForegroundColor $INFO
Write-Host "  • Management UI → http://localhost:15672" -ForegroundColor Gray
Write-Host "  • AMQP Port     → localhost:5672" -ForegroundColor Gray
Write-Host ""

# ============================================================
# PASO 6: Ejecutar prueba E2E
# ============================================================
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor $STEP
Write-Host "       ¿Listo para ejecutar la prueba E2E?" -ForegroundColor $STEP
Write-Host "═══════════════════════════════════════════════════════════════`n" -ForegroundColor $STEP

Write-Host "La prueba tomará aproximadamente 30-40 segundos" -ForegroundColor $INFO
Write-Host "Presiona ENTER para continuar o Ctrl+C para cancelar...`n" -ForegroundColor $INFO
$null = Read-Host

Write-Host "`n🚀 Ejecutando prueba E2E...`n" -ForegroundColor $SUCCESS

# Ejecutar el script de prueba
go run e2e_test.go

# Verificar el código de salida
if ($LASTEXITCODE -eq 0) {
    Write-Host "`n╔════════════════════════════════════════════════════════════════╗" -ForegroundColor $SUCCESS
    Write-Host "║                   ✅ PRUEBA EXITOSA ✅                        ║" -ForegroundColor $SUCCESS
    Write-Host "╚════════════════════════════════════════════════════════════════╝`n" -ForegroundColor $SUCCESS
    
    Write-Host "El sistema EPAGAL está funcionando correctamente" -ForegroundColor $SUCCESS
    Write-Host "Todos los microservicios se comunican exitosamente`n" -ForegroundColor $SUCCESS
}
else {
    Write-Host "`n╔════════════════════════════════════════════════════════════════╗" -ForegroundColor $ERROR
    Write-Host "║                   ❌ PRUEBA FALLIDA ❌                        ║" -ForegroundColor $ERROR
    Write-Host "╚════════════════════════════════════════════════════════════════╝`n" -ForegroundColor $ERROR
    
    Write-Host "Revisa los logs anteriores para identificar el problema`n" -ForegroundColor $INFO
    Write-Host "Tips para debugging:" -ForegroundColor $INFO
    Write-Host "  1. Revisa logs de servicios: docker-compose logs <servicio>" -ForegroundColor Gray
    Write-Host "  2. Verifica RabbitMQ: http://localhost:15672" -ForegroundColor Gray
    Write-Host "  3. Consulta la base de datos directamente" -ForegroundColor Gray
    Write-Host "  4. Revisa el archivo E2E_README.md para más detalles`n" -ForegroundColor Gray
}

Write-Host "Presiona ENTER para salir..." -ForegroundColor $INFO
$null = Read-Host
