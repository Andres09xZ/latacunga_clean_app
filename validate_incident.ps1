# Script para validar un incidente pendiente

param(
    [Parameter(Mandatory=$false)]
    [string]$IncidentID,
    
    [Parameter(Mandatory=$false)]
    [string]$Status = "incidente_valido",
    
    [Parameter(Mandatory=$false)]
    [string]$Notes = "Incidente verificado automáticamente"
)

$ValidationServiceURL = "http://localhost:8084"

# Si no se proporciona ID, obtener el primero pendiente
if (-not $IncidentID) {
    Write-Host "📋 Obteniendo incidentes pendientes..." -ForegroundColor Cyan
    $pending = Invoke-RestMethod -Uri "$ValidationServiceURL/api/v1/incidents/pending" -Method GET
    
    if ($pending.count -eq 0) {
        Write-Host "❌ No hay incidentes pendientes" -ForegroundColor Red
        exit 1
    }
    
    $IncidentID = $pending.data[0].incidente_id
    Write-Host "✅ Incidente encontrado: $IncidentID" -ForegroundColor Green
    Write-Host "   Tipo: $($pending.data[0].tipo)" -ForegroundColor Yellow
    Write-Host "   Latitud: $($pending.data[0].latitud)" -ForegroundColor Yellow
    Write-Host "   Longitud: $($pending.data[0].longitud)" -ForegroundColor Yellow
}

# Validar el incidente
Write-Host "`n🔍 Validando incidente $IncidentID..." -ForegroundColor Cyan

$body = @{
    status = $Status
    notes = $Notes
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$ValidationServiceURL/api/v1/incidents/$IncidentID/validate" `
                                  -Method POST `
                                  -Body $body `
                                  -ContentType "application/json"
    
    Write-Host "✅ Incidente validado exitosamente!" -ForegroundColor Green
    Write-Host "   Status: $($response.status)" -ForegroundColor Yellow
    Write-Host "   Validated At: $($response.validated_at)" -ForegroundColor Yellow
    
} catch {
    Write-Host "❌ Error al validar incidente: $_" -ForegroundColor Red
    exit 1
}
