# 🐰 RabbitMQ Event Streaming - Incident Service

El incident-service ahora publica eventos automáticamente a **RabbitMQ** cuando se crean o actualizan incidentes.

## 📋 Configuración

### Variables de Entorno

```env
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

### Estructura de Eventos

**Exchange**: `incidentes` (type: `topic`)

**Routing Keys**:
- `incidente.incidente_creado` - Cuando se crea un nuevo incidente
- `incidente.estado_actualizado` - Cuando cambia el estado
- `incidente.attachment_added` - Cuando se agrega una foto/archivo

## 🔄 Eventos Publicados

### 1. Incidente Creado

**Event Type**: `incidente_creado`

**Routing Key**: `incidente.incidente_creado`

**Payload**:
```json
{
  "id": "f44d56fc-c889-4271-b2b0-2dac928fb429",
  "reporter_kind": "ciudadano",
  "reporter_id": "79587662-defd-4ca2-bb25-9313aca1a43e",
  "type": "animal_muerto",
  "status": "emitido",
  "incident_day": "2025-11-14T00:00:00Z",
  "photos_count": 0,
  "created_at": "2025-11-13T19:20:27Z",
  "event_type": "incidente_creado",
  "timestamp": "2025-11-13T19:20:27Z"
}
```

### 2. Estado Actualizado

**Event Type**: `estado_actualizado`

**Routing Key**: `incidente.estado_actualizado`

**Payload**:
```json
{
  "id": "f44d56fc-c889-4271-b2b0-2dac928fb429",
  "reporter_kind": "ciudadano",
  "type": "animal_muerto",
  "status": "valido",
  "incident_day": "2025-11-14T00:00:00Z",
  "photos_count": 1,
  "created_at": "2025-11-13T19:20:27Z",
  "event_type": "estado_actualizado",
  "timestamp": "2025-11-13T19:25:00Z"
}
```

### 3. Foto/Archivo Agregado

**Event Type**: `attachment_added`

**Routing Key**: `incidente.attachment_added`

**Payload**:
```json
{
  "id": "f44d56fc-c889-4271-b2b0-2dac928fb429",
  "reporter_kind": "ciudadano",
  "type": "animal_muerto",
  "status": "valido",
  "incident_day": "2025-11-14T00:00:00Z",
  "photos_count": 2,
  "created_at": "2025-11-13T19:20:27Z",
  "event_type": "attachment_added",
  "timestamp": "2025-11-13T19:25:30Z"
}
```

## 🎯 Casos de Uso

### 1. Escuchar todos los eventos de incidentes

```bash
# En PowerShell o Terminal de RabbitMQ
# Conectarse a RabbitMQ y crear una cola anónima
queue_name=$(rabbitmqctl list_queues -p / | head -1)

# Vincular a todos los eventos
rabbitmqctl bind_exchange incidentes amq.topic "incidente.*"
```

### 2. Escuchar solo incidentes creados

**Binding**: `incidente.incidente_creado`

### 3. Escuchar solo cambios de estado

**Binding**: `incidente.estado_actualizado`

## 💻 Ejemplo: Consumer en Go

```go
package main

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declarar exchange
	err = ch.ExchangeDeclare("incidentes", amqp.ExchangeTopic, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	// Crear cola anónima
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Vincular cola a routing key
	err = ch.QueueBind(q.Name, "incidente.incidente_creado", "incidentes", false, nil)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	// Consumir mensajes
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to consume: %v", err)
	}

	log.Println("Listening for incident creation events...")
	for msg := range msgs {
		var event map[string]interface{}
		json.Unmarshal(msg.Body, &event)
		log.Printf("Event received: %+v", event)
	}
}
```

## 💻 Ejemplo: Consumer en Python

```python
import pika
import json

connection = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
channel = connection.channel()

# Declarar exchange
channel.exchange_declare(exchange='incidentes', exchange_type='topic', durable=True)

# Crear cola
queue = channel.queue_declare(queue='', exclusive=True)

# Vincular cola
channel.queue_bind(exchange='incidentes', queue=queue.method.queue, routing_key='incidente.incidente_creado')

# Callback
def on_message(ch, method, properties, body):
    event = json.loads(body)
    print(f"Event received: {event}")

channel.basic_consume(queue=queue.method.queue, on_message_callback=on_message)

print("Listening for incident creation events...")
channel.start_consuming()
```

## 💻 Ejemplo: Consumer en Node.js

```javascript
const amqplib = require('amqplib');

async function consume() {
  try {
    const connection = await amqplib.connect('amqp://guest:guest@localhost:5672/');
    const channel = await connection.createChannel();

    // Declarar exchange
    await channel.assertExchange('incidentes', 'topic', { durable: true });

    // Crear cola
    const { queue } = await channel.assertQueue('', { exclusive: true });

    // Vincular cola
    await channel.bindQueue(queue, 'incidentes', 'incidente.incidente_creado');

    // Consumir
    console.log('Listening for incident creation events...');
    await channel.consume(queue, (msg) => {
      const event = JSON.parse(msg.content.toString());
      console.log('Event received:', event);
      channel.ack(msg);
    });
  } catch (error) {
    console.error('Error:', error);
  }
}

consume();
```

## 🚀 Características

✅ **Publicación automática** - Los eventos se publican sin intervención manual
✅ **Topic-based routing** - Usa routing keys para filtrar eventos
✅ **Reutilización de conexión** - Una sola conexión para todos los eventos
✅ **Manejo de errores** - Los errores de RabbitMQ no afectan la operación del servicio
✅ **Escalable** - Soporta múltiples consumers simultáneos

## 📊 Flujo Completo

```
1. Cliente crea incidente (POST /api/v1/incidents)
   ↓
2. Incident-service procesa la solicitud
   ↓
3. Se guarda en BD (PostgreSQL)
   ↓
4. Se publica evento "incidente_creado" a RabbitMQ
   ↓
5. Todos los consumers escuchando ese evento lo reciben
   ↓
6. Otros servicios (notificaciones, reportes, etc.) procesan el evento
```

## 🔧 Troubleshooting

### RabbitMQ no está conectado
```
⚠️ Warning: RabbitMQ not connected, skipping event publication
```
**Solución**: Verifica que RabbitMQ está corriendo en `amqp://guest:guest@localhost:5672/`

### El evento no se recibe
1. Verifica que la cola está vinculada al routing key correcto
2. Verifica que estás escuchando en el exchange `incidentes`
3. Verifica los logs del incident-service

### Múltiples consumers no reciben el mismo evento
**Nota**: Por defecto en RabbitMQ, cuando múltiples consumers están en la misma cola, **se distribuyen los mensajes** entre ellos. Para que todos reciban el mismo evento, cada consumer debe tener su propia cola.

## 📚 Referencias

- [RabbitMQ Topic Exchange](https://www.rabbitmq.com/tutorials/amqp-concepts.html#topic-exchange)
- [Go AMQP Library](https://github.com/rabbitmq/amqp091-go)
- [RabbitMQ Management](http://localhost:15672/) - User: guest / Pass: guest
