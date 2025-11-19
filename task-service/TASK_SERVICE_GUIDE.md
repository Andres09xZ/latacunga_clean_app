# Task Service - Comprehensive Guide

## Overview

The **Task Service** is responsible for converting novedades (citizen reports) into tasks that can be claimed and completed by workers/operators. It handles:

- **Event Processing**: Subscribes to `novedad.creada` and `novedad.verificada` events from the Novedades service
- **Task Management**: CRUD operations for tasks with state machine (PENDIENTE → EN_PROGRESO → COMPLETADA/CANCELADA)
- **Worker Assignment**: Workers can view available tasks and claim them
- **Idempotency**: Processes events only once using `ProcessedEvent` table
- **Audit Trail**: Records all state changes in `TaskHistory`
- **Integration**: Publishes task events back to the system (tarea_creada, tarea_asignada, tarea_completada, tarea_cancelada)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     RabbitMQ Events                             │
└──────────────────────┬──────────────────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
        ▼                             ▼
  novedad.creada              novedad.verificada
        │                             │
        └──────────────┬──────────────┘
                       │
                       ▼
        ┌─────────────────────────────┐
        │   Event Consumer            │
        │  (consumer.go)              │
        └──────────────┬──────────────┘
                       │
                       ▼
        ┌─────────────────────────────────────────┐
        │    Handler Layer                        │
        │  HandleNovedadEvent()                   │
        │  HandleReportVerifiedEvent()            │
        └──────────────┬──────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
        ▼                             ▼
    ┌───────────────────┐      ┌────────────────────┐
    │  Create Task      │      │ Record Idempotency │
    │  Database         │      │ ProcessedEvent     │
    └───────────────────┘      └────────────────────┘
        │
        ▼
    ┌────────────────────────────────┐
    │  Publish Events                │
    │  - tarea_creada                │
    │  - tarea_asignada              │
    │  - tarea_completada            │
    │  - novedad_atendida            │
    └────────────────────────────────┘
```

---

## Data Models

### Task States (State Machine)

```
┌──────────────────────────────────────────────────────────────┐
│                    PENDIENTE                                 │
│  (New task, waiting to be claimed)                          │
└─────────┬──────────────────────────────────────────┬────────┘
          │                                          │
          ▼ (Worker claims)                          ▼ (Task cancelled)
    ┌──────────────────┐                       ┌──────────────┐
    │  EN_PROGRESO     │                       │  CANCELADA   │
    │  (Task assigned) │                       │  (Terminal)  │
    └────────┬─────────┘                       └──────────────┘
             │
             │ (Worker marks complete)
             ▼
    ┌──────────────────────┐
    │  COMPLETADA          │
    │  (Task finished)     │
    │  (Terminal)          │
    └──────────────────────┘
```

### Task Structure

```go
type Task struct {
    // Identification
    ID          uuid.UUID  // Primary key
    NovedadID   *uuid.UUID // Foreign key to novedad (if source="novedad")
    ReportID    *uuid.UUID // Foreign key to report (if source="report")
    
    // Assignment
    ActorID     *uuid.UUID // Worker/Operator who claimed the task
    Source      string     // "novedad" or "report"
    Type        string     // PUNTO_ACOPIO, ZONA_CRITICA, LIMPIEZA, RECOLECCION
    
    // Status & Priority
    State       string     // PENDIENTE, EN_PROGRESO, COMPLETADA, CANCELADA
    Priority    int        // 0-100, higher = more urgent
    
    // Task Details
    Description string
    Instructions string
    
    // Location
    Latitude    *float64
    Longitude   *float64
    
    // Evidence
    PhotoURL    *string    // Photo taken during task
    Evidence    []string   // Array of photo URLs for completed tasks
    
    // Timing
    StartedAt   *time.Time // When worker started
    CompletedAt *time.Time // When task was completed
    
    // Version Control
    Version     int        // For optimistic locking (future)
    
    // Metadata
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Histories   []TaskHistory
}

