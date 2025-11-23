-- Seed cleaning_zones with new polygon areas
-- WARNING: This will remove existing data
TRUNCATE TABLE cleaning_zones RESTART IDENTITY CASCADE;

-- 1. ZONA CENTRAL (Nocturna - Todos los días)
INSERT INTO cleaning_zones (zone_name, route_type, schedule_time, geom) VALUES 
('URBANO_CENTRAL', 'URBANO', '21:00:00', 
 ST_GeomFromText('POLYGON((-78.625 -0.925, -78.610 -0.925, -78.610 -0.945, -78.625 -0.945, -78.625 -0.925))', 4326));

-- 2. ZONA NORTE (Urbana - Diurna - Mar/Jue/Sab) -> day constraints handled in Go service
INSERT INTO cleaning_zones (zone_name, route_type, schedule_time, geom) VALUES 
('URBANO_NORTE', 'URBANO', '07:30:00', 
 ST_GeomFromText('POLYGON((-78.625 -0.925, -78.610 -0.925, -78.600 -0.890, -78.630 -0.890, -78.625 -0.925))', 4326));

-- 3. ZONA SUR (Urbana - Diurna - Lun/Mie/Vie)
INSERT INTO cleaning_zones (zone_name, route_type, schedule_time, geom) VALUES 
('URBANO_SUR', 'URBANO', '07:30:00', 
 ST_GeomFromText('POLYGON((-78.625 -0.945, -78.610 -0.945, -78.600 -0.980, -78.640 -0.980, -78.625 -0.945))', 4326));

-- 4. ZONA RURAL NORTE (Mulaló, Pastocalle, etc.)
INSERT INTO cleaning_zones (zone_name, route_type, schedule_time, geom) VALUES 
('RURAL_NORTE', 'RURAL', '07:30:00', 
 ST_GeomFromText('POLYGON((-78.630 -0.890, -78.600 -0.890, -78.550 -0.800, -78.700 -0.800, -78.630 -0.890))', 4326));

-- 5. ZONA RURAL SUR (Belisario, Poaló, etc.)
INSERT INTO cleaning_zones (zone_name, route_type, schedule_time, geom) VALUES 
('RURAL_SUR', 'RURAL', '07:30:00', 
 ST_GeomFromText('POLYGON((-78.640 -0.980, -78.600 -0.980, -78.550 -1.050, -78.700 -1.050, -78.640 -0.980))', 4326));
