# RabbitMQ Configuration - Latacunga Backend

Custom RabbitMQ Docker image con topología predefinida, plugins y configuración lista para producción.

## 📋 Características

- **RabbitMQ 3.13 Management** con interfaz web integrada
- **Plugins habilitados**: 
  - `rabbitmq_management` - Web UI y API REST
  - `rabbitmq_prometheus` - Métricas Prometheus
- **Topología predefinida**:
  - 4 Exchanges (direct y fanout)
  - 16 Queues (incluyendo Dead Letter Queues)
  - 17 Bindings configurados automáticamente
- **Auto-importación**: Definiciones cargadas automáticamente al iniciar
- **Healthcheck HTTP listo para Render**: usa `/api/alarms` en el puerto 15672
- **Documentación completa**: guías locales y despliegue en Render

## 🧭 Arquitectura (alto nivel)

```
Servicios (productores/consumidores)
   │         └──→ incidente.cmd (direct) ── rutas/routing_key → *.q
   └──→ fanouts: incidente.validado.fanout / incidente.rechazado.fanout → múltiples colas

Dead Letter Exchange (dlx.direct) ──> *.dlq
```

Topología pre-creada: 4 exchanges, 16 colas (8 principales + 8 DLQ) y 17 bindings.

## 🚀 Uso Rápido

### Opción 1: desde Docker Hub (recomendado)

```powershell
docker-compose up -d
```

El `docker-compose.yml` ya referencia la imagen publicada en Docker Hub.

### Opción 2: construir localmente

```powershell
docker build -t mrengineer09/rabbitmq:plugins .
docker-compose up -d
```

Puertos:
- 5672: AMQP (conexiones de aplicaciones)
- 15672: Management UI (HTTP)
- 25672: Clustering (interno)
Opcional métricas Prometheus: 15692 (exponer en compose si lo necesitas)


## 📊 Topología

### Exchanges
- `incidente.cmd` (direct)
- `incidente.validado.fanout` (fanout)
- `incidente.rechazado.fanout` (fanout)
- `dlx.direct` (Dead Letter Exchange)

### Queues
**Principales:**
- validacion.q
- incidentes.q
- tareas.q
- ubicacion.q
- horarios.q
- acopios.q
- rutas.q
- auditoria.q

**Dead Letter Queues (DLQ):**
- validacion.dlq
- incidentes.dlq
- tareas.dlq
- ubicacion.dlq
- horarios.dlq
- acopios.dlq
- rutas.dlq
- auditoria.dlq

## 🔧 Configuración


Modifica según tus necesidades de seguridad.

### Plugins Habilitados (enabled_plugins)

```
[rabbitmq_management,rabbitmq_prometheus].
```

### RabbitMQ Configuration (rabbitmq.conf)

```conf
management.load_definitions = /etc/rabbitmq/definitions.json
default_vhost = /
loopback_users.guest = false
```

### Healthchecks (local y Render)

- El contenedor realiza healthcheck HTTP contra `http://localhost:15672/api/alarms`.
- En Render configura el healthcheck así:
  - Protocol: HTTP
  - Path: `/api/alarms`
  - Port: `15672`
  - Timeout: `10s`
  - Start period: `60s`
  - Interval: `30s`

## 🐳 Comandos Útiles

### Listar usuarios
```bash
docker exec rabbitmq rabbitmqctl list_users
```

### Listar exchanges
```bash
docker exec rabbitmq rabbitmqctl list_exchanges
```

### Listar queues
```bash
docker exec rabbitmq rabbitmqctl list_queues
```

### Ver logs
```bash
docker logs -f rabbitmq
```

### Reiniciar contenedor
```bash
docker-compose restart
```

### Backup/Restore (topología)

Exportar definiciones:
```powershell
docker exec rabbitmq rabbitmqctl export_definitions /tmp/defs.json
docker cp rabbitmq:/tmp/defs.json .\backup-defs.json
```

Importar definiciones manualmente (si desactivas auto-import):
```powershell
docker cp .\backup-defs.json rabbitmq:/etc/rabbitmq/definitions.json
docker restart rabbitmq
```

## 📦 Imágenes en Docker Hub

```bash
# Usar cualquiera de estos tags
docker pull mrengineer09/rabbitmq:plugins
docker pull mrengineer09/rabbitmq:3.13-plugins
```

**Repositorio**: https://hub.docker.com/r/mrengineer09/rabbitmq