type TaskHistory struct {
    ID        uuid.UUID
    TaskID    uuid.UUID  // Foreign key
    ActorID   uuid.UUID  // Who made the change
    OldState  string
    NewState  string
    Reason    *string
    CreatedAt time.Time
}

type ProcessedEvent struct {
    EventID     uuid.UUID // From RabbitMQ
    Consumer    string    // "task-service"
    EventType   string    // "novedad" or "report"
    SourceID    uuid.UUID // novedad_id or report_id
    ProcessedAt time.Time
}
```

---

## API Endpoints

### 1. Get Available Tasks

**Request**
```http
GET /tasks/available?lat=-12.33&lng=-77.00
Authorization: Bearer <token>
```

**Response**
```json
{
  "tasks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "novedad_id": "550e8400-e29b-41d4-a716-446655440001",
      "source": "novedad",
      "type": "ZONA_CRITICA",
      "state": "PENDIENTE",
      "priority": 85,
      "description": "Zona con exceso de residuos en av. principal",
      "latitude": -12.3456,
      "longitude": -77.0123,
      "created_at": "2025-01-13T10:30:00Z",
      "updated_at": "2025-01-13T10:30:00Z"
    }
  ],
  "count": 1
}
```

### 2. Get Task Details

**Request**
```http
GET /tasks/{taskId}
Authorization: Bearer <token>
```

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "novedad_id": "550e8400-e29b-41d4-a716-446655440001",
  "actor_id": null,
  "source": "novedad",
  "type": "ZONA_CRITICA",
  "state": "PENDIENTE",
  "priority": 85,
  "description": "Zona con exceso de residuos",
  "latitude": -12.3456,
  "longitude": -77.0123,
  "created_at": "2025-01-13T10:30:00Z",
  "updated_at": "2025-01-13T10:30:00Z",
  "histories": []
}
```

### 3. Claim Task

**Request**
```http
POST /tasks/{taskId}/claim
Authorization: Bearer <token>
Content-Type: application/json
```

**Response** (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "actor_id": "550e8400-e29b-41d4-a716-446655440099",
  "state": "EN_PROGRESO",
  "started_at": "2025-01-13T10:35:00Z",
  "updated_at": "2025-01-13T10:35:00Z"
}
```

**Errors**
- `400 Bad Request`: Invalid task ID
- `404 Not Found`: Task not found
- `409 Conflict`: Task already claimed (state != PENDIENTE/PENDIENTE_ASIGNAR)
- `403 Forbidden`: User not a worker

### 4. Update Task Status

**Request**
```http
PUT /tasks/{taskId}/status
Authorization: Bearer <token>
Content-Type: application/json

{
  "state": "COMPLETADA",
  "evidence": ["https://cdn.example.com/photo1.jpg", "https://cdn.example.com/photo2.jpg"],
  "reason": null,
  "completed_at": "2025-01-13T16:45:00Z"
}
```

**Valid Transitions**
```
PENDIENTE/PENDIENTE_ASIGNAR → EN_PROGRESO
PENDIENTE/PENDIENTE_ASIGNAR → CANCELADA
EN_PROGRESO → COMPLETADA
EN_PROGRESO → CANCELADA
COMPLETADA → (terminal, no transitions)
CANCELADA → (terminal, no transitions)
```

**Response** (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "state": "COMPLETADA",
  "evidence": ["https://cdn.example.com/photo1.jpg"],
  "completed_at": "2025-01-13T16:45:00Z",
  "updated_at": "2025-01-13T16:45:00Z"
}
```

**Errors**
- `400 Bad Request`: Invalid state value
- `404 Not Found`: Task not found
- `409 Conflict`: Invalid state transition
- `403 Forbidden`: User not authorized (not the assigned worker)

### 5. List Tasks (Paginated, Filterable)

**Request**
```http
GET /tasks?state=EN_PROGRESO&source=novedad&actor_id=<uuid>&page=1&limit=20
Authorization: Bearer <token>
```

