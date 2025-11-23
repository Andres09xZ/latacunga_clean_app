-- Migración 010: Agregar columnas route_type y schedule_time para compatibilidad con scheduler
-- Esta migración adapta la tabla cleaning_zones (creada en 005) al nuevo esquema del scheduler

-- Agregar route_type (derivado o copia de route_name)
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS route_type VARCHAR(50);

-- Inicializar route_type basado en route_name si existe
UPDATE cleaning_zones SET route_type = 
  CASE 
    WHEN route_name LIKE 'RUTA_%' THEN 'URBANO'
    WHEN zone_name LIKE 'RURAL%' THEN 'RURAL'
    WHEN zone_name LIKE 'URBANO%' THEN 'URBANO'
    ELSE 'URBANO'
  END
WHERE route_type IS NULL;

-- Agregar schedule_time (hora del día para la recolección)
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS schedule_time TIME;

-- Inicializar schedule_time con valor por defecto 07:30 (mañana)
UPDATE cleaning_zones SET schedule_time = '07:30:00' WHERE schedule_time IS NULL;

-- Comentarios
COMMENT ON COLUMN cleaning_zones.route_type IS 'Tipo de ruta: URBANO, RURAL, ESPECIAL';
COMMENT ON COLUMN cleaning_zones.schedule_time IS 'Hora programada de recolección (HH:MM:SS)';
