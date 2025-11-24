# Script para purgar mensajes viejos de la cola de routing
# Uso: .\purge_routing_queue.ps1

Write-Host "🗑️  Purgando cola de routing-service..." -ForegroundColor Yellow

# Configuración de RabbitMQ
$rabbitHost = "localhost"
$rabbitPort = "15672"
$rabbitUser = "guest"
$rabbitPass = "guest"
$queueName = "q.routing.plan-requests"
$vhost = "%2F"  # URL encoded "/"

# Crear credenciales
$base64AuthInfo = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes(("{0}:{1}" -f $rabbitUser, $rabbitPass)))

# URL de la API de RabbitMQ
$url = "http://${rabbitHost}:${rabbitPort}/api/queues/${vhost}/${queueName}/contents"

try {
    Write-Host "📡 Conectando a RabbitMQ Management API..."
    
    # Purgar la cola
    $response = Invoke-RestMethod -Uri $url `
        -Method Delete `
        -Headers @{Authorization=("Basic {0}" -f $base64AuthInfo)} `
        -ContentType "application/json"
    
    Write-Host "✅ Cola '$queueName' purgada exitosamente" -ForegroundColor Green
    
    # Verificar estado de la cola
    $statusUrl = "http://${rabbitHost}:${rabbitPort}/api/queues/${vhost}/${queueName}"
    $queueInfo = Invoke-RestMethod -Uri $statusUrl `
        -Method Get `
        -Headers @{Authorization=("Basic {0}" -f $base64AuthInfo)}
    
    Write-Host "📊 Estado actual de la cola:" -ForegroundColor Cyan
    Write-Host "   - Mensajes en cola: $($queueInfo.messages)" -ForegroundColor White
    Write-Host "   - Mensajes listos: $($queueInfo.messages_ready)" -ForegroundColor White
    Write-Host "   - Consumidores: $($queueInfo.consumers)" -ForegroundColor White
    
} catch {
    Write-Host "❌ Error al purgar la cola: $_" -ForegroundColor Red
    Write-Host "   Asegúrate de que:" -ForegroundColor Yellow
    Write-Host "   1. RabbitMQ esté ejecutándose" -ForegroundColor Yellow
    Write-Host "   2. El plugin de management esté habilitado: rabbitmq-plugins enable rabbitmq_management" -ForegroundColor Yellow
    Write-Host "   3. Las credenciales sean correctas (usuario: $rabbitUser)" -ForegroundColor Yellow
    exit 1
}

Write-Host "`n✨ Proceso completado. Ahora puedes reiniciar el routing-service" -ForegroundColor Green
