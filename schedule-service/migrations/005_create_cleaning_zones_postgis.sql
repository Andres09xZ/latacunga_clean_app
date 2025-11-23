-- ============================================================
-- Migración 005: Infraestructura de Datos Espaciales (PostGIS)
-- Objetivo: Crear tabla para zonas de recolección con geometrías
-- Fecha: 2025-11-22
-- Autor: Sistema de Zonificación Automática
-- ============================================================

-- PASO 1: Habilitar extensión PostGIS
-- Esta extensión agrega soporte para tipos de datos geográficos
CREATE EXTENSION IF NOT EXISTS postgis;

-- Verificar versión de PostGIS instalada
-- SELECT PostGIS_Version();

-- ============================================================
-- PASO 2: Crear tabla cleaning_zones
-- ============================================================

-- IMPORTANTE: NO eliminar la tabla en producción - las zonas son persistentes
-- DROP TABLE IF EXISTS cleaning_zones CASCADE; -- COMENTADO para preservar datos

CREATE TABLE IF NOT EXISTS cleaning_zones (
    -- Identificador único
    id SERIAL PRIMARY KEY,
    
    -- Información de la zona
    zone_name VARCHAR(100) NOT NULL,   -- Ejemplo: "RUTA 1 LUNES"
    route_name VARCHAR(50) NOT NULL,   -- Ejemplo: "RUTA_1"
    schedule_day INTEGER NOT NULL,     -- 0=Domingo, 1=Lunes, 2=Martes... 6=Sábado
    
    -- Metadatos de la zona
    points_count INTEGER DEFAULT 0,    -- Cantidad de puntos usados para generar la geometría
    area_km2 DECIMAL(10, 4),          -- Área aproximada en km²
    
    -- Geometría espacial (WGS84 - EPSG:4326)
    -- MultiPolygon permite múltiples polígonos fusionados (manchas orgánicas)
    geom GEOMETRY(MULTIPOLYGON, 4326) NOT NULL,
    
    -- Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_schedule_day CHECK (schedule_day >= 0 AND schedule_day <= 6),
    CONSTRAINT unique_zone_schedule UNIQUE (route_name, schedule_day)
);

-- ============================================================
-- PASO 3: Crear índices para optimización
-- ============================================================

-- Índice espacial GIST (CRÍTICO para consultas geográficas)
-- Permite búsquedas rápidas tipo "¿qué zona contiene este punto?"
CREATE INDEX IF NOT EXISTS idx_zones_geom ON cleaning_zones USING GIST (geom);

-- Índices para búsquedas comunes
CREATE INDEX IF NOT EXISTS idx_zones_route_name ON cleaning_zones (route_name);
CREATE INDEX IF NOT EXISTS idx_zones_schedule_day ON cleaning_zones (schedule_day);
CREATE INDEX IF NOT EXISTS idx_zones_zone_name ON cleaning_zones (zone_name);

-- Índice compuesto para consultas por ruta y día
CREATE INDEX IF NOT EXISTS idx_zones_route_day ON cleaning_zones (route_name, schedule_day);

-- ============================================================
-- PASO 4: Comentarios para documentación
-- ============================================================

COMMENT ON TABLE cleaning_zones IS 'Zonas geográficas de recolección de basura generadas automáticamente';
COMMENT ON COLUMN cleaning_zones.zone_name IS 'Nombre completo de la zona (ej: RUTA 1 LUNES)';
COMMENT ON COLUMN cleaning_zones.route_name IS 'Identificador de la ruta (ej: RUTA_1, LATERAL_ORIENTAL)';
COMMENT ON COLUMN cleaning_zones.schedule_day IS 'Día de la semana: 0=Domingo, 1=Lunes, 2=Martes, 3=Miércoles, 4=Jueves, 5=Viernes, 6=Sábado';
COMMENT ON COLUMN cleaning_zones.points_count IS 'Número de puntos geográficos usados para generar el polígono';
COMMENT ON COLUMN cleaning_zones.area_km2 IS 'Área aproximada de la zona en kilómetros cuadrados';
COMMENT ON COLUMN cleaning_zones.geom IS 'Geometría MultiPolygon en sistema de coordenadas WGS84 (EPSG:4326)';

-- ============================================================
-- PASO 5: Función helper para calcular área automáticamente
-- ============================================================

CREATE OR REPLACE FUNCTION calculate_zone_area()
RETURNS TRIGGER AS $$
BEGIN
    -- Calcular área en km² usando Geography (más preciso que Geometry)
    -- Geography considera la curvatura de la Tierra
    NEW.area_km2 := ST_Area(NEW.geom::geography) / 1000000.0;
    NEW.updated_at := CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para calcular área automáticamente al insertar/actualizar
DROP TRIGGER IF EXISTS trigger_calculate_area ON cleaning_zones;
CREATE TRIGGER trigger_calculate_area
    BEFORE INSERT OR UPDATE OF geom
    ON cleaning_zones
    FOR EACH ROW
    EXECUTE FUNCTION calculate_zone_area();

-- ============================================================
-- PASO 6: Vistas útiles para consultas
-- ============================================================

