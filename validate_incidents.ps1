# Script para prueba E2E completa con validación automática de incidentes
# Este script crea 15 incidentes y los valida automáticamente

param(
    [int]$IncidentCount = 15,
    [string]$ValidationServiceURL = "http://localhost:8084"
)

Write-Host "🧹 VALIDADOR AUTOMÁTICO DE INCIDENTES E2E" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# Esperar 5 segundos para que los incidentes se creen
Write-Host "⏱️  Esperando 5 segundos para que incidentes sean creados..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Obtener lista de incidentes pendientes
Write-Host "📋 Consultando incidentes pendientes..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$ValidationServiceURL/api/v1/incidents/pending" -Method GET
    
    if ($response.success -and $response.count -gt 0) {
        Write-Host "✅ Encontrados $($response.count) incidentes pendientes`n" -ForegroundColor Green
        
        $validatedCount = 0
        foreach ($incident in $response.data) {
            $incidentId = $incident.incidente_id
            $tipo = $incident.tipo
            
            Write-Host "📝 Validando incidente $incidentId (Tipo: $tipo)..." -ForegroundColor Cyan
            
            # Validar incidente
            $validateBody = @{
                status = "incidente_valido"
                notes = "Validado automáticamente por script E2E"
            } | ConvertTo-Json
            
            try {
                $validateResponse = Invoke-RestMethod `
                    -Uri "$ValidationServiceURL/api/v1/incidents/$incidentId/validate" `
                    -Method POST `
                    -Body $validateBody `
                    -ContentType "application/json"
                
                if ($validateResponse.success) {
                    $validatedCount++
                    Write-Host "   ✅ Validado correctamente" -ForegroundColor Green
                } else {
                    Write-Host "   ❌ Error: $($validateResponse.error)" -ForegroundColor Red
                }
            } catch {
                Write-Host "   ❌ Error al validar: $($_.Exception.Message)" -ForegroundColor Red
            }
            
            Start-Sleep -Milliseconds 200
        }
        
        Write-Host "`n✅ RESUMEN: $validatedCount/$($response.count) incidentes validados" -ForegroundColor Green
        Write-Host "📡 Los eventos fueron publicados a RabbitMQ (incidents.validated.v1)" -ForegroundColor Cyan
        Write-Host "🔄 El scheduler debería procesar estos eventos automáticamente`n" -ForegroundColor Cyan
        
    } else {
        Write-Host "⚠️  No hay incidentes pendientes de validación" -ForegroundColor Yellow
    }
} catch {
    Write-Host "❌ Error al consultar incidentes: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
