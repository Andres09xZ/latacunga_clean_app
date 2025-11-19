# Task Service - API Testing Guide

## Quick Start

All endpoints require Bearer token authentication.

```bash
# Set your token
export TOKEN="your-jwt-token"

# Base URL
export BASE_URL="http://localhost:3003"
```

---

## 1. Get Available Tasks

**Get list of available tasks for a worker at specific location**

```bash
curl -X GET "$BASE_URL/tasks/available?lat=-12.0456&lng=-77.0123" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Expected Response** (200 OK)
```json
{
  "tasks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "novedad_id": "550e8400-e29b-41d4-a716-446655440001",
      "actor_id": null,
      "source": "novedad",
      "type": "ZONA_CRITICA",
      "state": "PENDIENTE",
      "priority": 85,
      "description": "Acumulación de residuos en esquina",
      "latitude": -12.0456789,
      "longitude": -77.0123456,
      "photo_url": null,
      "evidence": null,
      "started_at": null,
      "completed_at": null,
      "created_at": "2025-01-13T10:30:00Z",
      "updated_at": "2025-01-13T10:30:00Z"
    }
  ],
  "count": 1
}
```

---

## 2. Get Task Details

**Retrieve full details of a specific task including history**

```bash
TASK_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X GET "$BASE_URL/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Expected Response** (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "novedad_id": "550e8400-e29b-41d4-a716-446655440001",
  "actor_id": null,
  "source": "novedad",
  "type": "ZONA_CRITICA",
  "state": "PENDIENTE",
  "priority": 85,
  "description": "Acumulación de residuos en esquina",
  "latitude": -12.0456789,
  "longitude": -77.0123456,
  "photo_url": null,
  "evidence": null,
  "started_at": null,
  "completed_at": null,
  "created_at": "2025-01-13T10:30:00Z",
  "updated_at": "2025-01-13T10:30:00Z"
}
```

---

## 3. Claim Task (Start Working)

**Worker claims a task to start working on it**

State transition: `PENDIENTE` → `EN_PROGRESO`

```bash
TASK_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X POST "$BASE_URL/tasks/$TASK_ID/claim" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Expected Response** (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "novedad_id": "550e8400-e29b-41d4-a716-446655440001",
  "actor_id": "550e8400-e29b-41d4-a716-446655440099",
  "source": "novedad",
  "type": "ZONA_CRITICA",
  "state": "EN_PROGRESO",
  "priority": 85,
  "description": "Acumulación de residuos en esquina",
  "latitude": -12.0456789,
  "longitude": -77.0123456,
  "photo_url": null,
  "evidence": null,
  "started_at": "2025-01-13T10:35:00Z",
  "completed_at": null,
  "created_at": "2025-01-13T10:30:00Z",
  "updated_at": "2025-01-13T10:35:00Z"
}
```

**Error Cases**

```bash
# Task already claimed (409 Conflict)
{
  "error": "Task is already claimed or completed (state: EN_PROGRESO)"
}

# User not a worker (403 Forbidden)
{
  "error": "Actor not found"
}

# Task doesn't exist (404 Not Found)
{
  "error": "Task not found"
}
```

---

## 4. Update Task Status (Complete/Cancel)

**Worker marks task as completed with photo evidence**

State transitions:
- `EN_PROGRESO` → `COMPLETADA`
- `EN_PROGRESO` → `CANCELADA`

### 4.1 Mark as Completed

```bash
TASK_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X PUT "$BASE_URL/tasks/$TASK_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "COMPLETADA",
    "evidence": [
      "https://cdn.example.com/photo-1.jpg",
      "https://cdn.example.com/photo-2.jpg"
    ],
    "completed_at": "2025-01-13T16:45:00Z"
  }'
