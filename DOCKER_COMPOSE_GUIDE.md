# 🚀 Sistema Latacunga Clean - Docker Compose

## ✅ Estado de Servicios

Todos los microservicios están corriendo exitosamente en contenedores Docker:

| Servicio | Puerto | Estado | Contenedor |
|----------|--------|--------|------------|
| **RabbitMQ** | 5672, 15672 | ✅ Healthy | latacunga_rabbitmq |
| **OSRM Routing** | 5000 | ✅ Running | latacunga_routing |
| **Auth Service** | 8080 | ✅ Running | latacunga_auth |
| **Fleet Service** | 8081 | ✅ Healthy | latacunga_fleet |
| **Incident Service** | 8082 | ✅ Running | latacunga_incident |
| **Scheduler Service** | 8083 | ✅ Healthy | latacunga_scheduler |
| **Operations Service** | 8085 | ✅ Healthy | latacunga_operations |

## 🎯 Inicio Rápido

### Levantar todos los servicios:
```powershell
docker-compose up -d
```

### Ver estado de servicios:
```powershell
docker-compose ps
```

### Ver logs de todos los servicios:
```powershell
docker-compose logs -f
```

### Ver logs de un servicio específico:
```powershell
docker-compose logs -f auth-service
docker-compose logs -f fleet-service
docker-compose logs -f incident-service
docker-compose logs -f scheduler-service
docker-compose logs -f operations-service
```

### Detener todos los servicios:
```powershell
docker-compose down
```

### Reconstruir y levantar servicios:
```powershell
docker-compose build
docker-compose up -d
```

## 🌐 URLs de Acceso

- **RabbitMQ Management UI**: http://localhost:15672
  - Usuario: `tesis`
  - Password: `tesis`

- **OSRM Routing**: http://localhost:5000
  - Ejemplo: http://localhost:5000/trip/v1/driving/-78.614,-0.933;-78.612,-0.935?source=first

- **Auth Service**: http://localhost:8080
- **Fleet Service**: http://localhost:8081
- **Incident Service**: http://localhost:8082
- **Scheduler Service**: http://localhost:8083
- **Operations Service**: http://localhost:8085
  - Swagger: http://localhost:8085/swagger/index.html

## 🔧 Configuración

### Variables de Entorno

Todos los servicios están configurados para usar:
- **Base de Datos**: Neon PostgreSQL (Cloud)
- **RabbitMQ**: Contenedor Docker (localhost:5672 dentro de la red)
- **OSRM**: Contenedor Docker (localhost:5000)

### Red Docker

Todos los servicios están conectados a la red `latacunga-network`, permitiendo comunicación entre contenedores usando nombres de servicio.

## 🧪 Pruebas E2E

Una vez que todos los servicios están corriendo, puedes ejecutar las pruebas E2E:

```powershell
# Ejecutar pruebas completas
.\run_e2e_test.ps1

# O directamente con Go
go run e2e_test.go
```

## 🛠️ Troubleshooting

### Un servicio no inicia:
```powershell
# Ver logs del servicio
docker logs latacunga_[servicio]

# Reiniciar el servicio
docker-compose restart [servicio]
```

### Limpiar y reiniciar todo:
```powershell
# Detener y eliminar contenedores
docker-compose down

# Eliminar volúmenes (cuidado: borra datos de RabbitMQ)
docker-compose down -v

# Reconstruir todo desde cero
docker-compose build --no-cache
docker-compose up -d
```

### Verificar conectividad:
```powershell
# Probar RabbitMQ
Invoke-WebRequest -Uri "http://localhost:15672" -UseBasicParsing

# Probar Auth Service
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/auth/register" -Method POST -UseBasicParsing

# Probar Operations Service
Invoke-WebRequest -Uri "http://localhost:8085/swagger/index.html" -UseBasicParsing
```

## 📦 Estructura de Servicios

```
backend_latacunga_clean/
├── docker-compose.yaml          # Orquestación de servicios
├── auth-service/
│   └── Dockerfile
├── fleet-service/
│   └── Dockerfile
├── incident-service/
│   └── Dockerfile
├── schedule-service/
│   └── Dockerfile
├── operations-service/
│   └── Dockerfile
├── routing-service/
│   └── Dockerfile
└── RabbitMQ/
    └── Dockerfile
```

## 🔄 Flujo de Comunicación

1. **Auth Service** (8080) → Gestión de usuarios y autenticación
2. **Fleet Service** (8081) → Gestión de vehículos y conductores
3. **Incident Service** (8082) → Gestión de incidentes y demandas
4. **Scheduler Service** (8083) → Planificación y asignación
5. **Operations Service** (8085) → Órdenes de trabajo
6. **Routing Service** → Optimización de rutas (consumidor RabbitMQ)
7. **OSRM** (5000) → Motor de rutas
8. **RabbitMQ** (5672) → Mensajería asíncrona

## ✨ Características

- ✅ Multi-stage Docker builds para imágenes optimizadas
- ✅ Health checks para todos los servicios
- ✅ Auto-restart en caso de fallos
- ✅ Red Docker privada para comunicación inter-servicios
- ✅ Volúmenes persistentes para RabbitMQ
- ✅ Configuración mediante variables de entorno
- ✅ Soporte para Go 1.24+

---

**Última actualización**: Noviembre 2025
**Versión**: 1.0.0
