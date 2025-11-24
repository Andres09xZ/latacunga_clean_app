# 🧹 Test Frontend - Sistema de Limpieza Latacunga

Interfaz web para pruebas del sistema de gestión de limpieza de Latacunga.

## 📋 Requisitos Previos

- Todos los servicios backend corriendo:
  - Auth Service (puerto 8081)
  - Incident Service (puerto 8082)
  - Schedule Service (puerto 8083)
  - Routing Service (puerto 8086)
- Navegador web moderno
- Servidor HTTP simple (opcional)

## 🚀 Cómo Usar

### Opción 1: Abrir directamente
Simplemente abre el archivo `index.html` en tu navegador.

### Opción 2: Servidor HTTP (recomendado)
```bash
# Con Python 3
python -m http.server 8000

# Con Node.js (npx)
npx http-server -p 8000

# Con PHP
php -S localhost:8000
```

Luego abre: `http://localhost:8000`

## 📱 Funcionalidades

### 1. 👤 Registro de Usuarios

#### **Ciudadanos (OTP)**
⚠️ **IMPORTANTE:** Los ciudadanos se registran mediante OTP (One-Time Password)

**Flujo de registro:**
1. El ciudadano proporciona su número de teléfono
2. El sistema envía un código OTP por SMS
3. El ciudadano ingresa el código recibido
4. Se completa el registro y se genera el token

**Nota:** En el frontend actual, usa el endpoint `/api/v1/auth/register` con `role: CITIZEN` para pruebas. Para producción, implementa el flujo OTP completo.

#### **Operadores**
Los operadores se registran directamente con:
- Nombre
- Email
- Contraseña
- Zona asignada (ejemplo: URBANO_CENTRAL)

### 2. 🔐 Inicio de Sesión
Ambos tipos de usuarios pueden iniciar sesión con:
- Email
- Contraseña

El sistema devuelve un JWT token que se almacena localmente.

### 3. 🚨 Reportar Incidentes

**Tipos de incidentes disponibles:**
- 📦 **Punto de Acopio** (10 puntos)
- 🔴 **Zona Crítica** (8 puntos)
- 🐕 **Animal Muerto** (6 puntos)
- ♻️ **Zona Reciclaje** (3 puntos)

**Coordenadas de referencia:**
- **EPAGAL (Depósito):** -0.9364043, -78.6087099
- **Zona Central:** -0.935, -78.6175
- **Parque Vicente León:** -0.9321, -78.6132

### 4. ✅ Validación de Incidentes

Los operadores pueden:
- Ver incidentes pendientes
- Aprobar incidentes (se agregan al cálculo de score)
- Rechazar incidentes

### 5. 📊 Monitoreo de Zonas

Visualiza en tiempo real:
- Score actual de cada zona
- Umbral configurado
- Progreso hacia el umbral
- Estado de la zona
- Información de horarios y rutas

**Estados:**
- `PENDING`: Esperando acumulación
- `LISTO_PARA_RECOLECCION`: Umbral alcanzado

### 6. 🗺️ Rutas Optimizadas

Ver rutas optimizadas por OSRM que incluyen:
- 🏢 **Punto de partida (EPAGAL)** - Siempre es el primer punto
- 📌 **Puntos de recolección** - Ordenados óptimamente
- Distancia total (km)
- Duración estimada (minutos)
- Coordenadas de cada punto

**Características de las rutas:**
- Inician desde EPAGAL (depósito)
- Visitan todos los puntos optimizados
- **NO retornan al depósito** (roundtrip=false)
- Minimizan distancia y tiempo

## 🎯 Flujo Completo de Prueba

1. **Registrar usuarios:**
   ```
   - 1 Ciudadano
   - 1 Operador (zona: URBANO_CENTRAL)
   ```

2. **Reportar incidentes como ciudadano:**
   ```
   - Reportar 5 "Punto de Acopio" (10 pts c/u)
   - Total: 50 puntos (alcanza umbral)
   ```

3. **Validar incidentes como operador:**
   ```
   - Ir a pestaña "Validación"
   - Aprobar todos los incidentes
   ```

4. **Monitorear zona:**
   ```
   - Ir a pestaña "Monitoreo"
   - Ver score: 50/50
   - Estado: LISTO_PARA_RECOLECCION
   ```

5. **Ver ruta optimizada:**
   ```
   - Ir a pestaña "Rutas"
   - Ver ruta con EPAGAL + 5 puntos
   - Verificar distancia y duración
   ```

## 🔧 Configuración

Edita `js/config.js` para cambiar las URLs de los servicios:

```javascript
const API_CONFIG = {
    AUTH_SERVICE: 'http://localhost:8081',
    INCIDENT_SERVICE: 'http://localhost:8082',
    SCHEDULE_SERVICE: 'http://localhost:8083',
    ROUTING_SERVICE: 'http://localhost:8086'
};
```

## 📊 Pesos de Incidentes

| Tipo | Peso | Descripción |
|------|------|-------------|
| Punto de Acopio | 10 | Contenedor de basura lleno |
| Zona Crítica | 8 | Área con alta acumulación |
| Animal Muerto | 6 | Requiere atención urgente |
| Zona Reciclaje | 3 | Punto de reciclaje |

## 🚨 Solución de Problemas

### "Error de conexión"
- Verifica que todos los servicios estén corriendo
- Revisa las URLs en `config.js`
- Verifica CORS en los servicios backend

### "No se pudo registrar"
- Verifica que el email no esté duplicado
- Asegúrate de que todos los campos estén llenos
- Revisa los logs del auth-service

### "No hay rutas disponibles"
- Asegúrate de haber alcanzado el umbral
- Verifica que los incidentes fueron aprobados
- Revisa que OSRM esté corriendo (puerto 5000)

### "Se requieren al menos 2 puntos"
- Este error ocurría cuando solo había 1 incidente
- Ya está corregido: EPAGAL se agrega automáticamente
- Ahora mínimo son 2 puntos (DEPOT + 1 incidente)

## 📝 Notas Técnicas

- **Autenticación:** JWT Bearer tokens
- **Storage:** LocalStorage del navegador
- **Auto-refresh:** Métricas se actualizan cada 30 segundos
- **CORS:** Los servicios deben permitir peticiones del frontend
- **Polyline:** Las rutas usan Google Polyline encoding

## 🎨 Personalización

### Estilos
Edita `css/styles.css` para cambiar colores, fuentes, etc.

### Funcionalidades
Los archivos JS están modularizados:
- `auth.js` - Autenticación y registro
- `incidents.js` - Gestión de incidentes
- `validation.js` - Validación de incidentes
- `monitoring.js` - Monitoreo de zonas
- `routes.js` - Visualización de rutas
- `app.js` - Funciones generales

## 📞 Soporte

Para cualquier problema, revisa los logs de los servicios backend:
- Auth Service
- Incident Service
- Schedule Service
- Routing Service
