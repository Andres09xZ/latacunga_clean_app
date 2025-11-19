# language: es
@servicio_tareas
Característica: Generación de tareas desde novedades
  Como sistema
  Quiero convertir novedades verificadas en tareas
  Para planificar su atención

  Antecedentes:
    Given task-service está suscrito a "novedad.creada" y "novedad.verificada"
    And los eventos tienen campos: id, reporter_kind, tipo, descripcion, location (lat,lng), created_at
    And las tareas tienen estados: PENDIENTE, EN_PROGRESO, COMPLETADA, CANCELADA

  # ============================================================
  # Creación de tareas desde novedades
  # ============================================================
  @novedades @task_creation
  Escenario: Crear tarea a partir de una novedad
    Dado que existe una novedad con estado "verificada"
    Cuando el servicio de Tareas recibe "novedad_creada" o "novedad_verificada"
    Entonces crea una tarea con source "novedad" y tipo correspondiente
    Y publica "tarea_creada" con task_id, source, tipo, estado
    And registra el evento en la tabla de eventos procesados para idempotencia

  @novedades @task_creation @offline_first
  Escenario: Crear tarea a partir de lote de novedades offline
    Dado que existen 5 novedades de un ciudadano con source "offline"
    Cuando se sincroniza el lote de novedades
    Entonces el sistema crea 5 tareas, una por cada novedad
    And cada tarea tiene source = "novedad", state = "PENDIENTE"
    And publica evento "lote_tareas_creadas" con count de tareas

  @novedades @task_status_update
  Escenario: Actualizar estado de tarea
    Dado que la tarea fue asignada a un operador
    Cuando el operador marca "en_progreso" y luego "completada"
    Entonces el estado de la tarea cambia y se publica "tarea_actualizada"
    And cada cambio de estado se registra en el historial con timestamp
    And si state = "COMPLETADA", se publica "novedad_atendida" para el servicio de novedades

  # ============================================================
  # Trabajadores (limpieza de zona crítica) — visibilidad y desplazamiento
  # ============================================================
  @worker @visibility
  Escenario: Trabajador ve lista de tareas ordenada por proximidad y prioridad
    Dado existen tareas PENDIENTES con tipos variados en el sistema
    Y un trabajador T está autenticado en la app con ubicación lat -12.33 lng -77.00
    Cuando T solicita GET /tasks/available?lat=-12.33&lng=-77.00
    Entonces el sistema devuelve lista de tareas disponibles
    Y están ordenadas por (priority desc, distance asc)
    Y cada item incluye: task_id, tipo, descripcion, lat, lng, estado

  @worker @task_claim
  Escenario: Trabajador reclama y completa tarea de limpieza
    Dado un trabajador T ve una tarea disponible
    Cuando T reclama la tarea (POST /tasks/{task_id}/claim)
    Entonces se actualiza task.actor_id = T, state = "EN_PROGRESO"
    Y se publica "tarea_asignada"
    Cuando T marca "COMPLETADA" con evidencia (photo_url)
    Entonces la tarea cambia a "COMPLETADA" y publica "tarea_completada"

  # ============================================================
  # Casos de fallback, reintentos e idempotencia
  # ============================================================
  @fallback
  Escenario: No hay trabajadores disponibles; estado "En espera"
    Dado no hay trabajadores con last_seen_at < 30 minutos
    Cuando task-service recibe "novedad_verificada"
    Entonces crea la tarea con state = "PENDIENTE_ASIGNAR"
    Y publica alerta para admin

  @idempotency
  Escenario: Ignorar duplicados de evento novedad_verificada
    Dado task-service recibió "novedad_verificada" dos veces con mismo novedad_id
    Cuando procesa ambos mensajes
    Entonces solo crea una tarea y registra el segundo evento como procesado

  @concurrency
  Escenario: Manejar actualización concurrente de estado de tarea
    Dado una tarea con state = "PENDIENTE" y version = 1
    Cuando dos operadores intentan cambiar el estado simultáneamente
    Entonces el primero en llegar actualiza correctamente
    Y el segundo obtiene error "Conflict: version mismatch"

  # ============================================================
  # Métricas y aceptación mínimas
  # ============================================================
  @metrics
  Escenario: Métricas mínimas de aceptación para tareas
    Dado sistema en operación
    Cuando se miden métricas
    Entonces tiempo promedio novedad_verificada -> tarea_creada (p95) <= 5 segundos
    Y tiempo median reclamo -> completado (workers) <= 6 horas
    Y tasa de completación de tareas >= 85%