```

**Expected Response** (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "novedad_id": "550e8400-e29b-41d4-a716-446655440001",
  "actor_id": "550e8400-e29b-41d4-a716-446655440099",
  "source": "novedad",
  "type": "ZONA_CRITICA",
  "state": "COMPLETADA",
  "priority": 85,
  "description": "Acumulación de residuos en esquina",
  "evidence": [
    "https://cdn.example.com/photo-1.jpg",
    "https://cdn.example.com/photo-2.jpg"
  ],
  "started_at": "2025-01-13T10:35:00Z",
  "completed_at": "2025-01-13T16:45:00Z",
  "updated_at": "2025-01-13T16:45:00Z"
}
```

### 4.2 Mark as Cancelled (with reason)

```bash
TASK_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X PUT "$BASE_URL/tasks/$TASK_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "CANCELADA",
    "reason": "Ubicación inaccesible debido a condiciones climáticas"
  }'
```

**Expected Response** (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "state": "CANCELADA",
  "reason": "Ubicación inaccesible debido a condiciones climáticas",
  "updated_at": "2025-01-13T16:50:00Z"
}
```

---

## 5. List Tasks with Filtering

**Get paginated list of tasks with optional filters**

```bash
# List all tasks in progress for this worker
curl -X GET "$BASE_URL/tasks?state=EN_PROGRESO&actor_id=550e8400-e29b-41d4-a716-446655440099&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

# List all completed novedades
curl -X GET "$BASE_URL/tasks?source=novedad&state=COMPLETADA&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

# List pending tasks (default sorting by priority)
curl -X GET "$BASE_URL/tasks?state=PENDIENTE&limit=50" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Expected Response** (200 OK)
```json
{
  "tasks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "novedad_id": "550e8400-e29b-41d4-a716-446655440001",
      "source": "novedad",
      "type": "ZONA_CRITICA",
      "state": "EN_PROGRESO",
      "priority": 85
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "novedad_id": "550e8400-e29b-41d4-a716-446655440003",
      "source": "novedad",
      "type": "PUNTO_ACOPIO",
      "state": "EN_PROGRESO",
      "priority": 70
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 2
  }
}
```

---

## Error Response Reference

### 400 Bad Request
```json
{
  "error": "Invalid latitude"
}
```

### 404 Not Found
```json
{
  "error": "Task not found"
}
```

### 409 Conflict
```json
{
  "error": "Invalid state transition from COMPLETADA to EN_PROGRESO"
}
```

### 403 Forbidden
```json
{
  "error": "Not authorized to update this task"
}
```

### 500 Internal Server Error
```json
{
  "error": "Database error"
}
```

---

## Testing Scenarios

### Scenario 1: Complete Happy Path

```bash
#!/bin/bash

BASE_URL="http://localhost:3003"
TOKEN="your-jwt-token"

# 1. Get available tasks
echo "1. Fetching available tasks..."
RESPONSE=$(curl -s -X GET "$BASE_URL/tasks/available?lat=-12.0456&lng=-77.0123" \
  -H "Authorization: Bearer $TOKEN")
TASK_ID=$(echo $RESPONSE | jq -r '.tasks[0].id')
echo "Task ID: $TASK_ID"

# 2. Claim the task
echo "2. Claiming task..."
curl -s -X POST "$BASE_URL/tasks/$TASK_ID/claim" \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# 3. Check task is now EN_PROGRESO
echo "3. Getting task details..."
curl -s -X GET "$BASE_URL/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.state'

# 4. Complete the task
echo "4. Completing task with evidence..."
curl -s -X PUT "$BASE_URL/tasks/$TASK_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "COMPLETADA",
    "evidence": ["https://example.com/photo.jpg"]
  }' | jq '.'

echo "✓ Task completed successfully"
```

### Scenario 2: Concurrency Test (Two workers claim same task)

