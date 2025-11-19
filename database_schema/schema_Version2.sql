-- PostgreSQL schema (no PostGIS) para interacción de usuarios, operadores y reportes
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enums
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname='user_role') THEN
    CREATE TYPE user_role AS ENUM ('user','operador','admin');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname='report_type') THEN
    CREATE TYPE report_type AS ENUM ('ZONA_CRITICA','PUNTO_ACOPIO_LLENO');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname='report_state') THEN
    CREATE TYPE report_state AS ENUM ('ENVIADO','PENDIENTE','VERIFICADO','EMITIDO','RECHAZADO','COMPLETADO');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname='task_type') THEN
    CREATE TYPE task_type AS ENUM ('LIMPIEZA','RECOLECCION');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname='task_state') THEN
    CREATE TYPE task_state AS ENUM ('PENDIENTE_ASIGNAR','ASIGNADA','EN_CURSO','COMPLETADA','FALLIDA','CANCELADA');
  END IF;
END$$;

-- Users (única tabla) + perfiles por rol (operador opcional)
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE,
  phone TEXT UNIQUE,
  password_hash TEXT,
  role user_role NOT NULL,
  display_name TEXT,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS operator_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  vehicle_capacity INT DEFAULT 0,
  depot_lat NUMERIC(10,7),
  depot_lng NUMERIC(10,7),
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Schedules (operador 1:N)
CREATE TABLE IF NOT EXISTS operator_schedules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  operator_id UUID NOT NULL REFERENCES operator_profiles(id) ON DELETE CASCADE,
  day_of_week SMALLINT,                    -- 0..6
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,
  effective_from DATE DEFAULT CURRENT_DATE,
  effective_to DATE,
  timezone TEXT DEFAULT 'America/Guayaquil',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Collection Points (puntos de acopio)
CREATE TABLE IF NOT EXISTS collection_points (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT UNIQUE,
  name TEXT,
  address TEXT,
  lat NUMERIC(10,7) NOT NULL,
  lng NUMERIC(10,7) NOT NULL,
  zone TEXT,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Reports (ciudadano 1:N reports)
CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  type report_type NOT NULL,
  description TEXT NOT NULL,
  priority_score REAL DEFAULT 0.0,
  state report_state NOT NULL DEFAULT 'ENVIADO',
  collection_point_id UUID REFERENCES collection_points(id) ON DELETE SET NULL, -- asociación directa a punto de acopio (nullable)
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  emitted_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  version INT NOT NULL DEFAULT 1
);

-- Report 1:1 Location
CREATE TABLE IF NOT EXISTS report_locations (
  report_id UUID PRIMARY KEY REFERENCES reports(id) ON DELETE CASCADE,
  lat NUMERIC(10,7) NOT NULL,
  lng NUMERIC(10,7) NOT NULL,
  accuracy_m INT,
  address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Report History 1:N
CREATE TABLE IF NOT EXISTS report_history (
  id BIGSERIAL PRIMARY KEY,
  report_id UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
  from_state report_state,
  to_state report_state,
  changed_by UUID REFERENCES users(id),
  comment TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tasks (report 1:N tasks; operador 1:N tasks)
CREATE TABLE IF NOT EXISTS tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
  operator_id UUID REFERENCES operator_profiles(id) ON DELETE SET NULL,
  type task_type NOT NULL,                 -- LIMPIEZA para workers futuro, RECOLECCION para operadores
  state task_state NOT NULL DEFAULT 'ASIGNADA',
  priority INT DEFAULT 0,
  route_id UUID,                           -- plan de ruta al que pertenece (opcional)
  eta_seconds INT,
  scheduled_for TIMESTAMPTZ,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  version INT NOT NULL DEFAULT 1,
  CONSTRAINT chk_tasks_operator_type CHECK (
    (type='RECOLECCION' AND operator_id IS NOT NULL)
    OR (type='LIMPIEZA')  -- para futuro: worker_id
  )
);

-- Task evidences (N)
CREATE TABLE IF NOT EXISTS task_attachments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  filename TEXT,
  mime_type TEXT,
  remote_url TEXT,
  kind TEXT DEFAULT 'AFTER',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Índices clave
CREATE INDEX IF NOT EXISTS idx_reports_state ON reports(state, created_at);
CREATE INDEX IF NOT EXISTS idx_tasks_operator ON tasks(operator_id, state);
CREATE INDEX IF NOT EXISTS idx_tasks_report ON tasks(report_id);

-- Triggers updated_at
CREATE OR REPLACE FUNCTION trg_set_updated_at() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at=now(); RETURN NEW; END; $$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='users_set_updated_at') THEN
    CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='reports_set_updated_at') THEN
    CREATE TRIGGER reports_set_updated_at BEFORE UPDATE ON reports
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='tasks_set_updated_at') THEN
    CREATE TRIGGER tasks_set_updated_at BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='operator_profiles_set_updated_at') THEN
    CREATE TRIGGER operator_profiles_set_updated_at BEFORE UPDATE ON operator_profiles
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
  END IF;
END$$;