**Response**
```json
{
  "tasks": [ /* array of task objects */ ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45
  }
}
```

---

## Event Flow

### Scenario 1: Creating a Task from a Novedad

```
┌─────────────────────────────────────────────────────────────────┐
│  1. Citizen creates novedad on mobile app (offline)             │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│  2. Novedad verified by operator                                │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│  3. Novedades Service publishes event:                          │
│     Topic: "novedad.verificada"                                 │
│     Payload: {                                                  │
│       id: UUID,                                                 │
│       novedad_id: UUID,                                         │
│       type: "ZONA_CRITICA",                                     │
│       description: "...",                                       │
│       location: {latitude: -12.34, longitude: -77.01},          │
│       priority: 85,                                             │
│       timestamp: "2025-01-13T10:30:00Z"                         │
│     }                                                           │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│  4. Task Service Event Consumer receives event                  │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│  5. Check idempotency:                                          │
│     SELECT * FROM processed_events                              │
│     WHERE event_id = UUID AND consumer = "task-service"         │
│     AND event_type = "novedad"                                  │
│     IF already processed: SKIP                                  │
│     ELSE: Continue                                              │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│  6. Create Task in database:                                    │
│     INSERT INTO tasks (                                         │
│       novedad_id, source, type, state,                          │
│       description, latitude, longitude, priority                │
│     ) VALUES (...)                                              │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│  7. Record processed event for idempotency:                     │
│     INSERT INTO processed_events (                              │
│       event_id, consumer, event_type, source_id, processed_at   │
│     ) VALUES (UUID, "task-service", "novedad", novedad_id, now) │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│  8. Publish event: "tarea_creada"                               │
│     Topic: "tasks"                                              │
│     RoutingKey: "task.creada"                                   │
│     Payload: {                                                  │
│       task_id: UUID,                                            │
│       novedad_id: UUID,                                         │
│       source: "novedad",                                        │
│       type: "ZONA_CRITICA",                                     │
│       state: "PENDIENTE",                                       │
│       timestamp: now                                            │
│     }                                                           │
└─────────────────────────────────────────────────────────────────┘
```

### Scenario 2: Worker Claiming and Completing a Task

```
┌──────────────────────────────────────────────────────────────────┐
│  1. Worker calls GET /tasks/available?lat=...&lng=...            │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  2. Task Service returns PENDIENTE tasks ordered by priority     │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  3. Worker selects a task and calls POST /tasks/{taskId}/claim   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  4. Task Service updates task:                                   │
│     OLD STATE: PENDIENTE                                        │
│     NEW STATE: EN_PROGRESO                                      │
│     SET actor_id = worker_uuid, started_at = now                │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  5. Record in TaskHistory:                                       │
│     old_state: PENDIENTE                                        │
│     new_state: EN_PROGRESO                                      │
│     actor_id: worker_uuid                                       │
│     reason: "Tarea reclamada por trabajador"                    │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  6. Publish event: "tarea_asignada"                              │
│     Payload: {task_id, source, type, state, actor_id, ...}      │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  7. Worker completes task with photos                           │
│     PUT /tasks/{taskId}/status                                  │
│     {state: "COMPLETADA", evidence: [...urls...]}               │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  8. Task Service updates task:                                   │
│     NEW STATE: COMPLETADA                                       │
│     SET evidence = [...], completed_at = now                    │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│  9. Record in TaskHistory and emit two events:                   │
│     - "tarea_completada"                                        │
│     - "novedad_atendida" (sent back to Novedades Service)       │
│       Payload: {novedad_id, timestamp}                          │
└──────────────────────────────────────────────────────────────────┘
```

---

## Event Message Examples

### Incoming: `novedad.verificada`

