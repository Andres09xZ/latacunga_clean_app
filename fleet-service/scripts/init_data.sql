-- Script de inicialización de datos de ejemplo para Fleet Service
-- Ejecutar después de las migraciones automáticas

-- Insertar camiones de ejemplo
INSERT INTO trucks (plate, type, status, created_at, updated_at) VALUES
('ABC-1234', 'CARGA_LATERAL', 'DISPONIBLE', NOW(), NOW()),
('ABC-5678', 'CARGA_LATERAL', 'DISPONIBLE', NOW(), NOW()),
('XYZ-9012', 'CARGA_POSTERIOR', 'DISPONIBLE', NOW(), NOW()),
('XYZ-3456', 'CARGA_POSTERIOR', 'DISPONIBLE', NOW(), NOW()),
('DEF-7890', 'CARGA_LATERAL', 'MANTENIMIENTO', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Nota: Los conductores y perfiles se crean automáticamente 
-- mediante el evento identity.operator.created.v1 desde el Identity Service

-- Consultas útiles para verificar el estado

-- Ver todos los camiones
-- SELECT * FROM trucks ORDER BY created_at DESC;

-- Ver todos los conductores
-- SELECT d.id, d.full_name, d.status, op.preferred_zone_id, op.can_drive_lateral, op.can_drive_posterior
-- FROM drivers d
-- LEFT JOIN operator_profiles op ON d.id = op.driver_id
-- ORDER BY d.created_at DESC;

-- Ver turnos activos
-- SELECT 
--     a_s.id as shift_id,
--     d.full_name as driver,
--     t.plate as truck,
--     t.type as truck_type,
--     a_s.start_time,
--     a_s.is_active
-- FROM active_shifts a_s
-- JOIN drivers d ON a_s.driver_id = d.id
-- JOIN trucks t ON a_s.truck_id = t.id
-- WHERE a_s.is_active = true
-- ORDER BY a_s.start_time DESC;

-- Ver conductores disponibles para asignación
-- SELECT 
--     d.id,
--     d.full_name,
--     d.status,
--     t.plate,
--     t.type,
--     op.preferred_zone_id,
--     op.can_drive_lateral,
--     op.can_drive_posterior
-- FROM drivers d
-- JOIN active_shifts a_s ON d.id = a_s.driver_id AND a_s.is_active = true
-- JOIN trucks t ON a_s.truck_id = t.id
-- JOIN operator_profiles op ON d.id = op.driver_id
-- WHERE d.status = 'DISPONIBLE'
-- ORDER BY d.full_name;
