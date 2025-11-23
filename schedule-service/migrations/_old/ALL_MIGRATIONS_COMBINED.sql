-- ============================================================
-- EJECUTAR EN NEON SQL EDITOR (o psql)
-- Orden: Ejecutar todo de una vez o línea por línea
-- ============================================================

-- MIGRACIÓN 007: Threshold y current_score
-- ============================================================
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS threshold INTEGER DEFAULT 0;
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS current_score INTEGER DEFAULT 0;

-- Establecer umbrales por zona
UPDATE cleaning_zones SET threshold = 50 WHERE zone_name = 'URBANO_CENTRAL';
UPDATE cleaning_zones SET threshold = 30 WHERE zone_name LIKE 'URBANO%' AND zone_name != 'URBANO_CENTRAL';
UPDATE cleaning_zones SET threshold = 20 WHERE zone_name LIKE 'RURAL%';

-- MIGRACIÓN 009: Status y last_updated
-- ============================================================
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'ACUMULANDO';
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS last_updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP;

UPDATE cleaning_zones SET last_updated = updated_at WHERE last_updated IS NULL;

CREATE INDEX IF NOT EXISTS idx_cleaning_zones_status ON cleaning_zones(status);

-- MIGRACIÓN 010: Route_type y schedule_time
-- ============================================================
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS route_type VARCHAR(50);
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS schedule_time TIME;

-- Inicializar valores basados en datos existentes
UPDATE cleaning_zones SET route_type = 
  CASE 
    WHEN route_name LIKE 'RUTA_%' THEN 'URBANO'
    WHEN zone_name LIKE 'RURAL%' THEN 'RURAL'
    WHEN zone_name LIKE 'URBANO%' THEN 'URBANO'
    ELSE 'URBANO'
  END
WHERE route_type IS NULL;

UPDATE cleaning_zones SET schedule_time = '07:30:00' WHERE schedule_time IS NULL;

-- VERIFICACIÓN: Consultar estructura final
-- ============================================================
SELECT 
    zone_name, 
    route_type, 
    schedule_time, 
    threshold, 
    current_score, 
    status, 
    last_updated
FROM cleaning_zones
LIMIT 5;

-- Si todo está OK, deberías ver las columnas nuevas
