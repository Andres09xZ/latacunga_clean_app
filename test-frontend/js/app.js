// Funciones generales de la aplicación

function showTab(tabName) {
    // Ocultar todos los tabs
    const tabs = document.querySelectorAll('.tab-content');
    tabs.forEach(tab => tab.classList.remove('active'));
    
    // Desactivar todos los botones
    const buttons = document.querySelectorAll('.tab-btn');
    buttons.forEach(btn => btn.classList.remove('active'));
    
    // Mostrar el tab seleccionado
    const selectedTab = document.getElementById(`tab-${tabName}`);
    if (selectedTab) {
        selectedTab.classList.add('active');
    }
    
    // Activar el botón correspondiente
    const selectedButton = Array.from(buttons).find(btn => 
        btn.textContent.includes(getTabIcon(tabName))
    );
    if (selectedButton) {
        selectedButton.classList.add('active');
    }
    
    // Cargar datos si es necesario
    if (tabName === 'monitoring') {
        loadZoneMetrics();
    } else if (tabName === 'routes') {
        loadRoutes();
    } else if (tabName === 'validation') {
        if (isAuthenticated()) {
            loadPendingIncidents();
        }
    }
}

function getTabIcon(tabName) {
    const icons = {
        'login': '🔐',
        'register': '📝',
        'incidents': '🚨',
        'validation': '✅',
        'monitoring': '📊',
        'routes': '🗺️'
    };
    return icons[tabName] || '';
}

function showMessage(elementId, message, type) {
    const element = document.getElementById(elementId);
    if (element) {
        element.textContent = message;
        element.className = `message ${type}`;
        element.style.display = 'block';
        
        // Auto-ocultar después de 5 segundos
        setTimeout(() => {
            element.style.display = 'none';
        }, 5000);
    }
}

// Manejo de errores global
window.addEventListener('error', (event) => {
    console.error('Error global:', event.error);
});

// Verificar estado de los servicios al cargar
async function checkServicesHealth() {
    const services = [
        { name: 'Auth Service', url: `${API_CONFIG.AUTH_SERVICE}/health` },
        { name: 'Incident Service', url: `${API_CONFIG.INCIDENT_SERVICE}/health` },
        { name: 'Schedule Service', url: `${API_CONFIG.SCHEDULE_SERVICE}/health` },
        { name: 'Routing Service', url: `${API_CONFIG.ROUTING_SERVICE}/health` }
    ];
    
    console.log('🔍 Verificando estado de servicios...');
    
    for (const service of services) {
        try {
            const response = await fetch(service.url, { method: 'GET' });
            if (response.ok) {
                console.log(`✅ ${service.name}: OK`);
            } else {
                console.warn(`⚠️ ${service.name}: ${response.status}`);
            }
        } catch (error) {
            console.error(`❌ ${service.name}: ${error.message}`);
        }
    }
}

// Formatear fechas
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('es-EC', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Inicialización de la aplicación
document.addEventListener('DOMContentLoaded', () => {
    console.log('🚀 Aplicación inicializada');
    checkServicesHealth();
    updateUserInfo();
});
