# Fleet Service - Gestión de Flota de Camiones

Microservicio encargado de gestionar la flota de camiones, conductores, turnos y asignación inteligente de recursos.

## 🚀 Stack Tecnológico

- **Lenguaje:** Go 1.21+
- **Framework HTTP:** Gin Gonic
- **ORM:** GORM (PostgreSQL Driver)
- **Mensajería:** RabbitMQ (amqp091-go)
- **Base de Datos:** PostgreSQL (Neon Serverless)

## 📋 Características Principales

### 1. Gestión de Turnos
- **Clock-In:** Los operadores registran entrada con su camión asignado
- **Clock-Out:** Finalización de turno y liberación de recursos
- Validaciones automáticas de disponibilidad

### 2. Motor de Asignación Inteligente
- Algoritmo de búsqueda con priorización por zona preferida
- Sistema de bloqueo (locking) para evitar doble asignación
- Matching automático conductor-camión según tipo requerido

### 3. Arquitectura de Eventos (RabbitMQ)
- **Sincronización de Identidad:** Recibe operadores creados
- **Solicitud de Recursos:** Asigna conductores bajo demanda
- **Liberación de Recursos:** Libera conductores al completar tareas

## 🗂️ Estructura del Proyecto

```
fleet-service/
├── cmd/
│   └── fleet-service/
│       └── main.go                 # Punto de entrada
├── internal/
│   ├── config/
│   │   └── config.go               # Configuración de variables de entorno
│   ├── database/
│   │   └── database.go             # Conexión y migraciones
│   ├── models/
│   │   └── models.go               # Modelos de datos (Truck, Driver, etc.)
│   ├── handlers/
│   │   └── shift_handler.go        # Handlers REST (clock-in/out)
│   ├── services/
│   │   ├── allocation_service.go   # Motor de asignación inteligente
│   │   └── publisher.go            # Publicador de eventos RabbitMQ
│   └── consumers/
│       ├── identity_consumer.go    # Consumer sincronización identidad
│       ├── resource_consumer.go    # Consumer solicitudes de recursos
│       └── workorder_consumer.go   # Consumer actualizaciones de órdenes
├── pkg/
│   └── utils/
│       └── logger.go               # Utilidades de logging
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## 🔧 Configuración e Instalación

### 1. Clonar y configurar
```bash
cd fleet-service
cp .env.example .env
# Editar .env con tus credenciales
```

### 2. Instalar dependencias
```bash
go mod download
```

### 3. Ejecutar migraciones
Las migraciones se ejecutan automáticamente al iniciar el servicio.

### 4. Iniciar el servicio
```bash
go run cmd/fleet-service/main.go
```

### 5. Acceder a la documentación Swagger
Una vez iniciado el servicio:
```
http://localhost:8082/swagger/index.html
```

## 📚 Documentación API

La API está completamente documentada con **Swagger/OpenAPI**. Accede a la documentación interactiva en:

**URL:** `http://localhost:8082/swagger/index.html`

### Regenerar documentación Swagger
```bash
make swagger
# O manualmente:
swag init -g cmd/fleet-service/main.go -o docs
```

## 🌐 API Endpoints

### POST `/api/v1/shifts/clock-in`
Registrar entrada de turno.

**Request:**
```json
{
  "driver_id": "uuid-del-conductor",
  "truck_plate": "ABC-1234"
}
```

**Response (200):**
```json
{
  "shift_id": "uuid-del-turno",
  "message": "Turno iniciado exitosamente"
}
```

### POST `/api/v1/shifts/clock-out`
Finalizar turno activo.

**Request:**
```json
{
  "driver_id": "uuid-del-conductor"
}
```

**Response (200):**
```json
{
  "message": "Turno finalizado exitosamente"
}
```

## 📡 Eventos RabbitMQ

### Consumidos

| Cola | Routing Key | Descripción |
|------|-------------|-------------|
| `q.fleet.identity-sync` | `identity.operator.created.v1` | Sincroniza nuevos operadores |
| `q.fleet.resource-requests` | `planning.resource.requested.v1` | Solicita asignación de conductor |
| `q.fleet.workorder-updates` | `workorder.completed.v1` | Libera conductor al completar tarea |

### Publicados

| Exchange | Routing Key | Descripción |
|----------|-------------|-------------|
| `city.cleaning.resources` | `resource.available.v1` | Notifica recurso disponible (clock-in) |
| `city.cleaning.resources` | `resources.driver.allocated.v1` | Confirma asignación de conductor |

## 🗃️ Modelos de Datos

### Truck
- `ID`, `Plate` (Unique), `Type`, `Status`
- **Types:** `CARGA_LATERAL`, `CARGA_POSTERIOR`
- **Status:** `DISPONIBLE`, `EN_USO`, `MANTENIMIENTO`

### Driver
- `ID` (UUID), `FullName`, `Status`
- **Status:** `OFFLINE`, `DISPONIBLE`, `OCUPADO`

### OperatorProfile
- `DriverID`, `PreferredZoneID`, `CanDriveLateral`, `CanDrivePosterior`

### ActiveShift
- `ID`, `DriverID`, `TruckID`, `StartTime`, `EndTime`, `IsActive`

## 🔒 Sistema de Bloqueo (Locking)

El motor de asignación usa transacciones SQL con `UPDATE` atómico para evitar condiciones de carrera:

```sql
UPDATE drivers SET status = 'OCUPADO' WHERE id = ? AND status = 'DISPONIBLE'
```

## 📊 Algoritmo de Asignación

1. **Filtrado:** Conductores con turno activo, tipo de camión correcto y disponibles
2. **Priorización:** Conductores con zona preferida coincidente
3. **Bloqueo:** Actualización atómica del estado a `OCUPADO`
4. **Publicación:** Evento de asignación exitosa

## 🧪 Testing

```bash
# Ejecutar tests unitarios
go test ./...

# Con cobertura
go test -cover ./...
```

## 📝 Logs

El servicio utiliza logging estructurado con niveles INFO, WARN y ERROR.

## 🤝 Contribución

1. Fork el repositorio
2. Crea tu feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push al branch (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📄 Licencia

Este proyecto es parte del sistema de gestión de limpieza urbana de Latacunga.