```json
{
  "id": "e1234567-890a-bcde-f012-345678900000",
  "novedad_id": "n1234567-890a-bcde-f012-345678900001",
  "type": "ZONA_CRITICA",
  "reporter_kind": "ciudadano",
  "description": "Acumulación de residuos en la esquina de la avenida",
  "location": {
    "latitude": -12.0456789,
    "longitude": -77.0123456
  },
  "priority": 85,
  "status": "verificada",
  "verified_at": "2025-01-13T10:30:00Z",
  "timestamp": "2025-01-13T10:30:00Z"
}
```

### Outgoing: `task.creada`

```json
{
  "task_id": "t9876543-210f-edcb-a098-765432100000",
  "source": "novedad",
  "type": "ZONA_CRITICA",
  "state": "PENDIENTE",
  "novedad_id": "n1234567-890a-bcde-f012-345678900001",
  "priority": 85,
  "timestamp": "2025-01-13T10:30:01Z"
}
```

### Outgoing: `novedad.atendida`

```json
{
  "novedad_id": "n1234567-890a-bcde-f012-345678900001",
  "task_id": "t9876543-210f-edcb-a098-765432100000",
  "status": "ATENDIDA",
  "completed_at": "2025-01-13T16:45:00Z",
  "timestamp": "2025-01-13T16:45:00Z"
}
```

---

## Key Features

### 1. Idempotency
Every event is processed only once using the `ProcessedEvent` table:
- Event ID uniquely identifies the message
- Consumer + Event Type + Source ID combination prevents duplicates
- If a message is received twice, the second one is skipped
- Ensures consistency even with RabbitMQ redeliveries

### 2. State Machine Validation
- Only valid transitions are allowed
- Attempting invalid transition returns HTTP 409 Conflict
- Examples:
  - Can move from PENDIENTE to EN_PROGRESO ✓
  - Can move from PENDIENTE to CANCELADA ✓
  - Can move from COMPLETADA to EN_PROGRESO ✗ (Invalid)

### 3. Audit Trail
- Every state change is recorded in `TaskHistory`
- Tracks: who changed it (ActorID), when, and why
- Enables debugging and compliance reporting

### 4. Multi-Source Support
- Tasks can come from `novedad` (citizen reports)
- Or from `report` (legacy system)
- Source field disambiguates and enables proper integration

### 5. Event Publishing
- When a task is completed, system publishes `novedad_atendida` back to Novedades Service
- Allows Novedades Service to update novedad status to "ATENDIDA"
- Closes the integration loop

---

## SQL Queries for Testing

### Create a task manually
```sql
INSERT INTO tasks (
  id, novedad_id, source, type, state, priority,
  description, latitude, longitude, created_at, updated_at
) VALUES (
  gen_random_uuid(),
  '550e8400-e29b-41d4-a716-446655440001'::uuid,
  'novedad',
  'ZONA_CRITICA',
  'PENDIENTE',
  85,
  'Test novedad task',
  -12.0456789,
  -77.0123456,
  NOW(),
  NOW()
);
```

### Query available tasks
```sql
SELECT id, type, state, priority, created_at
FROM tasks
WHERE state = 'PENDIENTE'
ORDER BY priority DESC, created_at ASC
LIMIT 50;
```

### Get task history
```sql
SELECT h.old_state, h.new_state, h.reason, h.created_at
FROM task_histories h
WHERE h.task_id = '550e8400-e29b-41d4-a716-446655440000'::uuid
ORDER BY h.created_at DESC;
```

### Check processed events (idempotency)
```sql
SELECT * FROM processed_events
WHERE consumer = 'task-service'
AND event_type = 'novedad'
ORDER BY processed_at DESC;
```

### Get tasks claimed by a specific worker
```sql
SELECT t.*, a.user_id
FROM tasks t
LEFT JOIN actors a ON t.actor_id = a.id
WHERE a.user_id = '<worker_uuid>'
AND t.state = 'EN_PROGRESO';
```

---

## Integration with Novedades Service

