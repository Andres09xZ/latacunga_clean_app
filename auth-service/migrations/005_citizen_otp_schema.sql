-- Auth Service - Citizen OTP and Operator Schema
-- Migration: Add citizen authentication via OTP and structured operator management

-- Create schema
CREATE SCHEMA IF NOT EXISTS usuario;

-- =============== OPERATORS ===============
-- Operadores (usuarios internos del sistema: operador, despachador, admin)
CREATE TABLE IF NOT EXISTS usuario.operators (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  email CITEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('operador', 'despachador', 'admin')),
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_operators_email ON usuario.operators (email);
CREATE INDEX IF NOT EXISTS idx_operators_active ON usuario.operators (active);

-- =============== CITIZENS ===============
-- Ciudadanos (autenticación OTP por teléfono para reportar novedades)
CREATE TABLE IF NOT EXISTS usuario.citizens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  phone_e164 TEXT UNIQUE NOT NULL,
  verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_citizens_phone ON usuario.citizens (phone_e164);
CREATE INDEX IF NOT EXISTS idx_citizens_verified ON usuario.citizens (verified_at);

-- =============== OTP REQUESTS ===============
-- Registro de solicitudes OTP (para auditoría y rate limiting)
CREATE TABLE IF NOT EXISTS usuario.otp_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  phone_e164 TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT 'twilio',
  status TEXT NOT NULL CHECK (status IN ('requested', 'verified', 'failed', 'expired')),
  attempts INT NOT NULL DEFAULT 0,
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  verified_at TIMESTAMPTZ,
  error_code TEXT,
  CONSTRAINT fk_otp_citizen_phone FOREIGN KEY (phone_e164)
    REFERENCES usuario.citizens(phone_e164) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_otp_phone ON usuario.otp_requests (phone_e164);
CREATE INDEX IF NOT EXISTS idx_otp_status ON usuario.otp_requests (status);
CREATE INDEX IF NOT EXISTS idx_otp_requested_at ON usuario.otp_requests (requested_at);

-- =============== IDEMPOTENCY ===============
-- Idempotencia para endpoints sensibles (prevenir duplicados)
CREATE TABLE IF NOT EXISTS usuario.idempotency_keys (
  key TEXT PRIMARY KEY,
  request_fingerprint TEXT NOT NULL,
  response_payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_idempotency_created_at ON usuario.idempotency_keys (created_at);

-- =============== OUTBOX EVENTS ===============
-- Event sourcing - Outbox pattern para garantizar consistencia eventual
CREATE TABLE IF NOT EXISTS usuario.outbox_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  type TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'published', 'failed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON usuario.outbox_events (status, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate ON usuario.outbox_events (aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_outbox_type ON usuario.outbox_events (type);
