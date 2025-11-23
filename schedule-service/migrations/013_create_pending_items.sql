-- Migration 013: Crear tabla pending_items para almacenamiento temporal de incidentes
-- Cada vez que llega un incidente, se guarda aquí con su ubicación exacta
-- mientras la zona acumula puntos hasta alcanzar el umbral.

CREATE TABLE IF NOT EXISTS pending_items (
    id SERIAL PRIMARY KEY,
    lat DOUBLE PRECISION NOT NULL,
    lon DOUBLE PRECISION NOT NULL,
    incident_id VARCHAR(100) UNIQUE NOT NULL,
    zone_id INTEGER NOT NULL,
    gravity_points INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key a cleaning_zones
    CONSTRAINT fk_pending_zone FOREIGN KEY (zone_id) 
        REFERENCES cleaning_zones(id) ON DELETE CASCADE
);

-- Índices para optimizar consultas frecuentes
CREATE INDEX IF NOT EXISTS idx_pending_zone ON pending_items(zone_id);
CREATE INDEX IF NOT EXISTS idx_pending_status ON pending_items(status);
CREATE INDEX IF NOT EXISTS idx_pending_zone_status ON pending_items(zone_id, status);

-- Trigger para actualizar updated_at automáticamente
CREATE OR REPLACE FUNCTION update_pending_items_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_pending_items_updated_at ON pending_items;
CREATE TRIGGER trigger_update_pending_items_updated_at
    BEFORE UPDATE ON pending_items
    FOR EACH ROW
    EXECUTE FUNCTION update_pending_items_updated_at();

-- Comentarios para documentación
COMMENT ON TABLE pending_items IS 'Almacena temporalmente cada incidente recibido con su ubicación exacta';
COMMENT ON COLUMN pending_items.incident_id IS 'ID único generado para cada incidente (UUID o timestamp-based)';
COMMENT ON COLUMN pending_items.gravity_points IS 'Puntos de gravedad según tipo: SENSOR_LLENO=15, REPORTE_CIUDADANO=10, DESBORDAMIENTO=25';
COMMENT ON COLUMN pending_items.status IS 'Estados: PENDING (acumulando), PROCESSED (umbral alcanzado), CANCELLED (descartado)';
