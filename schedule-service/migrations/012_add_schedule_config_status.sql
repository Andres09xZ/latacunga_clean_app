-- Migration 012: Add schedule_config and status fields for Planning Core compatibility
-- These fields allow text-based schedule descriptions and zone status tracking

ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS schedule_config TEXT;
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS status VARCHAR(40) DEFAULT 'ACUMULANDO';

-- Create index for status queries
CREATE INDEX IF NOT EXISTS idx_zones_status ON cleaning_zones(status);

-- Update schedule_config from schedule_day for existing rows
UPDATE cleaning_zones 
SET schedule_config = CASE schedule_day
    WHEN 0 THEN 'DOMINGO'
    WHEN 1 THEN 'LUNES'
    WHEN 2 THEN 'MARTES'
    WHEN 3 THEN 'MIÉRCOLES'
    WHEN 4 THEN 'JUEVES'
    WHEN 5 THEN 'VIERNES'
    WHEN 6 THEN 'SÁBADO'
END
WHERE schedule_config IS NULL;
