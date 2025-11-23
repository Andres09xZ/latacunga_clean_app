# Script para cargar las 5 zonas macro de Latacunga
# Ejecutar solo si las zonas se eliminaron accidentalmente

Write-Host "🔧 Preparando para cargar las 5 zonas macro..." -ForegroundColor Cyan

# 1. Arreglar el schema
Write-Host "`n📋 Paso 1: Verificando schema de la base de datos..." -ForegroundColor Yellow
go run scripts/fix_schema.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Error en fix_schema" -ForegroundColor Red
    exit 1
}

# 2. Cargar las zonas
Write-Host "`n📋 Paso 2: Cargando las 5 zonas macro desde zonas_macro.geojson..." -ForegroundColor Yellow
go run scripts/load_macro_zones.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Error cargando zonas" -ForegroundColor Red
    exit 1
}

# 3. Inicializar métricas
Write-Host "`n📋 Paso 3: Inicializando zone_metrics..." -ForegroundColor Yellow
go run scripts/init_zone_metrics.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Error inicializando métricas" -ForegroundColor Red
    exit 1
}

# 4. Verificar
Write-Host "`n📋 Paso 4: Verificando zonas cargadas..." -ForegroundColor Yellow
go run scripts/check_db.go

Write-Host "`n✅ ¡Proceso completado! Las 5 zonas macro están cargadas." -ForegroundColor Green
Write-Host "Las zonas son persistentes en Neon PostgreSQL y no se eliminarán." -ForegroundColor Green
