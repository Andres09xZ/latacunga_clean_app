-- Migration: Create reports table
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,  -- References users(id) from auth-service, but no FK since separate DB
    type VARCHAR(20) NOT NULL CHECK (type IN ('acopio', 'critico')),
    location GEOGRAPHY(POINT, 4326),  -- PostGIS point
    photo_url TEXT,
    description TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'Pendiente'
        CHECK (status IN ('Pendiente', 'Procesado', 'Aprobado', 'Rechazado')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    synced BOOLEAN DEFAULT FALSE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_reports_user_id ON reports(user_id);
CREATE INDEX IF NOT EXISTS idx_reports_type ON reports(type);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_synced ON reports(synced);
CREATE INDEX IF NOT EXISTS idx_reports_location ON reports USING GIST (location);  -- Spatial index