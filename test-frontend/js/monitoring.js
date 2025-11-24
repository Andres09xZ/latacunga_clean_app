// Monitoreo de Zonas y Score

async function loadZoneMetrics() {
    const container = document.getElementById('zoneMetrics');
    container.innerHTML = '<p>Cargando métricas de zonas...</p>';
    
    try {
        // Obtener lista de zonas
        const zonesResponse = await fetch(`${API_CONFIG.SCHEDULE_SERVICE}/api/v1/zones`);
        
        if (!zonesResponse.ok) {
            throw new Error('No se pudo cargar las zonas');
        }
        
        const zones = await zonesResponse.json();
        
        if (!zones || zones.length === 0) {
            container.innerHTML = '<p class="info">⚠️ No hay zonas registradas</p>';
            return;
        }
        
        container.innerHTML = '';
        
        // Cargar métricas de cada zona
        for (const zone of zones) {
            try {
                const metricsResponse = await fetch(`${API_CONFIG.SCHEDULE_SERVICE}/api/v1/zones/${zone.id}/metrics`);
                
                let metrics = { current_score: 0, threshold: 50 };
                if (metricsResponse.ok) {
                    metrics = await metricsResponse.json();
                }
                
                const triggered = metrics.current_score >= metrics.threshold;
                const progress = (metrics.current_score / metrics.threshold) * 100;
                
                const card = document.createElement('div');
                card.className = `zone-card ${triggered ? 'triggered' : ''}`;
                card.innerHTML = `
                    <h3>📍 ${zone.zone_name}</h3>
                    <div class="score">${metrics.current_score} / ${metrics.threshold}</div>
                    <div class="threshold">
                        <div style="background: #e0e0e0; height: 20px; border-radius: 10px; overflow: hidden;">
                            <div style="background: ${triggered ? '#e74c3c' : '#667eea'}; height: 100%; width: ${Math.min(progress, 100)}%; transition: width 0.3s;"></div>
                        </div>
                        <p style="margin-top: 5px; font-size: 0.9em;">${progress.toFixed(1)}% del umbral</p>
                    </div>
                    <p style="margin-top: 10px; color: #666;">
                        <strong>Ruta:</strong> ${zone.route_name || 'N/A'}<br>
                        <strong>Horario:</strong> ${zone.schedule_config || 'N/A'}<br>
                        <strong>Estado:</strong> 
                        <span class="status ${zone.status === 'LISTO_PARA_RECOLECCION' ? 'ready' : 'pending'}">
                            ${zone.status || 'PENDING'}
                        </span>
                    </p>
                    ${triggered ? '<p style="margin-top: 10px; color: #e74c3c; font-weight: bold;">🚨 ¡Umbral alcanzado! Listo para recolección</p>' : ''}
                `;
                
                container.appendChild(card);
            } catch (error) {
                console.error(`Error cargando métricas de zona ${zone.id}:`, error);
            }
        }
    } catch (error) {
        container.innerHTML = `<p class="message error">❌ Error: ${error.message}</p>`;
    }
}

// Auto-actualizar métricas cada 30 segundos
let metricsInterval = null;

function startMetricsAutoRefresh() {
    if (metricsInterval) {
        clearInterval(metricsInterval);
    }
    metricsInterval = setInterval(() => {
        const activeTab = document.querySelector('.tab-content.active');
        if (activeTab && activeTab.id === 'tab-monitoring') {
            loadZoneMetrics();
        }
    }, 30000);
}

// Iniciar auto-actualización cuando se carga la página
document.addEventListener('DOMContentLoaded', () => {
    startMetricsAutoRefresh();
});
