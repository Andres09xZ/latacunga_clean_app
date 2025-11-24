// Validación de Incidentes

async function loadPendingIncidents() {
    if (!requireAuth()) return;
    
    const token = getAuthToken();
    const container = document.getElementById('pendingIncidentsList');
    container.innerHTML = '<p>Cargando incidentes pendientes...</p>';
    
    try {
        const response = await fetch(`${API_CONFIG.INCIDENT_SERVICE}/api/v1/incidents/pending`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (!response.ok) {
            throw new Error('No se pudo cargar los incidentes');
        }
        
        const incidents = await response.json();
        
        if (!incidents || incidents.length === 0) {
            container.innerHTML = '<p class="info">✅ No hay incidentes pendientes de validación</p>';
            return;
        }
        
        container.innerHTML = '';
        incidents.forEach(incident => {
            const item = document.createElement('div');
            item.className = 'incident-item';
            item.innerHTML = `
                <h4>🚨 ${incident.type || 'Incidente'}</h4>
                <p><strong>ID:</strong> ${incident.incident_id}</p>
                <p><strong>Ubicación:</strong> ${incident.latitude}, ${incident.longitude}</p>
                <p><strong>Descripción:</strong> ${incident.description || 'N/A'}</p>
                <p><strong>Estado:</strong> ${incident.status}</p>
                <p><strong>Reportado:</strong> ${new Date(incident.created_at).toLocaleString()}</p>
                <div class="actions">
                    <button class="btn-success" onclick="validateIncident('${incident.incident_id}', true)">
                        ✅ Aprobar
                    </button>
                    <button class="btn-danger" onclick="validateIncident('${incident.incident_id}', false)">
                        ❌ Rechazar
                    </button>
                </div>
            `;
            container.appendChild(item);
        });
    } catch (error) {
        container.innerHTML = `<p class="message error">❌ Error: ${error.message}</p>`;
    }
}

async function validateIncident(incidentId, approved) {
    if (!requireAuth()) return;
    
    const token = getAuthToken();
    const action = approved ? 'approve' : 'reject';
    
    try {
        const response = await fetch(`${API_CONFIG.INCIDENT_SERVICE}/api/v1/incidents/${incidentId}/${action}`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });
        
        const data = await response.json();
        
        if (response.ok) {
            alert(`✅ Incidente ${approved ? 'aprobado' : 'rechazado'} exitosamente`);
            loadPendingIncidents(); // Recargar lista
            
            // Si fue aprobado, actualizar métricas
            if (approved) {
                setTimeout(() => loadZoneMetrics(), 1000);
            }
        } else {
            alert(`❌ Error: ${data.error || 'No se pudo validar el incidente'}`);
        }
    } catch (error) {
        alert(`❌ Error de conexión: ${error.message}`);
    }
}
