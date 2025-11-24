# ============================================================
# Script de Limpieza - Datos de Prueba E2E
# ============================================================

param(
    [switch]$ConfirmAll = $false
)

Write-Host "`n╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Yellow
Write-Host "║     LIMPIEZA DE DATOS DE PRUEBA E2E                          ║" -ForegroundColor Yellow
Write-Host "╚════════════════════════════════════════════════════════════════╝`n" -ForegroundColor Yellow

# Database credentials
$DB_HOST = "ep-gentle-pond-adcmrdsv-pooler.c-2.us-east-1.aws.neon.tech"
$DB_NAME = "neondb"
$DB_USER = "neondb_owner"
$DB_PASSWORD = "npg_jnw3bVupEP5i"
$DB_URL = "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}/${DB_NAME}?sslmode=require"

Write-Host "⚠️  ADVERTENCIA: Esta operación eliminará:" -ForegroundColor Red
Write-Host "   • Todos los operadores de prueba (username: test_driver_*)" -ForegroundColor Yellow
Write-Host "   • Turnos asociados a operadores de prueba" -ForegroundColor Yellow
Write-Host "   • Incidentes creados durante las pruebas E2E" -ForegroundColor Yellow
Write-Host "   • Órdenes de trabajo generadas en pruebas`n" -ForegroundColor Yellow

if (-not $ConfirmAll) {
    $confirmation = Read-Host "¿Estás seguro? Escribe 'SI' para continuar"
    if ($confirmation -ne "SI") {
        Write-Host "`n❌ Operación cancelada`n" -ForegroundColor Red
        exit 0
    }
}

Write-Host "`n🧹 Iniciando limpieza...`n" -ForegroundColor Cyan

# Verificar si psql está disponible
try {
    $null = psql --version 2>&1
    Write-Host "✓ PostgreSQL client (psql) encontrado`n" -ForegroundColor Green
}
catch {
    Write-Host "✗ psql no está instalado o no está en PATH" -ForegroundColor Red
    Write-Host "→ Instala PostgreSQL client desde: https://www.postgresql.org/download/`n" -ForegroundColor Yellow
    exit 1
}

# SQL para limpiar datos de prueba
$cleanupSQL = @"
-- Limpieza de datos de prueba E2E

BEGIN;

-- 1. Eliminar órdenes de trabajo de prueba
DELETE FROM work_orders 
WHERE driver_id IN (
    SELECT id FROM users WHERE username LIKE 'test_driver_%'
);

-- 2. Eliminar paradas de órdenes de trabajo
DELETE FROM work_order_stops 
WHERE work_order_id NOT IN (
    SELECT id FROM work_orders
);

-- 3. Eliminar turnos de prueba
DELETE FROM shifts 
WHERE driver_id IN (
    SELECT id FROM users WHERE username LIKE 'test_driver_%'
);

-- 4. Eliminar perfiles de operador de prueba
DELETE FROM operator_profiles 
WHERE user_id IN (
    SELECT id FROM users WHERE username LIKE 'test_driver_%'
);

-- 5. Eliminar usuarios de prueba
DELETE FROM users 
WHERE username LIKE 'test_driver_%' 
   OR email LIKE 'test_%@test.com';

-- 6. Eliminar incidentes de prueba (opcional)
DELETE FROM incidents 
WHERE description LIKE '%E2E Test%';

-- 7. Limpiar eventos de outbox antiguos (opcional)
DELETE FROM outbox_events 
WHERE created_at < NOW() - INTERVAL '7 days' 
  AND status = 'published';

COMMIT;

-- Mostrar resumen
SELECT 
    'Users eliminados' as tabla, 
    COUNT(*) as registros 
FROM users 
WHERE username LIKE 'test_driver_%'
UNION ALL
SELECT 
    'Incidents eliminados', 
    COUNT(*) 
FROM incidents 
WHERE description LIKE '%E2E Test%';
"@

# Guardar SQL temporalmente
$tempSQLFile = [System.IO.Path]::GetTempFileName() + ".sql"
$cleanupSQL | Out-File -FilePath $tempSQLFile -Encoding UTF8

Write-Host "Ejecutando script de limpieza en la base de datos..." -ForegroundColor Cyan

# Ejecutar limpieza
try {
    $env:PGPASSWORD = $DB_PASSWORD
    $output = psql $DB_URL -f $tempSQLFile 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n✓ Limpieza completada exitosamente`n" -ForegroundColor Green
        Write-Host $output -ForegroundColor Gray
    }
    else {
        Write-Host "`n✗ Error durante la limpieza" -ForegroundColor Red
        Write-Host $output -ForegroundColor Red
    }
}
catch {
    Write-Host "`n✗ Error ejecutando psql: $_" -ForegroundColor Red
}
finally {
    # Limpiar archivo temporal
    Remove-Item $tempSQLFile -ErrorAction SilentlyContinue
    Remove-Item env:PGPASSWORD -ErrorAction SilentlyContinue
}

Write-Host "`n═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "                    LIMPIEZA FINALIZADA" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════════`n" -ForegroundColor Cyan

Write-Host "Notas importantes:" -ForegroundColor Yellow
Write-Host "  • Los datos de producción NO fueron afectados" -ForegroundColor Gray
Write-Host "  • Solo se eliminaron registros con 'test_driver_' o 'E2E Test'" -ForegroundColor Gray
Write-Host "  • Puedes ejecutar la prueba E2E nuevamente sin problemas`n" -ForegroundColor Gray

Write-Host "Presiona ENTER para salir..." -ForegroundColor Cyan
$null = Read-Host
