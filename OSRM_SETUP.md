# 🎉 OSRM - Motor de Rutas Configurado

## ✅ Lo que se creó

```
backend_latacunga_clean/
├── docker-compose.yaml          # ← Configuración del servicio OSRM
└── osrm-data/                   # ← Carpeta para datos del mapa
    ├── README.md                # Documentación rápida
    ├── INSTALLATION.md          # Guía detallada paso a paso
    ├── setup-osrm.ps1          # Script automático de instalación
    ├── test-osrm.ps1           # Script de pruebas
    └── .gitignore              # Excluye archivos pesados de Git
```

## 🚀 Instalación en 3 Pasos

### 1. Procesar el Mapa

Abre PowerShell en la carpeta `osrm-data`:

```powershell
cd "D:\Octavo Semestre\Tesis\backend_latacunga_clean\osrm-data"
.\setup-osrm.ps1
```

Este script:
- ✅ Descarga el mapa de Ecuador (~80 MB)
- ✅ Extrae la red vial
- ✅ Particiona el grafo
- ✅ Calcula pesos/tiempos
- ⏱️ Tiempo total: ~5-8 minutos

### 2. Levantar el Servicio

```powershell
cd ..
docker-compose up -d routing-engine
```

### 3. Probar

Espera 10 segundos y ejecuta:

```powershell
cd osrm-data
.\test-osrm.ps1
```

O abre en el navegador:
```
http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.612,-0.935?source=first
```

## 📖 Documentación

- **Guía Completa**: `osrm-data/INSTALLATION.md`
- **API Reference**: https://project-osrm.org/docs/v5.24.0/api/

## 🎯 Uso en tu Proyecto

El motor estará disponible en `http://localhost:5000` para:

✅ **Calcular rutas**: `/route/v1/driving/lon1,lat1;lon2,lat2`  
✅ **Optimizar tours**: `/trip/v1/driving/lon1,lat1;lon2,lat2;...?roundtrip=true`  
✅ **Snap a calles**: `/nearest/v1/driving/lon,lat`  
✅ **Matrices**: `/table/v1/driving/lon1,lat1;lon2,lat2;...`

## ⚡ Próximos Pasos

1. Ejecuta `.\setup-osrm.ps1` para procesar el mapa
2. Levanta el servicio con `docker-compose up -d`
3. Prueba con `.\test-osrm.ps1`
4. Integra con tu backend Go (crear cliente HTTP)

---

**¿Dudas?** Revisa `INSTALLATION.md` para troubleshooting y ejemplos.