### Build & Push (mantenimiento)

```powershell
docker build -t mrengineer09/rabbitmq:plugins -t mrengineer09/rabbitmq:3.13-plugins .
docker push mrengineer09/rabbitmq:plugins
docker push mrengineer09/rabbitmq:3.13-plugins
```

## 🔄 Actualizar Topología

Para modificar exchanges, queues o bindings:

1. Edita `rabbitmq-definitions.json`
2. Reconstruye la imagen: `docker build -t mrengineer09/rabbitmq:plugins .`
3. Reinicia el contenedor: `docker-compose down && docker-compose up -d`

**Nota**: Las definiciones se cargan automáticamente al iniciar el contenedor.

### Ejemplos de conexión de clientes

Node.js (amqplib):
```js
const amqp = require('amqplib');
(async () => {
  const url = process.env.RABBITMQ_URL || 'amqp://<user>:<pass>@localhost:5672/';
  const conn = await amqp.connect(url);
  const ch = await conn.createChannel();
  await ch.assertQueue('incidentes.q', { durable: true });
  await ch.sendToQueue('incidentes.q', Buffer.from(JSON.stringify({ id: 1 })));
  await ch.close();
  await conn.close();
})();
```

Python (pika):
```python
import os, pika, json
params = pika.URLParameters(os.getenv('RABBITMQ_URL', 'amqp://<user>:<pass>@localhost:5672/'))
conn = pika.BlockingConnection(params)
ch = conn.channel()
ch.queue_declare(queue='incidentes.q', durable=True)
ch.basic_publish('', 'incidentes.q', json.dumps({'id': 1}).encode())
conn.close()
```

Spring Boot (application.yml):
```yaml
spring:
  rabbitmq:
    host: localhost
    port: 5672
    username: ${RABBITMQ_USER}
    password: ${RABBITMQ_PASS}
```

## 📝 Notas

- El usuario `guest` está deshabilitado por seguridad
- Las definiciones se importan automáticamente en el arranque
- Los volúmenes persisten datos de RabbitMQ
- El contenedor reinicia automáticamente a menos que se detenga manualmente

## 🛠️ Solución de Problemas

### Contenedor no inicia
```bash
docker logs rabbitmq
```

### No puedo acceder al Management UI
- Verifica que el puerto 15672 esté disponible
- Usa las credenciales configuradas en tu entorno (no se publican en este README)
- Comprueba que el .env tenga los valores correctos

### Queues/Exchanges no se crean
- Verifica que `rabbitmq-definitions.json` sea válido JSON
- Revisa los logs: `docker logs rabbitmq | grep Importing`

### Errores `{bad_header,<<"HEAD / H">>}`
Esto pasa cuando un healthcheck HTTP intenta conectarse al puerto AMQP (5672). Solución:
- Asegura que los healthchecks apunten al Management UI (HTTP) en el puerto 15672, por ejemplo a `/api/alarms`.
- En Render, usa el healthcheck HTTP descrito en la sección de Healthchecks.

### Autenticación falla
- Verifica que el import de definiciones haya creado el usuario: `docker exec rabbitmq rabbitmqctl list_users`.
- Si modificaste el hash de contraseña en `rabbitmq-definitions.json`, debe ser base64 válido para `rabbit_password_hashing_sha256`.
- Alternativa rápida desde CLI:
  ```powershell
  docker exec rabbitmq rabbitmqctl add_user <user> <pass>
  docker exec rabbitmq rabbitmqctl set_user_tags <user> administrator
  docker exec rabbitmq rabbitmqctl set_permissions -p / <user> ".*" ".*" ".*"
  ```

## ☁️ Despliegue en Render (resumen)

- Imagen: `mrengineer09/rabbitmq:plugins`
- Healthcheck HTTP: Path `/api/alarms`, Port `15672`
- Variables: configura secretos `RABBITMQ_DEFAULT_USER` y `RABBITMQ_DEFAULT_PASS` en Render (no publiques sus valores)
- Disco persistente: `/var/lib/rabbitmq`

Más detalles en `RENDER_DEPLOYMENT.md` y ejemplo de servicio en `render.yaml`.

## 📄 Licencia

RabbitMQ está bajo MPL 2.0 License.

---

**Creado para**: Tesis - Backend Latacunga Clean  
**Fecha**: Noviembre 2025  
**Autor**: Andres09xZ