```bash
#!/bin/bash

TASK_ID="550e8400-e29b-41d4-a716-446655440000"
BASE_URL="http://localhost:3003"
TOKEN1="worker1-token"
TOKEN2="worker2-token"

echo "Worker 1 claims task..."
RESULT1=$(curl -s -X POST "$BASE_URL/tasks/$TASK_ID/claim" \
  -H "Authorization: Bearer $TOKEN1")
echo $RESULT1 | jq '.state'  # Should be EN_PROGRESO

echo "Worker 2 tries to claim same task..."
RESULT2=$(curl -s -X POST "$BASE_URL/tasks/$TASK_ID/claim" \
  -H "Authorization: Bearer $TOKEN2")
echo $RESULT2 | jq '.error'  # Should be Conflict
```

### Scenario 3: Invalid State Transition

```bash
#!/bin/bash

TASK_ID="550e8400-e29b-41d4-a716-446655440000"
BASE_URL="http://localhost:3003"
TOKEN="your-jwt-token"

# First complete the task
curl -s -X PUT "$BASE_URL/tasks/$TASK_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"state": "COMPLETADA"}'

# Try to move it back to EN_PROGRESO (should fail)
echo "Attempting invalid transition (COMPLETADA → EN_PROGRESO)..."
curl -s -X PUT "$BASE_URL/tasks/$TASK_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"state": "EN_PROGRESO"}' | jq '.error'
# Expected: "Invalid state transition from COMPLETADA to EN_PROGRESO"
```

---

## SQL Testing Queries

### Check created tasks
```sql
SELECT id, source, type, state, priority, created_at
FROM tasks
ORDER BY created_at DESC
LIMIT 10;
```

### Check task assigned to specific worker
```sql
SELECT t.id, t.state, t.started_at, t.completed_at, a.user_id
FROM tasks t
LEFT JOIN actors a ON t.actor_id = a.id
WHERE a.user_id = '550e8400-e29b-41d4-a716-446655440099'::uuid;
```

### Check task history (audit trail)
```sql
SELECT old_state, new_state, reason, created_at
FROM task_histories
WHERE task_id = '550e8400-e29b-41d4-a716-446655440000'::uuid
ORDER BY created_at DESC;
```

### Check processed events (idempotency)
```sql
SELECT event_id, consumer, event_type, source_id, processed_at
FROM processed_events
WHERE consumer = 'task-service'
ORDER BY processed_at DESC;
```

### Check for duplicate event processing
```sql
SELECT event_id, COUNT(*) as count
FROM processed_events
WHERE consumer = 'task-service'
GROUP BY event_id
HAVING COUNT(*) > 1;
-- Should return no rows (meaning no duplicates)
```

---

## Performance Testing

### Load Test: Create 1000 available tasks

```bash
#!/bin/bash

echo "Creating 1000 tasks for testing..."
for i in {1..1000}; do
  psql -c "INSERT INTO tasks (novedad_id, source, type, state, priority, description, created_at, updated_at) 
           VALUES (gen_random_uuid(), 'novedad', 'ZONA_CRITICA', 'PENDIENTE', $((RANDOM % 100)), 'Test task $i', NOW(), NOW())"
done

echo "✓ Created 1000 tasks"

# Now test query performance
echo "Testing query performance..."
time psql -c "SELECT id, type, priority FROM tasks WHERE state = 'PENDIENTE' ORDER BY priority DESC LIMIT 50"
```

### Concurrency Test: 50 workers claim tasks simultaneously

```bash
#!/bin/bash

BASE_URL="http://localhost:3003"

for worker_num in {1..50}; do
  (
    TOKEN=$(curl -s -X POST "http://localhost:3001/api/v1/auth/login" \
      -d "{\"email\": \"worker$worker_num@example.com\", \"password\": \"password\"}" | jq -r '.token')
    
    curl -s -X GET "$BASE_URL/tasks/available?lat=-12.0456&lng=-77.0123" \
      -H "Authorization: Bearer $TOKEN" | jq -r '.tasks[0].id' | \
    xargs -I {} curl -X POST "$BASE_URL/tasks/{}/claim" \
      -H "Authorization: Bearer $TOKEN"
  ) &
done

wait
echo "✓ Concurrency test completed"
```

---

