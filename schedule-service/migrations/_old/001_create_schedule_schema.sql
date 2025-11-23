-- Schedule Service Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS horarios;

-- Shifts (Turnos del día)
CREATE TABLE IF NOT EXISTS horarios.shifts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  date DATE NOT NULL,
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,
  max_tasks INTEGER NOT NULL DEFAULT 10,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_shifts_date ON horarios.shifts (date);
CREATE INDEX IF NOT EXISTS idx_shifts_status ON horarios.shifts (status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_shifts_date_times ON horarios.shifts (date, start_time, end_time);

-- Scheduled Tasks (Tareas planificadas)
CREATE TABLE IF NOT EXISTS horarios.schedule_tasks (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  task_id TEXT NOT NULL,
  shift_id UUID,
  date DATE NOT NULL,
  status TEXT NOT NULL DEFAULT 'no_planificada' CHECK (status IN ('no_planificada', 'planificada', 'en_progreso', 'completada')),
  task_type TEXT,
  priority TEXT DEFAULT 'medium',
  location TEXT,
  latitude NUMERIC(10, 8),
  longitude NUMERIC(11, 8),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (shift_id) REFERENCES horarios.shifts(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_schedule_tasks_date ON horarios.schedule_tasks (date);
CREATE INDEX IF NOT EXISTS idx_schedule_tasks_status ON horarios.schedule_tasks (status);
CREATE INDEX IF NOT EXISTS idx_schedule_tasks_shift ON horarios.schedule_tasks (shift_id);
CREATE INDEX IF NOT EXISTS idx_schedule_tasks_task_id ON horarios.schedule_tasks (task_id);
