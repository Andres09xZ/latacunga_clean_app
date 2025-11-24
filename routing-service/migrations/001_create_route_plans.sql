-- Migration: Create route_plans table
-- Description: Almacena planes de ruta optimizados por OSRM

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS route_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id VARCHAR(100) NOT NULL,
    zone_id INTEGER NOT NULL,
    distance DECIMAL(10,2) NOT NULL,
    duration DECIMAL(10,2) NOT NULL,
    geometry TEXT NOT NULL,
    waypoint_order TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índices para búsquedas rápidas
CREATE INDEX IF NOT EXISTS idx_route_plans_request_id ON route_plans(request_id);
CREATE INDEX IF NOT EXISTS idx_route_plans_zone_id ON route_plans(zone_id);
CREATE INDEX IF NOT EXISTS idx_route_plans_created_at ON route_plans(created_at DESC);

-- Comentarios
COMMENT ON TABLE route_plans IS 'Planes de ruta optimizados con OSRM';
COMMENT ON COLUMN route_plans.geometry IS 'Polyline codificado de la geometría de la ruta';
COMMENT ON COLUMN route_plans.waypoint_order IS 'Orden optimizado de los waypoints en formato JSON';
COMMENT ON COLUMN route_plans.distance IS 'Distancia total de la ruta en metros';
COMMENT ON COLUMN route_plans.duration IS 'Duración total de la ruta en segundos';
