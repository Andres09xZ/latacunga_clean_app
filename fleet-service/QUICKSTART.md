# 🚀 Guía de Inicio Rápido - Fleet Service

## Prerequisitos

Antes de comenzar, asegúrate de tener instalado:

- ✅ **Go 1.21+**: [Descargar aquí](https://golang.org/dl/)
- ✅ **PostgreSQL** (o cuenta en Neon Serverless): [Neon](https://neon.tech/)
- ✅ **RabbitMQ**: [Docker](https://hub.docker.com/_/rabbitmq) o [CloudAMQP](https://www.cloudamqp.com/)
- ✅ **Git**: Para clonar el repositorio

## Paso 1: Configuración del Entorno

### 1.1. Clonar o navegar al proyecto
```bash
cd "d:\Octavo Semestre\Tesis\backend_latacunga_clean\fleet-service"
```

### 1.2. Copiar archivo de configuración
```bash
copy .env.example .env
```

### 1.3. Editar `.env` con tus credenciales

**PostgreSQL (Neon Serverless):**
```env
DB_HOST=ep-xxx-xxx.us-east-2.aws.neon.tech
DB_PORT=5432
DB_USER=tu_usuario
DB_PASSWORD=tu_password
DB_NAME=fleet_db
DB_SSLMODE=require
```

**RabbitMQ:**
```env
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
# O si usas CloudAMQP:
# RABBITMQ_URL=amqps://user:pass@xxx.cloudamqp.com/vhost
```

**Server:**
```env
SERVER_PORT=8082
GIN_MODE=debug
```

## Paso 2: Instalar Dependencias

### Opción A: Con Make (recomendado)
```bash
make deps
```

### Opción B: Manual
```bash
go mod download
go mod tidy
```

## Paso 3: Iniciar RabbitMQ (si no está corriendo)

### Con Docker:
```bash
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

Accede al panel de administración: http://localhost:15672
- Usuario: `guest`
- Password: `guest`

## Paso 4: Crear Base de Datos

### En Neon Serverless:
1. Ve a https://console.neon.tech/
2. Crea un nuevo proyecto
3. Copia la cadena de conexión
4. Actualiza el `.env` con los valores

### PostgreSQL Local:
```bash
createdb fleet_db
```

## Paso 5: Ejecutar el Servicio

### Opción A: Con Make
```bash
make run
```

### Opción B: Con Go directamente
```bash
go run cmd/fleet-service/main.go
```

### Opción C: Compilar y ejecutar
```bash
make build
.\bin\fleet-service.exe
```

## Paso 6: Verificar que Funciona

### Health Check:
```bash
curl http://localhost:8082/health
```

**Respuesta esperada:**
```json
{
  "service": "fleet-service",
  "status": "healthy"
}
```

### Ver logs:
Si todo está bien, deberías ver:
```
🚀 Iniciando Fleet Service...
✓ Configuración cargada exitosamente
✓ Conectado a PostgreSQL exitosamente
Ejecutando migraciones...
✓ Migraciones ejecutadas exitosamente
✓ Conectado a RabbitMQ
✓ EventPublisher inicializado correctamente
✓ IdentityConsumer configurado - Queue: q.fleet.identity-sync
✓ ResourceConsumer configurado - Queue: q.fleet.resource-requests
✓ WorkorderConsumer configurado - Queue: q.fleet.workorder-updates
✓ Todos los consumers están activos
🔵 IdentityConsumer escuchando en cola: q.fleet.identity-sync
🟢 ResourceConsumer escuchando en cola: q.fleet.resource-requests
🟡 WorkorderConsumer escuchando en cola: q.fleet.workorder-updates
🌐 Servidor HTTP escuchando en :8082
```

## Paso 7: Cargar Datos de Prueba

### Con psql:
```bash
psql "postgresql://user:pass@host/fleet_db?sslmode=require" -f scripts/init_data.sql
```

### Con cliente de PostgreSQL:
Ejecuta el contenido de `scripts/init_data.sql`

## Paso 8: Probar los Endpoints

### Clock-In (Inicio de Turno)

**PowerShell:**
```powershell
$body = @{
    driver_id = "550e8400-e29b-41d4-a716-446655440000"
    truck_plate = "ABC-1234"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8082/api/v1/shifts/clock-in" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body
```

**curl:**
```bash
curl -X POST http://localhost:8082/api/v1/shifts/clock-in \
  -H "Content-Type: application/json" \
  -d '{"driver_id": "550e8400-e29b-41d4-a716-446655440000", "truck_plate": "ABC-1234"}'
```

### Clock-Out (Fin de Turno)

**PowerShell:**
```powershell
$body = @{
    driver_id = "550e8400-e29b-41d4-a716-446655440000"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8082/api/v1/shifts/clock-out" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body
```

## Paso 9: Monitorear RabbitMQ

Accede al panel de RabbitMQ:
- URL: http://localhost:15672
- Usuario: `guest`
- Password: `guest`

Verifica que existan los exchanges y colas:
- **Exchanges**: `city.cleaning.resources`, `identity.management`, `planning.scheduler`, `operations.workorders`
- **Queues**: `q.fleet.identity-sync`, `q.fleet.resource-requests`, `q.fleet.workorder-updates`

## Comandos Útiles

### Ver estado de conductores:
```sql
SELECT id, full_name, status FROM drivers;
```

### Ver turnos activos:
```sql
SELECT * FROM active_shifts WHERE is_active = true;
```

### Ver camiones disponibles:
```sql
SELECT plate, type, status FROM trucks WHERE status = 'DISPONIBLE';
```

### Ver logs en tiempo real:
```bash
make run
```

### Compilar para producción:
```bash
make build
```

### Ejecutar tests:
```bash
make test
```

## Troubleshooting

### Error: "error al conectar con la base de datos"
- Verifica las credenciales en `.env`
- Verifica que el host sea accesible
- Verifica que el puerto 5432 esté abierto

### Error: "error al conectar con RabbitMQ"
- Verifica que RabbitMQ esté corriendo: `docker ps | grep rabbitmq`
- Verifica la URL en `.env`
- Verifica que el puerto 5672 esté abierto

### Error: "conductor no encontrado"
- Los conductores se crean mediante eventos desde Identity Service
- O créalos manualmente en la base de datos

### Error: "camión no encontrado"
- Ejecuta el script `scripts/init_data.sql` para crear camiones de prueba

## Próximos Pasos

1. **Integración con Identity Service**: Configura el Identity Service para publicar eventos de creación de operadores
2. **Integración con Scheduler Service**: Configura el Scheduler para solicitar recursos
3. **Integración con Operations Service**: Configura Operations para publicar eventos de workorder completada
4. **Testing Completo**: Ejecuta el flujo end-to-end con todos los microservicios

## Recursos Adicionales

- 📖 **README.md**: Documentación general del proyecto
- 📖 **TECHNICAL_DOCS.md**: Documentación técnica detallada
- 📖 **API_TESTING.md**: Ejemplos de testing con curl y PowerShell
- 📦 **postman_collection.json**: Colección de Postman para importar

## Soporte

Si encuentras problemas:
1. Revisa los logs del servicio
2. Revisa los logs de RabbitMQ
3. Verifica la conectividad de base de datos
4. Consulta la documentación técnica

---

✅ **¡Fleet Service está listo para usar!**
