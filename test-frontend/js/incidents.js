// Gestión de Incidentes

async function reportIncident(event) {
    event.preventDefault();
    
    if (!requireAuth()) return;
    
    const type = document.getElementById('incidentType').value;
    const lat = parseFloat(document.getElementById('incidentLat').value);
    const lon = parseFloat(document.getElementById('incidentLon').value);
    const description = document.getElementById('incidentDescription').value;
    const token = getAuthToken();
    
    try {
        const response = await fetch(`${API_CONFIG.INCIDENT_SERVICE}/api/v1/incidents`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({
                type: type,
                latitude: lat,
                longitude: lon,
                description: description,
                status: 'PENDING'
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showMessage('incidentMessage', 
                `✅ Incidente reportado exitosamente!\n` +
                `ID: ${data.incident_id}\n` +
                `Tipo: ${type}\n` +
                `Ubicación: ${lat}, ${lon}`, 
                'success'
            );
            
            // Limpiar descripción
            document.getElementById('incidentDescription').value = '';
        } else {
            showMessage('incidentMessage', `❌ Error: ${data.error || 'No se pudo reportar el incidente'}`, 'error');
        }
    } catch (error) {
        showMessage('incidentMessage', `❌ Error de conexión: ${error.message}`, 'error');
    }
}
