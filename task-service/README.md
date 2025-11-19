# Task Service

Microservicio para asignación y gestión de tareas de limpieza de zonas críticas.

## Endpoints

- GET /api/v1/tasks/available?actorType=WORKER&lat=...&lng=... - Lista reportes disponibles
- POST /api/v1/tasks/{reportId}/claim - Reclamar tarea
- PUT /api/v1/tasks/{taskId}/complete - Completar tarea

## Environment Variables

- DATABASE_URL
- JWT_SECRET
- RABBITMQ_URL

## Run

go run cmd/server/main.go