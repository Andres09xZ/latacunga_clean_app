# 🚀 Quick Start - OSRM en 3 Comandos

## Paso 1: Procesar Mapa de Ecuador

```powershell
cd "D:\Octavo Semestre\Tesis\backend_latacunga_clean\osrm-data"
.\setup-osrm.ps1
```

**Salida esperada:**
```
🗺️  Configurando OSRM - Motor de Rutas para Latacunga Clean

📥 Paso 1/4: Descargando mapa de Ecuador (~80MB)...
   ✅ Descarga completada

⚙️  Paso 2/4: Extrayendo red vial (esto toma ~2-5 min)...
   ✅ Extracción completada

🔀 Paso 3/4: Particionando grafo...
   ✅ Partición completada

⚡ Paso 4/4: Personalizando pesos...
   ✅ Personalización completada

🎉 ¡OSRM configurado exitosamente!
```

---

## Paso 2: Levantar el Servicio Docker

```powershell
cd ..
docker-compose up -d routing-engine
```

**Salida esperada:**
```
Creating network "backend_latacunga_clean_latacunga-network" with driver "bridge"
Creating latacunga_routing ... done
```

**Verificar que está corriendo:**
```powershell
docker ps
```

Deberías ver:
```
CONTAINER ID   IMAGE                  STATUS         PORTS                    NAMES
xxxxxx         osrm/osrm-backend      Up 5 seconds   0.0.0.0:5000->5000/tcp   latacunga_routing
```

---

## Paso 3: Probar el Servicio

**Opción A: Script Automático**
```powershell
cd osrm-data
.\test-osrm.ps1
```

**Salida esperada:**
```
🧪 Testeando OSRM - Motor de Rutas

1️⃣  Health Check...
   ✅ OSRM está activo

2️⃣  Ruta simple (2 puntos en centro de Latacunga)...
   ✅ Distancia: 312 m | Tiempo: 0.8 min

3️⃣  Trip optimizado (5 puntos de recolección)...
   ✅ Ruta optimizada: 5 puntos | 1847 m | 4.2 min

4️⃣  Nearest (snap a calle más cercana)...
   ✅ Snap: [-78.614012, -0.932987] | Distancia al punto: 1.3 m

✨ Tests completados. OSRM está listo para producción.
```

---

**Opción B: Test Manual en el Navegador**

Abre esta URL:
```
http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.612,-0.935?source=first
```

Deberías ver un JSON similar a:
```json
{
  "code": "Ok",
  "trips": [{
    "distance": 312.4,
    "duration": 47.8,
    "geometry": "..."
  }],
  "waypoints": [...]
}
```

---

## ✅ ¡Listo! Ahora puedes:

1. **Calcular rutas en tu backend Go**:
   ```go
   resp, _ := http.Get("http://localhost:5000/route/v1/driving/-78.614,-0.933;-78.612,-0.935")
   ```

2. **Optimizar tours de recolección**:
   ```
   GET /trip/v1/driving/punto1;punto2;punto3;...?roundtrip=true
   ```

3. **Validar coordenadas de contenedores**:
   ```
   GET /nearest/v1/driving/lon,lat
   ```

---

## 🔧 Comandos Útiles

**Ver logs en tiempo real:**
```powershell
docker logs latacunga_routing -f
```

**Reiniciar el servicio:**
```powershell
docker-compose restart routing-engine
```

**Detener todo:**
```powershell
docker-compose down
```

---

## ⚠️ Si algo falla

1. **"Cannot connect to Docker daemon"**
   - Abre Docker Desktop y espera a que esté completamente iniciado

2. **"Address already in use"**
   - Cambia el puerto en `docker-compose.yaml`: `"5001:5000"`

3. **El servicio no responde**
   - Espera 10-15 segundos (OSRM carga el grafo en RAM)
   - Verifica logs: `docker logs latacunga_routing`

4. **Archivos .osrm no encontrados**
   - Re-ejecuta `.\setup-osrm.ps1` desde la carpeta `osrm-data`

---

## 📚 Más Info

- **Guía Completa**: `osrm-data/INSTALLATION.md`
- **API Docs**: https://project-osrm.org/docs/v5.24.0/api/
- **OpenStreetMap**: https://www.openstreetmap.org