-- Vista: Zonas agrupadas por ruta
CREATE OR REPLACE VIEW zones_by_route AS
SELECT 
    route_name,
    COUNT(*) as total_zones,
    SUM(area_km2) as total_area_km2,
    AVG(points_count) as avg_points_per_zone,
    STRING_AGG(zone_name, ', ' ORDER BY schedule_day) as zones
FROM cleaning_zones
GROUP BY route_name
ORDER BY route_name;

COMMENT ON VIEW zones_by_route IS 'Resumen de zonas agrupadas por ruta de recolección';

-- Vista: Zonas por día de la semana
CREATE OR REPLACE VIEW zones_by_day AS
SELECT 
    schedule_day,
    CASE schedule_day
        WHEN 0 THEN 'Domingo'
        WHEN 1 THEN 'Lunes'
        WHEN 2 THEN 'Martes'
        WHEN 3 THEN 'Miércoles'
        WHEN 4 THEN 'Jueves'
        WHEN 5 THEN 'Viernes'
        WHEN 6 THEN 'Sábado'
    END as day_name,
    COUNT(*) as total_zones,
    SUM(area_km2) as total_area_km2,
    STRING_AGG(route_name, ', ' ORDER BY route_name) as routes
FROM cleaning_zones
GROUP BY schedule_day
ORDER BY schedule_day;

COMMENT ON VIEW zones_by_day IS 'Resumen de zonas agrupadas por día de la semana';

-- ============================================================
-- PASO 7: Función de búsqueda espacial
-- ============================================================

-- Función: Encontrar zona que contiene un punto (lat, lon)
CREATE OR REPLACE FUNCTION find_zone_by_point(
    p_latitude DECIMAL,
    p_longitude DECIMAL
)
RETURNS TABLE (
    zone_id INTEGER,
    zone_name VARCHAR,
    route_name VARCHAR,
    day_name VARCHAR,
    distance_meters DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        cz.id,
        cz.zone_name,
        cz.route_name,
        CASE cz.schedule_day
            WHEN 0 THEN 'Domingo'
            WHEN 1 THEN 'Lunes'
            WHEN 2 THEN 'Martes'
            WHEN 3 THEN 'Miércoles'
            WHEN 4 THEN 'Jueves'
            WHEN 5 THEN 'Viernes'
            WHEN 6 THEN 'Sábado'
        END,
        ST_Distance(
            cz.geom::geography,
            ST_SetSRID(ST_MakePoint(p_longitude, p_latitude), 4326)::geography
        ) as distance_meters
    FROM cleaning_zones cz
    WHERE ST_Contains(
        cz.geom,
        ST_SetSRID(ST_MakePoint(p_longitude, p_latitude), 4326)
    )
    ORDER BY distance_meters
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION find_zone_by_point IS 'Encuentra la zona de recolección que contiene un punto geográfico dado';

-- ============================================================
-- PASO 8: Datos de prueba (opcional - comentado)
-- ============================================================

-- Ejemplo de consulta para insertar datos desde el GeoJSON:
/*
INSERT INTO cleaning_zones (zone_name, route_name, schedule_day, points_count, geom)
VALUES (
    'RUTA 1 LUNES',
    'RUTA_1',
    1, -- Lunes
    2, -- Puntos
    ST_GeomFromGeoJSON('{"type":"MultiPolygon","coordinates":[[...]]}')
);
*/

-- ============================================================
-- PASO 9: Verificaciones finales
-- ============================================================

-- Verificar que PostGIS está instalado correctamente
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'postgis') THEN
        RAISE EXCEPTION 'PostGIS extension is not installed!';
    END IF;
    RAISE NOTICE 'PostGIS extension is installed and ready!';
END $$;

-- Verificar que la tabla se creó correctamente
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'cleaning_zones') THEN
        RAISE NOTICE 'Table cleaning_zones created successfully!';
    ELSE
        RAISE EXCEPTION 'Failed to create cleaning_zones table!';
    END IF;
END $$;

-- Mostrar información de la tabla
SELECT 
    column_name,
    data_type,
    character_maximum_length,
    is_nullable
FROM information_schema.columns
WHERE table_name = 'cleaning_zones'
ORDER BY ordinal_position;

-- ============================================================
-- RESUMEN DE LA MIGRACIÓN
-- ============================================================

/*
✅ Extensión PostGIS habilitada
✅ Tabla cleaning_zones creada con:
   - Campos: id, zone_name, route_name, schedule_day, points_count, area_km2, geom
   - Tipo: MULTIPOLYGON (WGS84 - EPSG:4326)
   - Constraints: schedule_day [0-6], unique (route_name, schedule_day)
   
✅ Índices creados:
   - idx_zones_geom (GIST) - Búsquedas espaciales rápidas
   - idx_zones_route_name
   - idx_zones_schedule_day
   - idx_zones_zone_name
   - idx_zones_route_day (compuesto)
   
✅ Triggers:
   - trigger_calculate_area - Calcula área automáticamente
   
✅ Vistas:
   - zones_by_route - Resumen por ruta
   - zones_by_day - Resumen por día
   
✅ Funciones:
   - find_zone_by_point(lat, lon) - Búsqueda espacial

PRÓXIMO PASO: Cargar los datos del GeoJSON
Ejecutar: python scripts/load_zones_to_db.py
*/