### Workflow
1. Citizen creates/updates novedad → Novedades service saves it
2. Operator verifies novedad → Novedades service publishes `novedad.verificada`
3. **Task Service receives event** → Creates task in PENDIENTE state
4. **Worker claims task** → Task state → EN_PROGRESO
5. **Worker completes task with photos** → Task state → COMPLETADA
6. **Task Service publishes** `novedad.atendida` → Novedades service receives it
7. Novedades service updates novedad status to "ATENDIDA" ✓

### Expected Latencies
- Novedad verification → Task creation: **< 5 seconds** (p95)
- Worker claim → Task assignment: **< 1 second**
- Worker completion → Novedad update: **< 5 seconds** (end-to-end)

---

## Database Schema

### tasks table
```sql
CREATE TABLE tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  novedad_id UUID REFERENCES novedades(id),
  report_id UUID,
  actor_id UUID,
  source VARCHAR(20) NOT NULL DEFAULT 'novedad', -- 'novedad' or 'report'
  type VARCHAR(50) NOT NULL DEFAULT 'LIMPIEZA',
  state VARCHAR(50) NOT NULL DEFAULT 'PENDIENTE',
  priority INT DEFAULT 0,
  instructions TEXT,
  description TEXT,
  latitude NUMERIC(10, 8),
  longitude NUMERIC(11, 8),
  photo_url TEXT,
  evidence JSONB DEFAULT '[]'::jsonb,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  version INT DEFAULT 1,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  INDEX idx_state (state),
  INDEX idx_actor_id (actor_id),
  INDEX idx_novedad_id (novedad_id)
);

CREATE TABLE task_histories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  actor_id UUID NOT NULL,
  old_state VARCHAR(50) NOT NULL,
  new_state VARCHAR(50) NOT NULL,
  reason TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  
  INDEX idx_task_id (task_id),
  INDEX idx_created_at (created_at)
);

CREATE TABLE processed_events (
  event_id UUID PRIMARY KEY,
  consumer VARCHAR(50) NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  source_id UUID NOT NULL,
  processed_at TIMESTAMP DEFAULT NOW(),
  
  UNIQUE (event_id, consumer, event_type),
  INDEX idx_consumer (consumer),
  INDEX idx_event_type (event_type),
  INDEX idx_processed_at (processed_at)
);
```

---

## Configuration

### Environment Variables

```bash
# Database
DB_URL=postgres://user:password@localhost:5432/latacunga_clean

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Server
PORT=3003
GIN_MODE=debug  # or "release"
```

---

## Testing Checklist

- [ ] **Event Idempotency**: Send same event twice, verify only one task created
- [ ] **State Transitions**: Test all valid transitions work, invalid ones fail
- [ ] **Authorization**: Only assigned worker can update task status
- [ ] **Concurrency**: Two workers try to claim same task simultaneously (one fails)
- [ ] **Event Publishing**: Task completion triggers `novedad_atendida` event
- [ ] **Database Recovery**: Restart service, events still process (using durable queues)
- [ ] **Latency**: Measure time from novedad.verificada → task.creada (should be < 5 seconds)

---

## Future Improvements

1. **Optimistic Locking**: Use `version` field to prevent race conditions
2. **Task Reassignment**: If worker doesn't complete task within timeout, reassign
3. **Location-Based Matching**: Use PostGIS to find workers closest to task location
4. **Task Priorities**: Recalculate priority based on time, location, worker availability
5. **Batch Operations**: Support creating multiple tasks from batch novedades
6. **Webhooks**: Allow external systems to subscribe to task state changes
7. **Analytics**: Track metrics like avg completion time, worker efficiency, etc.

---

## Running the Service

```bash
# Install dependencies
go mod download

# Run migrations (auto-migrate handled by GORM)
# Just ensure DB_URL is set

# Start service
go run ./cmd/server/main.go

# Service will:
# 1. Initialize database connection
# 2. Auto-migrate models
# 3. Start RabbitMQ consumer (subscribing to novedades + reports exchanges)
# 4. Listen on :3003 for HTTP requests
# 5. Log all events to stdout
```

---

