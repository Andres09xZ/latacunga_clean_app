-- Migración 001: Crear schema y tablas de incidentes
-- Servicio de Incidentes (PostgreSQL + PostGIS)

-- Extensiones necesarias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

-- Crear schema
CREATE SCHEMA IF NOT EXISTS incidentes;

-- Tipos enumerados
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'incident_type') THEN
    CREATE TYPE incidentes.incident_type AS ENUM ('punto_acopio','zona_critica','animal_muerto','zona_reciclaje');
  ELSE
    -- Agregar valores que falten
    BEGIN
      ALTER TYPE incidentes.incident_type ADD VALUE IF NOT EXISTS 'animal_muerto';
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
    BEGIN
      ALTER TYPE incidentes.incident_type ADD VALUE IF NOT EXISTS 'zona_reciclaje';
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'incident_status') THEN
    CREATE TYPE incidentes.incident_status AS ENUM ('emitido','valido','rechazado','convertido_en_tarea','cerrado');
  ELSE
    -- Agregar valores faltantes si aplica
    BEGIN
      ALTER TYPE incidentes.incident_status ADD VALUE IF NOT EXISTS 'emitido';
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
    BEGIN
      ALTER TYPE incidentes.incident_status ADD VALUE IF NOT EXISTS 'valido';
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
    BEGIN
      ALTER TYPE incidentes.incident_status ADD VALUE IF NOT EXISTS 'rechazado';
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
    BEGIN
      ALTER TYPE incidentes.incident_status ADD VALUE IF NOT EXISTS 'convertido_en_tarea';
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
  END IF;
END $$;

-- Tabla principal de incidentes
CREATE TABLE IF NOT EXISTS incidentes.incidents (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  reporter_kind TEXT NOT NULL CHECK (reporter_kind IN ('ciudadano','operador')),
  reporter_id UUID,
  type incidentes.incident_type NOT NULL,
  description TEXT,
  location geography(Point,4326) NOT NULL,
  address TEXT,
  status incidentes.incident_status NOT NULL DEFAULT 'emitido',
  incident_day DATE NOT NULL DEFAULT ((now() AT TIME ZONE 'UTC')::date),
  photos_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tabla de adjuntos
CREATE TABLE IF NOT EXISTS incidentes.incident_attachments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  incident_id UUID NOT NULL REFERENCES incidentes.incidents(id) ON DELETE CASCADE,
  file_url TEXT NOT NULL,
  mime_type TEXT,
  size_bytes BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tabla de eventos (histórico/auditoría)
CREATE TABLE IF NOT EXISTS incidentes.incident_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  incident_id UUID NOT NULL REFERENCES incidentes.incidents(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tabla de claves de idempotencia (offline-first)
CREATE TABLE IF NOT EXISTS incidentes.idempotency_keys (
  key TEXT PRIMARY KEY,
  resource_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tabla outbox para eventos
CREATE TABLE IF NOT EXISTS incidentes.outbox_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  type TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','published','failed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);

-- Índices para optimizar consultas
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidentes.incidents (status);
CREATE INDEX IF NOT EXISTS idx_incidents_type ON incidentes.incidents (type);
CREATE INDEX IF NOT EXISTS idx_incidents_day ON incidentes.incidents (incident_day);
CREATE INDEX IF NOT EXISTS idx_incidents_location_gix ON incidentes.incidents USING GIST (location);
CREATE INDEX IF NOT EXISTS idx_incidentes_outbox_pending ON incidentes.outbox_events (status, created_at);

-- Comentarios
COMMENT ON TABLE incidentes.incidents IS 'Tabla principal de incidentes reportados por ciudadanos';
COMMENT ON COLUMN incidentes.incidents.incident_day IS 'Día del incidente capturado automáticamente por el servidor';
COMMENT ON COLUMN incidentes.incidents.location IS 'Ubicación geográfica del incidente (PostGIS Point)';
COMMENT ON TABLE incidentes.idempotency_keys IS 'Claves de idempotencia para operación offline-first';
