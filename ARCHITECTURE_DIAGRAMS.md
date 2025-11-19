# Architecture Diagrams - Latacunga Clean App

## 🏗️ Microservices Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        API Gateway / Load Balancer                  │
│                              (Port 8080)                            │
└──────────────────────────────────────────────────────────────────────┘
                              ▲  │  ▼
                ┌─────────────┘  │  └─────────────┐
                │                │                │
        ┌───────▼────────┐  ┌──────▼────────┐  ┌─▼────────────────┐
        │  Auth Service  │  │ Report Service │ │  Task Service    │
        │   (Port 8080)  │  │  (Port 8081)   │  │  (Port 8082)     │
        └────────────────┘  └────────────────┘  └──────────────────┘
              │                    │                     │
              │                    │                     │
         [PostgreSQL]          [PostgreSQL]         [PostgreSQL]
         (usuario.*)          (reportes.*)          (tareas.*)
```

## 👥 Auth Service - Dual Authentication Flow

```
                    ┌─────────────────────────────────┐
                    │   Client / Mobile App           │
                    └──────────────┬────────────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │                             │
              ┌─────▼──────┐            ┌────────▼──────┐
              │ Ciudadano   │            │  Operador     │
              │ (Sin Admin) │            │ (Interno)     │
              └─────┬──────┘            └────────┬───────┘
                    │                           │
         ┌──────────▼──────────┐      ┌────────▼────────┐
         │   OTP Via SMS       │      │ Email/Password  │
         │   (Twilio)          │      │                 │
         └──────────┬──────────┘      └────────┬────────┘
                    │                         │
         ┌──────────▼──────────┐      ┌────────▼────────┐
         │  POST /otp/send     │      │POST /register   │
         │  POST /otp/verify   │      │POST /login      │
         └──────────┬──────────┘      └────────┬────────┘
                    │                         │
                    └──────────┬──────────────┘
                               │
         ┌─────────────────────▼──────────────────────┐
         │         JWT Token Generation              │
         │    (HS256, 8 horas de expiración)         │
         └──────────┬────────────────────┬───────────┘
                    │                    │
      ┌─────────────▼─┐      ┌──────────▼─────────┐
      │  Scope:       │      │  Scope:            │
      │ novedades:    │      │ tareas:gestionar   │
      │    crear      │      │ reportes:*         │
      └───────────────┘      └────────────────────┘
