-- Validación Service Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS validacion;

-- Validaciones (auditoría del proceso)
CREATE TABLE IF NOT EXISTS validacion.validations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  incident_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pendiente','valido','rechazado')),
  reason TEXT,
  validator_kind TEXT NOT NULL DEFAULT 'manual' CHECK (validator_kind IN ('manual','automatico')),
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_validations_incident ON validacion.validations (incident_id, status);
CREATE INDEX IF NOT EXISTS idx_validations_created ON validacion.validations (created_at DESC);

-- Idempotencia (evitar validar dos veces)
CREATE TABLE IF NOT EXISTS validacion.idempotency_keys (
  key TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_idempotency_incident ON validacion.idempotency_keys (incident_id);

-- Outbox de eventos (validacion_creada / validacion_rechazada)
CREATE TABLE IF NOT EXISTS validacion.outbox_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  type TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','published','failed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_status ON validacion.outbox_events (status, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate ON validacion.outbox_events (aggregate_id);
