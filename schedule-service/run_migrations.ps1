# Script para ejecutar migraciones en Neon PostgreSQL
# Uso: .\run_migrations.ps1

Write-Host "=== Ejecutor de Migraciones - Schedule Service ===" -ForegroundColor Cyan

# Leer DB_URL del .env
$envPath = ".env"
if (Test-Path $envPath) {
    Get-Content $envPath | ForEach-Object {
        if ($_ -match '^DB_URL=(.+)$') {
            $DB_URL = $matches[1]
        }
    }
}

if (-not $DB_URL) {
    Write-Host "Error: DB_URL no encontrado en .env" -ForegroundColor Red
    Write-Host "Formato esperado: postgres://user:password@host:port/database?sslmode=require" -ForegroundColor Yellow
    exit 1
}

Write-Host "Conexión: $($DB_URL -replace ':[^:@]+@', ':****@')" -ForegroundColor Green

# Parsear connection string (soporta postgres:// y postgresql://)
if ($DB_URL -match 'postgres(?:ql)?://([^:]+):([^@]+)@([^:/]+)(?::(\d+))?/([^\?]+)') {
    $DB_USER = $matches[1]
    $DB_PASSWORD = $matches[2]
    $DB_HOST = $matches[3]
    $DB_PORT = if ($matches[4]) { $matches[4] } else { "5432" }
    $DB_NAME = $matches[5]
} else {
    Write-Host "Error: Formato de DB_URL inválido" -ForegroundColor Red
    Write-Host "DB_URL recibido: $DB_URL" -ForegroundColor Yellow
    Write-Host "Formato esperado: postgres://user:pass@host:port/db" -ForegroundColor Yellow
    exit 1
}

# Lista de migraciones
$migrations = @(
    @{File = "migrations/007_add_zone_scoring.sql"; Name = "007 - Threshold y current_score"},
    @{File = "migrations/009_add_zone_status.sql"; Name = "009 - Status y last_updated"},
    @{File = "migrations/010_add_route_type_schedule_time.sql"; Name = "010 - Route_type y schedule_time"}
)

Write-Host "`nEjecutando migraciones..." -ForegroundColor Cyan

foreach ($migration in $migrations) {
    if (-not (Test-Path $migration.File)) {
        Write-Host "  ⚠️  $($migration.Name): Archivo no encontrado - SKIP" -ForegroundColor Yellow
        continue
    }
    
    Write-Host "  ▶ $($migration.Name)..." -NoNewline
    
    $env:PGPASSWORD = $DB_PASSWORD
    $output = & psql -h $DB_HOST -U $DB_USER -d $DB_NAME -p $DB_PORT -f $migration.File 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host " ✅ OK" -ForegroundColor Green
    } else {
        Write-Host " ❌ ERROR" -ForegroundColor Red
        Write-Host $output -ForegroundColor Red
    }
}

Write-Host "`n=== Migraciones completadas ===" -ForegroundColor Cyan
Write-Host "Ahora puedes iniciar el servicio con: go run cmd/server/main.go" -ForegroundColor Green