```

## 🗄️ Database Schema - usuario (Auth Service)

```
┌─────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                          │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Schema: usuario                                         │   │
│  │                                                         │   │
│  │  ┌─────────────────┐                                    │   │
│  │  │   operators     │                                    │   │
│  │  ├─────────────────┤                                    │   │
│  │  │ id (UUID)       │◄──────────┐                        │   │
│  │  │ name            │           │                        │   │
│  │  │ email (UNIQUE)  │           │                        │   │
│  │  │ password_hash   │           │ 0..1               1:N │   │
│  │  │ role (ENUM)     │           │                        │   │
│  │  │ active          │           │                        │   │
│  │  │ created_at      │           │                        │   │
│  │  └─────────────────┘           │                        │   │
│  │                                │                        │   │
│  │  ┌──────────────────┐          │    ┌──────────────────┐  │
│  │  │   citizens       │          │    │ outbox_events    │  │
│  │  ├──────────────────┤          │    ├──────────────────┤  │
│  │  │ id (UUID)        │          │    │ id (UUID)        │  │
│  │  │ phone_e164       │          │    │ aggregate_type   │  │
│  │  │ (UNIQUE)         │          │    │ aggregate_id ─────┼──┤
│  │  │ verified_at      │          │    │ type             │  │
│  │  │ created_at       │          │    │ payload (JSONB)  │  │
│  │  └──────────────────┘          │    │ status           │  │
│  │           ▲                    │    │ created_at       │  │
│  │           │                    │    │ published_at     │  │
│  │           │ 1:N                │    └──────────────────┘  │
│  │           │                    │                          │
│  │  ┌────────┴───────────┐        │                          │
│  │  │  otp_requests      │        │                          │
│  │  ├────────────────────┤        │                          │
│  │  │ id (UUID)          │        │                          │
│  │  │ phone_e164 (FK)────┼────────┘                          │
│  │  │ status (ENUM)      │                                   │
│  │  │ attempts           │                                   │
│  │  │ requested_at       │                                   │
│  │  │ verified_at        │                                   │
│  │  │ error_code         │                                   │
│  │  └────────────────────┘                                   │
│  │                                                         │   │
│  │  ┌────────────────────────────────────────┐            │   │
│  │  │  idempotency_keys (Deduplication)      │            │   │
│  │  ├────────────────────────────────────────┤            │   │
│  │  │ key (TEXT, PK)                         │            │   │
│  │  │ request_fingerprint (SHA256)           │            │   │
│  │  │ response_payload (JSONB, cached)       │            │   │
│  │  │ created_at                             │            │   │
│  │  └────────────────────────────────────────┘            │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Indices: 9 (optimización de queries)
Constraints: CHECK para enums, UNIQUE para email/phone
```

## 📊 Report Service - Schema completo

```
┌─────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                          │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Schema: public (por defecto)                            │   │
│  │                                                         │   │
│  │  ┌──────────────────────────────────────────────────┐   │   │
│  │  │           reports                               │   │   │
│  │  │  (Novedades de ciudadanos)                       │   │   │
│  │  ├──────────────────────────────────────────────────┤   │   │
│  │  │ id (UUID, PK)                                    │   │   │
│  │  │ user_id (UUID, FK → auth_service)               │   │   │
│  │  │ type (ENUM: ZONA_CRITICA|PUNTO_ACOPIO_LLENO)    │   │   │
│  │  │ description (TEXT)                               │   │   │
│  │  │ status (ENUM: ENVIADO|PENDIENTE|EN_PROCESO|...) │   │   │
│  │  │ report_location_id (FK 1:1)  ───────┐           │   │   │
│  │  │ created_at, updated_at               │           │   │   │
│  │  └──────────────────────────────────────┼──────────┘   │   │
│  │                                         │ 1:1          │   │
│  │                    ┌────────────────────▼──────────┐   │   │
│  │                    │  report_locations            │   │   │
│  │                    ├──────────────────────────────┤   │   │
│  │                    │ id (UUID, PK)                │   │   │
│  │                    │ report_id (FK, UNIQUE)       │   │   │
│  │                    │ latitude, longitude          │   │   │
│  │                    │ address, zone                │   │   │
│  │                    │ created_at, updated_at       │   │   │
│  │                    └──────────────────────────────┘   │   │
│  │                                                         │   │
│  │  ┌──────────────────┐       ┌──────────────────────┐   │   │
│  │  │  attachments     │       │  report_histories    │   │   │
│  │  │  (Evidencia)     │       │  (Audit log)         │   │   │
│  │  ├──────────────────┤       ├──────────────────────┤   │   │
│  │  │ id (UUID, PK)    │ 1:N   │ id (UUID, PK)        │   │   │
│  │  │ report_id (FK)──────────▶│ report_id (FK) 1:N ─┐│   │   │
│  │  │ filename         │       │ previous_status  │││   │   │
│  │  │ mime_type        │       │ new_status       │││   │   │
│  │  │ file_path        │       │ change_reason    │││   │   │
│  │  │ file_size        │       │ changed_by       │││   │   │
│  │  │ uploaded_at      │       │ changed_at       │││   │   │
│  │  └──────────────────┘       └────┬─────────────┘││   │   │
│  │                                  │              ││   │   │
│  │                    ┌─────────────┘              ││   │   │
│  │                    │ 1:N                        ││   │   │
│  │  ┌────────────────▼────────────────────────────┼┘   │   │
│  │  │           tasks                            │     │   │
│  │  │  (Tareas para operadores)                  │     │   │
│  │  ├──────────────────────────────────────────────┤    │   │
│  │  │ id (UUID, PK)                              │    │   │
│  │  │ report_id (FK) ──────────────────────────┐ │    │   │
│  │  │ worker_id (FK) ──────────────────┐       │ │    │   │
│  │  │ status (ENUM: ASIGNADO|...)      │       │ │    │   │
│  │  │ priority (BAJA|NORMAL|ALTA|...)  │       │ │    │   │
│  │  │ assigned_at, started_at, ...     │       │ │    │   │
│  │  │ notes                            │       │ │    │   │
│  │  └────────────────────────────────────────┼─┼──────┘    │
│  │                                          │ │            │
│  │  ┌───────────────────────────────────────┼─┘            │
│  │  │  worker_profiles (Operadores)        │ 1:N          │
│  │  │  (FK → auth_service.operators)       │              │
│  │  ├──────────────────────────────────────┤              │
│  │  │ id (UUID, PK)                        │              │
│  │  │ user_id (FK, UNIQUE) ──────────────┐ │ 0..1        │
│  │  │ full_name                          │ │              │
│  │  │ phone                              │ │              │
│  │  │ status (ACTIVO|INACTIVO|...)       │ │              │
│  │  └────────────────────┬────────────────┘ │              │
│  │                       │ 1:N              │              │
│  │  ┌────────────────────▼──────────────┐   │              │
│  │  │  worker_locations                │   │              │
│  │  │  (Tracking GPS de operadores)     │   │              │
│  │  ├──────────────────────────────────┤   │              │
│  │  │ id (UUID, PK)                   │   │              │
│  │  │ worker_id (FK) ────────────────────┘ │              │
│  │  │ latitude, longitude              │                  │
│  │  │ address                          │                  │
│  │  │ recorded_at                      │                  │
│  │  └──────────────────────────────────┘                  │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Indices: 12 (optimización de queries)
Constraints: CHECK para enums, FK CASCADE/SET NULL
```

## 🔄 Data Flow - Ciudadano Reporta Novedad

```
┌─────────────┐
│  Ciudadano  │
│  (Mobile)   │
└──────┬──────┘
       │
       │ 1. Solicita OTP
       │ POST /api/v1/auth/otp/send
       ▼
