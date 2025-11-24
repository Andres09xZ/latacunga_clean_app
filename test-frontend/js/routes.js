// Visualización de Rutas Optimizadas

async function loadRoutes() {
    const container = document.getElementById('routesList');
    container.innerHTML = '<p>Cargando rutas optimizadas...</p>';
    
    try {
        const response = await fetch(`${API_CONFIG.ROUTING_SERVICE}/api/v1/routes`);
        
        if (!response.ok) {
            throw new Error('No se pudo cargar las rutas');
        }
        
        const data = await response.json();
        
        if (!data.routes || Object.keys(data.routes).length === 0) {
            container.innerHTML = '<p class="info">⚠️ No hay rutas optimizadas disponibles</p>';
            return;
        }
        
        container.innerHTML = `<p class="info">📊 Total de zonas con rutas: ${data.total_zones}</p>`;
        
        // Ordenar por zone_id
        const sortedZones = Object.entries(data.routes).sort((a, b) => a[1].zone_id - b[1].zone_id);
        
        sortedZones.forEach(([zoneId, route]) => {
            const card = document.createElement('div');
            card.className = 'route-card';
            
            const distanceKm = (route.distance_m / 1000).toFixed(2);
            const durationMin = (route.duration_s / 60).toFixed(1);
            
            card.innerHTML = `
                <h3>🗺️ Zona ${route.zone_id}</h3>
                
                <div class="route-info">
                    <div class="route-info-item">
                        <label>Request ID:</label>
                        <span>${route.request_id}</span>
                    </div>
                    <div class="route-info-item">
                        <label>Distancia:</label>
                        <span>${distanceKm} km</span>
                    </div>
                    <div class="route-info-item">
                        <label>Duración:</label>
                        <span>${durationMin} min</span>
                    </div>
                    <div class="route-info-item">
                        <label>Creada:</label>
                        <span>${new Date(route.created_at).toLocaleString()}</span>
                    </div>
                </div>
                
                <div class="coordinates-list">
                    <h4>📍 Puntos de la Ruta (${route.coordinates.length} puntos)</h4>
                    <div id="coords-${zoneId}"></div>
                </div>
                
                <button onclick="toggleCoordinates('${zoneId}')" class="btn-primary" style="margin-top: 10px;">
                    👁️ Mostrar/Ocultar Coordenadas
                </button>
            `;
            
            container.appendChild(card);
            
            // Agregar coordenadas (inicialmente ocultas)
            const coordsContainer = document.getElementById(`coords-${zoneId}`);
            coordsContainer.style.display = 'none';
            
            route.coordinates.forEach((coord, index) => {
                const item = document.createElement('div');
                item.className = 'coordinate-item';
                
                // Detectar si es el depósito (primer punto)
                const isDepot = index === 0;
                if (isDepot) {
                    item.classList.add('depot');
                }
                
                item.innerHTML = `
                    <strong>${isDepot ? '🏢 DEPÓSITO (EPAGAL)' : `📌 Punto ${index}`}</strong>
                    <span>Lat: ${coord.latitude.toFixed(6)}, Lon: ${coord.longitude.toFixed(6)}</span>
                `;
                
                coordsContainer.appendChild(item);
            });
        });
        
    } catch (error) {
        container.innerHTML = `<p class="message error">❌ Error: ${error.message}</p>`;
    }
}

function toggleCoordinates(zoneId) {
    const container = document.getElementById(`coords-${zoneId}`);
    if (container.style.display === 'none') {
        container.style.display = 'block';
    } else {
        container.style.display = 'none';
    }
}

async function loadRouteByZone(zoneId) {
    const container = document.getElementById('routesList');
    container.innerHTML = '<p>Cargando ruta de la zona...</p>';
    
    try {
        const response = await fetch(`${API_CONFIG.ROUTING_SERVICE}/api/v1/routes/zone/${zoneId}`);
        
        if (!response.ok) {
            throw new Error(`No se encontró ruta para la zona ${zoneId}`);
        }
        
        const route = await response.json();
        
        const distanceKm = (route.distance_m / 1000).toFixed(2);
        const durationMin = (route.duration_s / 60).toFixed(1);
        
        container.innerHTML = `
            <div class="route-card">
                <h3>🗺️ Zona ${route.zone_id}</h3>
                
                <div class="route-info">
                    <div class="route-info-item">
                        <label>Request ID:</label>
                        <span>${route.request_id}</span>
                    </div>
                    <div class="route-info-item">
                        <label>Distancia:</label>
                        <span>${distanceKm} km</span>
                    </div>
                    <div class="route-info-item">
                        <label>Duración:</label>
                        <span>${durationMin} min</span>
                    </div>
                    <div class="route-info-item">
                        <label>Creada:</label>
                        <span>${new Date(route.created_at).toLocaleString()}</span>
                    </div>
                </div>
                
                <div class="coordinates-list">
                    <h4>📍 Puntos de la Ruta (${route.coordinates.length} puntos)</h4>
                    ${route.coordinates.map((coord, index) => {
                        const isDepot = index === 0;
                        return `
                            <div class="coordinate-item ${isDepot ? 'depot' : ''}">
                                <strong>${isDepot ? '🏢 DEPÓSITO (EPAGAL)' : `📌 Punto ${index}`}</strong>
                                <span>Lat: ${coord.latitude.toFixed(6)}, Lon: ${coord.longitude.toFixed(6)}</span>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        `;
        
    } catch (error) {
        container.innerHTML = `<p class="message error">❌ Error: ${error.message}</p>`;
    }
}
