# ============================================================
# Script de Inicio Rápido - Docker Compose
# ============================================================
# Este script levanta todos los microservicios del sistema

Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║      🚀 Sistema de Gestión de Limpieza - Latacunga         ║" -ForegroundColor Cyan
Write-Host "║           Iniciando Todos los Microservicios                ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Verificar Docker
Write-Host "🔍 Verificando Docker..." -ForegroundColor Yellow
if (!(Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Error: Docker no está instalado" -ForegroundColor Red
    exit 1
}

# Detener contenedores previos
Write-Host "🛑 Deteniendo contenedores previos..." -ForegroundColor Yellow
docker-compose down 2>$null

# Construir y levantar servicios
Write-Host "🏗️  Construyendo imágenes Docker..." -ForegroundColor Yellow
docker-compose build

Write-Host "🚀 Levantando todos los servicios..." -ForegroundColor Green
docker-compose up -d

# Esperar a que los servicios inicien
Write-Host ""
Write-Host "⏳ Esperando inicialización de servicios (60 segundos)..." -ForegroundColor Yellow
for ($i = 1; $i -le 60; $i++) {
    Write-Progress -Activity "Inicializando servicios" -Status "$i/60 segundos" -PercentComplete ($i/60*100)
    Start-Sleep -Seconds 1
}
Write-Progress -Activity "Inicializando servicios" -Completed

# Verificar servicios
Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║              📊 Estado de los Servicios                     ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

$services = @(
    @{Name="RabbitMQ Management"; Port=15672; Path="/"; Type="UI"},
    @{Name="RabbitMQ AMQP"; Port=5672; Path=""; Type="Port"},
    @{Name="OSRM Routing"; Port=5000; Path="/trip/v1/driving/-78.614,-0.933;-78.612,-0.935?source=first"; Type="API"},
    @{Name="Auth Service"; Port=8080; Path="/health"; Type="API"},
    @{Name="Fleet Service"; Port=8081; Path="/health"; Type="API"},
    @{Name="Incident Service"; Port=8082; Path="/health"; Type="API"},
    @{Name="Scheduler Service"; Port=8083; Path="/health"; Type="API"},
    @{Name="Operations Service"; Port=8085; Path="/health"; Type="API"}
)

foreach ($service in $services) {
    $serviceName = $service.Name.PadRight(25)
    if ($service.Type -eq "Port") {
        Write-Host "  $serviceName : " -NoNewline
        Write-Host "localhost:$($service.Port)" -ForegroundColor Green
    } else {
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:$($service.Port)$($service.Path)" -Method GET -TimeoutSec 5 -UseBasicParsing 2>$null
            Write-Host "  $serviceName : " -NoNewline
            Write-Host "✅ Online (localhost:$($service.Port))" -ForegroundColor Green
        } catch {
            Write-Host "  $serviceName : " -NoNewline
            Write-Host "❌ No disponible" -ForegroundColor Red
        }
    }
}

# Mostrar logs
Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║                    📝 Información                            ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "  🌐 RabbitMQ Management UI: " -NoNewline
Write-Host "http://localhost:15672" -ForegroundColor Yellow
Write-Host "     Usuario: tesis | Password: tesis" -ForegroundColor Gray
Write-Host ""
Write-Host "  📋 Ver logs de todos los servicios:" -ForegroundColor White
Write-Host "     docker-compose logs -f" -ForegroundColor Gray
Write-Host ""
Write-Host "  📋 Ver logs de un servicio específico:" -ForegroundColor White
Write-Host "     docker-compose logs -f [servicio]" -ForegroundColor Gray
Write-Host "     Ejemplo: docker-compose logs -f auth-service" -ForegroundColor Gray
Write-Host ""
Write-Host "  🛑 Detener todos los servicios:" -ForegroundColor White
Write-Host "     docker-compose down" -ForegroundColor Gray
Write-Host ""
Write-Host "  🧪 Ejecutar pruebas E2E:" -ForegroundColor White
Write-Host "     .\run_e2e_test.ps1" -ForegroundColor Gray
Write-Host ""
Write-Host "✅ Sistema listo para usar!" -ForegroundColor Green
Write-Host ""