┌─────────────────────────────────────┐
│      Auth Service                   │
│  ┌───────────────────────────────┐  │
│  │ 1. Crear citizen record       │  │
│  │ 2. Generar código OTP (6 dígits)
│  │ 3. Crear OTPRequest           │  │
│  │ 4. Enviar vía Twilio          │  │
│  │ 5. Crear evento outbox        │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
       │
       │ SMS recibido
       ▼
┌─────────────────────────────────────┐
│      Ciudadano verifica             │
│  2. Envía código                    │
│  POST /api/v1/auth/otp/verify      │
└─────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Auth Service                   │
│  ┌───────────────────────────────┐  │
│  │ 1. Validar código (max 5)     │  │
│  │ 2. Verificar expiración (5m)  │  │
│  │ 3. Marcar citizen.verified_at │  │
│  │ 4. Generar JWT                │  │
│  │    scope: novedades:crear     │  │
│  │ 5. Crear evento outbox        │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
       │
       │ JWT Token recibido
       ▼
┌─────────────────────────────────────┐
│  Ciudadano crea novedad             │
│  3. POST /api/v1/reports            │
│     Authorization: Bearer <JWT>     │
└─────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Report Service                 │
│  ┌───────────────────────────────┐  │
│  │ 1. Validar JWT               │  │
│  │ 2. Crear report              │  │
│  │ 3. Crear report_location     │  │
│  │ 4. Guardar attachments       │  │
│  │ 5. Crear report_history      │  │
│  │ 6. Crear evento outbox       │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
       │
       │ Evento: report.created
       ▼
┌─────────────────────────────────────┐
│      Task Service                   │
│  ┌───────────────────────────────┐  │
│  │ 1. Recibir evento             │  │
│  │ 2. Crear tarea                │  │
│  │ 3. Asignar a operador         │  │
│  │ 4. Notificar operador         │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

## 👥 Data Flow - Operador Gestiona Tareas

```
┌─────────────────┐
│   Operador      │
│   (Dashboard)   │
└────────┬────────┘
         │
         │ 1. Login
         │ POST /api/v1/auth/login
         ▼
┌──────────────────────────────────┐
│      Auth Service                │
│  ┌──────────────────────────────┐│
│  │ 1. Validar credenciales      ││
│  │ 2. Hash contraseña (bcrypt)  ││
│  │ 3. Generar JWT               ││
│  │    role: operador            ││
│  │    scope: tareas:gestionar   ││
│  │ 4. Crear evento outbox       ││
│  └──────────────────────────────┘│
└──────────────────────────────────┘
         │
         │ JWT Token
         ▼
┌──────────────────────────────────────────┐
│  Operador obtiene tareas                 │
│  2. GET /api/v1/tasks                    │
│     Authorization: Bearer <JWT>          │
└──────────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────┐
│      Task Service                        │
│  ┌──────────────────────────────────────┐│
│  │ 1. Validar JWT + scope               ││
│  │ 2. Query tasks WHERE worker_id=...   ││
│  │ 3. Devolver lista de tareas          ││
│  └──────────────────────────────────────┘│
└──────────────────────────────────────────┘
         │
         │ Tareas asignadas
         ▼
┌──────────────────────────────────────────┐
│  Operador actualiza estado               │
│  3. PATCH /api/v1/tasks/{id}             │
│     { "status": "EN_PROGRESO" }          │
│     Authorization: Bearer <JWT>          │
└──────────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────┐
│      Task Service                        │
│  ┌──────────────────────────────────────┐│
│  │ 1. Validar JWT + scope               ││
│  │ 2. Actualizar task.status            ││
│  │ 3. Crear report_history              ││
│  │ 4. Crear evento outbox               ││
│  │    type: task.status_changed         ││
│  └──────────────────────────────────────┘│
└──────────────────────────────────────────┘
         │
         │ Evento: task.completed
         ▼
┌──────────────────────────────────────────┐
│      Report Service                      │
│  ┌──────────────────────────────────────┐│
│  │ 1. Recibir evento                   ││
│  │ 2. Actualizar report.status        ││
│  │ 3. Crear report_history            ││
│  └──────────────────────────────────────┘│
└──────────────────────────────────────────┘
```

---

**Diagrams Created**: 12 Noviembre 2025
**Version**: 1.0 - Full Architecture
**Services**: 3 (Auth + Report + Task)
**Database Schemas**: 2 (usuario + public)
**Data Flows**: 2 (Citizen OTP + Operator)
