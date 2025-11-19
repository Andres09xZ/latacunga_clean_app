-- Route Service Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS rutas;

-- Ruta calculada por operador/día
CREATE TABLE IF NOT EXISTS rutas.routes (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  operator_id UUID NOT NULL,
  work_date DATE NOT NULL,
  shift_id UUID,
  status TEXT NOT NULL DEFAULT 'generada' CHECK (status IN ('pendiente','generando','generada','fallida','heuristica')),
  total_distance_m BIGINT,
  total_duration_s BIGINT,
  polyline TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (operator_id, work_date)
);

-- Pasos/visitas de la ruta
CREATE TABLE IF NOT EXISTS rutas.route_steps (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  route_id UUID NOT NULL REFERENCES rutas.routes(id) ON DELETE CASCADE,
  task_id UUID NOT NULL,
  seq INT NOT NULL,
  arrival_eta TIMESTAMPTZ,
  departure_time TIMESTAMPTZ,
  leg_distance_m BIGINT,
  leg_duration_s BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (route_id, seq)
);

-- Caché de matrices de distancia
CREATE TABLE IF NOT EXISTS rutas.distance_matrix_cache (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  key_hash TEXT UNIQUE NOT NULL,
  payload JSONB NOT NULL,
  ttl_until TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Outbox para publicar eventos
CREATE TABLE IF NOT EXISTS rutas.outbox_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  type TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','published','failed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_routes_operator_date ON rutas.routes (operator_id, work_date);
CREATE INDEX IF NOT EXISTS idx_steps_route_seq ON rutas.route_steps (route_id, seq);
CREATE INDEX IF NOT EXISTS idx_rutas_outbox_pending ON rutas.outbox_events (status, created_at);
