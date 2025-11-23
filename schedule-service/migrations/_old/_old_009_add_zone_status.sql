-- Migración 009: Agregar columna status y last_updated para control de estado de zonas
-- Ejecutar después de 008_add_incidents_work_orders.sql

-- Agregar columna status (valores: ACUMULANDO, LISTO_PARA_RECOLECCION, EN_PROGRESO)
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'ACUMULANDO';

-- Agregar columna last_updated para tracking de cambios
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS last_updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP;

-- Inicializar last_updated con updated_at existente si existe
UPDATE cleaning_zones SET last_updated = updated_at WHERE last_updated IS NULL;

-- Índice para consultas por estado
CREATE INDEX IF NOT EXISTS idx_cleaning_zones_status ON cleaning_zones(status);

-- Comentarios
COMMENT ON COLUMN cleaning_zones.status IS 'Estado operacional: ACUMULANDO | LISTO_PARA_RECOLECCION | EN_PROGRESO';
COMMENT ON COLUMN cleaning_zones.last_updated IS 'Timestamp del último cambio de estado o score';
