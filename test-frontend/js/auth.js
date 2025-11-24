// Autenticación y Registro

// Variable global para almacenar el teléfono temporalmente
let tempCitizenPhone = '';

async function requestCitizenOTP(event) {
    event.preventDefault();
    
    const phone = document.getElementById('citizenPhone').value;
    
    // Validar formato E.164 básico
    if (!phone.match(/^\+[1-9]\d{1,14}$/)) {
        showMessage('citizenMessage', '❌ Formato de teléfono inválido. Use E.164 (ejemplo: +593987654321)', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/otp/send`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                phone: phone
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            tempCitizenPhone = phone; // Guardar para verificación
            showMessage('citizenMessage', `✅ ${data.message || 'Código OTP enviado al teléfono'} (Revisa la consola del servidor)`, 'success');
            
            // Mostrar sección de verificación
            document.getElementById('otpRequestForm').style.display = 'none';
            document.getElementById('otpVerifySection').style.display = 'block';
            document.getElementById('otpCode').focus();
        } else {
            showMessage('citizenMessage', `❌ Error: ${data.message || data.error || 'No se pudo enviar OTP'}`, 'error');
        }
    } catch (error) {
        showMessage('citizenMessage', `❌ Error de conexión: ${error.message}`, 'error');
    }
}

async function verifyCitizenOTP(event) {
    event.preventDefault();
    
    const code = document.getElementById('otpCode').value;
    
    if (!tempCitizenPhone) {
        showMessage('citizenMessage', '❌ Error: Primero debes solicitar un código OTP', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/otp/verify`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                phone: tempCitizenPhone,
                code: code
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // Guardar token y datos de usuario
            localStorage.setItem(STORAGE_KEYS.TOKEN, data.access_token);
            localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify({
                id: data.user_id,
                phone: tempCitizenPhone,
                role: 'user',
                name: tempCitizenPhone // Usar teléfono como nombre por defecto
            }));
            
            updateUserInfo();
            showMessage('citizenMessage', `✅ Registro exitoso! Bienvenido ciudadano`, 'success');
            
            // Resetear formulario
            resetOTPForm();
            
            // Cambiar a tab de incidentes
            setTimeout(() => {
                showTab('incidents');
            }, 2000);
        } else {
            showMessage('citizenMessage', `❌ Error: ${data.message || data.error || 'Código OTP inválido'}`, 'error');
        }
    } catch (error) {
        showMessage('citizenMessage', `❌ Error de conexión: ${error.message}`, 'error');
    }
}

function resetOTPForm() {
    document.getElementById('otpRequestForm').style.display = 'block';
    document.getElementById('otpVerifySection').style.display = 'none';
    document.getElementById('citizenPhone').value = '';
    document.getElementById('otpCode').value = '';
    tempCitizenPhone = '';
    showMessage('citizenMessage', '', '');
}

async function registerOperator(event) {
    event.preventDefault();
    
    const name = document.getElementById('operatorName').value;
    const email = document.getElementById('operatorEmail').value;
    const password = document.getElementById('operatorPassword').value;
    const zone = document.getElementById('operatorZone').value;
    
    try {
        const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/register-operator`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                name: name,
                email: email,
                password: password,
                zone_name: zone
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showMessage('operatorMessage', `✅ Operador registrado exitosamente. Token: ${data.token}`, 'success');
            document.getElementById('operatorName').value = '';
            document.getElementById('operatorEmail').value = '';
            document.getElementById('operatorPassword').value = '';
            document.getElementById('operatorZone').value = '';
        } else {
            showMessage('operatorMessage', `❌ Error: ${data.error || 'No se pudo registrar'}`, 'error');
        }
    } catch (error) {
        showMessage('operatorMessage', `❌ Error de conexión: ${error.message}`, 'error');
    }
}

async function login(event) {
    event.preventDefault();
    
    const email = document.getElementById('loginEmail').value;
    const password = document.getElementById('loginPassword').value;
    
    try {
        const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                email: email,
                password: password
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // Guardar token y datos de usuario
            localStorage.setItem(STORAGE_KEYS.TOKEN, data.token);
            localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify({
                id: data.user_id,
                name: data.name,
                email: data.email,
                role: data.role
            }));
            
            updateUserInfo();
            showMessage('loginMessage', `✅ Bienvenido ${data.name}`, 'success');
            
            // Limpiar formulario
            document.getElementById('loginEmail').value = '';
            document.getElementById('loginPassword').value = '';
        } else {
            showMessage('loginMessage', `❌ Error: ${data.error || 'Credenciales inválidas'}`, 'error');
        }
    } catch (error) {
        showMessage('loginMessage', `❌ Error de conexión: ${error.message}`, 'error');
    }
}

function logout() {
    localStorage.removeItem(STORAGE_KEYS.TOKEN);
    localStorage.removeItem(STORAGE_KEYS.USER);
    updateUserInfo();
    showTab('login');
}

function updateUserInfo() {
    const userInfo = document.getElementById('userInfo');
    const userName = document.getElementById('userName');
    const userData = getUserData();
    
    if (userData) {
        userInfo.style.display = 'flex';
        userName.textContent = `${userData.name} (${userData.role})`;
    } else {
        userInfo.style.display = 'none';
    }
}

function getAuthToken() {
    return localStorage.getItem(STORAGE_KEYS.TOKEN);
}

function getUserData() {
    const data = localStorage.getItem(STORAGE_KEYS.USER);
    return data ? JSON.parse(data) : null;
}

function isAuthenticated() {
    return !!getAuthToken();
}

function requireAuth() {
    if (!isAuthenticated()) {
        alert('⚠️ Debes iniciar sesión primero');
        showTab('login');
        return false;
    }
    return true;
}

// Inicializar al cargar la página
document.addEventListener('DOMContentLoaded', () => {
    updateUserInfo();
});
