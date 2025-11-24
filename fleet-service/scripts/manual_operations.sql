-- Scripts útiles para gestión manual de datos en Fleet Service
-- ================================================================

-- ======================
-- 1. CREAR CONDUCTORES Y PERFILES DE PRUEBA
-- ======================

-- Conductor 1: Juan Pérez (Zona 1, Solo Carga Lateral)
INSERT INTO drivers (id, full_name, status, created_at, updated_at) 
VALUES (
    '550e8400-e29b-41d4-a716-446655440000'::uuid,
    'Juan Pérez',
    'OFFLINE',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

INSERT INTO operator_profiles (driver_id, preferred_zone_id, can_drive_lateral, can_drive_posterior, created_at, updated_at)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000'::uuid,
    1,
    true,
    false,
    NOW(),
    NOW()
) ON CONFLICT (driver_id) DO NOTHING;

-- Conductor 2: María González (Zona 2, Ambos tipos)
INSERT INTO drivers (id, full_name, status, created_at, updated_at) 
VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::uuid,
    'María González',
    'OFFLINE',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

INSERT INTO operator_profiles (driver_id, preferred_zone_id, can_drive_lateral, can_drive_posterior, created_at, updated_at)
VALUES (
    '550e8400-e29b-41d4-a716-446655440001'::uuid,
    2,
    true,
    true,
    NOW(),
    NOW()
) ON CONFLICT (driver_id) DO NOTHING;

-- Conductor 3: Carlos Ramírez (Zona 1, Solo Carga Posterior)
INSERT INTO drivers (id, full_name, status, created_at, updated_at) 
VALUES (
    '550e8400-e29b-41d4-a716-446655440002'::uuid,
    'Carlos Ramírez',
    'OFFLINE',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

INSERT INTO operator_profiles (driver_id, preferred_zone_id, can_drive_lateral, can_drive_posterior, created_at, updated_at)
VALUES (
    '550e8400-e29b-41d4-a716-446655440002'::uuid,
    1,
    false,
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ======================
-- 2. CONSULTAS ÚTILES
-- ======================

-- Ver todos los conductores con sus perfiles
SELECT 
    d.id,
    d.full_name,
    d.status as driver_status,
    op.preferred_zone_id,
    op.can_drive_lateral,
    op.can_drive_posterior
FROM drivers d
LEFT JOIN operator_profiles op ON d.id = op.driver_id
ORDER BY d.full_name;

-- Ver turnos activos con detalles
SELECT 
    a_s.id as shift_id,
    d.full_name as driver,
    d.status as driver_status,
    t.plate as truck,
    t.type as truck_type,
    t.status as truck_status,
    a_s.start_time,
    a_s.is_active
FROM active_shifts a_s
JOIN drivers d ON a_s.driver_id = d.id
JOIN trucks t ON a_s.truck_id = t.id
ORDER BY a_s.start_time DESC;

-- Ver solo turnos activos
SELECT 
    a_s.id as shift_id,
    d.full_name as driver,
    t.plate as truck,
    t.type as truck_type,
    a_s.start_time
FROM active_shifts a_s
JOIN drivers d ON a_s.driver_id = d.id
JOIN trucks t ON a_s.truck_id = t.id
WHERE a_s.is_active = true
ORDER BY a_s.start_time DESC;

-- Ver conductores elegibles para asignación (con turno activo y disponibles)
SELECT 
    d.id,
    d.full_name,
    d.status,
    t.plate,
    t.type,
    op.preferred_zone_id,
    op.can_drive_lateral,
    op.can_drive_posterior
FROM drivers d
JOIN active_shifts a_s ON d.id = a_s.driver_id AND a_s.is_active = true
JOIN trucks t ON a_s.truck_id = t.id
JOIN operator_profiles op ON d.id = op.driver_id
WHERE d.status = 'DISPONIBLE'
ORDER BY d.full_name;

-- Ver camiones disponibles por tipo
SELECT 
    plate,
    type,
    status
FROM trucks
WHERE status = 'DISPONIBLE'
ORDER BY type, plate;

-- ======================
-- 3. OPERACIONES DE MANTENIMIENTO
-- ======================

-- Cerrar todos los turnos activos (emergencia)
UPDATE active_shifts
SET is_active = false, end_time = NOW()
WHERE is_active = true;

-- Liberar todos los conductores ocupados
UPDATE drivers
SET status = 'DISPONIBLE'
WHERE status = 'OCUPADO';

-- Poner todos los camiones disponibles
UPDATE trucks
SET status = 'DISPONIBLE'
WHERE status = 'EN_USO';

-- Resetear todos los conductores a OFFLINE
UPDATE drivers
SET status = 'OFFLINE';

-- ======================
-- 4. LIMPIAR DATOS DE PRUEBA
-- ======================

-- CUIDADO: Esto eliminará TODOS los datos
-- Descomentar solo si realmente quieres limpiar todo

-- DELETE FROM active_shifts;
-- DELETE FROM operator_profiles;
-- DELETE FROM drivers;
-- DELETE FROM trucks;

-- ======================
-- 5. REPORTES Y ESTADÍSTICAS
-- ======================

-- Contar conductores por estado
SELECT 
    status,
    COUNT(*) as cantidad
FROM drivers
GROUP BY status
ORDER BY status;

-- Contar camiones por estado
SELECT 
    status,
    COUNT(*) as cantidad
FROM trucks
GROUP BY status
ORDER BY status;

-- Contar turnos activos vs cerrados
SELECT 
    CASE 
        WHEN is_active THEN 'Activos'
        ELSE 'Cerrados'
    END as tipo_turno,
    COUNT(*) as cantidad
FROM active_shifts
GROUP BY is_active;

-- Conductores con más turnos realizados
SELECT 
    d.full_name,
    COUNT(a_s.id) as total_turnos
FROM drivers d
LEFT JOIN active_shifts a_s ON d.id = a_s.driver_id
GROUP BY d.id, d.full_name
ORDER BY total_turnos DESC;

-- Promedio de duración de turnos (en horas)
SELECT 
    d.full_name,
    AVG(EXTRACT(EPOCH FROM (a_s.end_time - a_s.start_time)) / 3600) as horas_promedio
FROM drivers d
JOIN active_shifts a_s ON d.id = a_s.driver_id
WHERE a_s.end_time IS NOT NULL
GROUP BY d.id, d.full_name
ORDER BY horas_promedio DESC;

-- Camiones más utilizados
SELECT 
    t.plate,
    t.type,
    COUNT(a_s.id) as veces_usado
FROM trucks t
LEFT JOIN active_shifts a_s ON t.id = a_s.truck_id
GROUP BY t.id, t.plate, t.type
ORDER BY veces_usado DESC;

-- ======================
-- 6. SIMULAR ESCENARIOS DE TESTING
-- ======================

-- Escenario 1: Iniciar turno de Juan con camión ABC-1234
-- (Primero asegúrate de que Juan esté OFFLINE y el camión DISPONIBLE)
INSERT INTO active_shifts (id, driver_id, truck_id, start_time, is_active, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '550e8400-e29b-41d4-a716-446655440000'::uuid,
    (SELECT id FROM trucks WHERE plate = 'ABC-1234'),
    NOW(),
    true,
    NOW(),
    NOW()
);

UPDATE drivers 
SET status = 'DISPONIBLE' 
WHERE id = '550e8400-e29b-41d4-a716-446655440000'::uuid;

UPDATE trucks 
SET status = 'EN_USO' 
WHERE plate = 'ABC-1234';

-- Escenario 2: Simular que Juan fue asignado a una tarea (OCUPADO)
UPDATE drivers 
SET status = 'OCUPADO' 
WHERE id = '550e8400-e29b-41d4-a716-446655440000'::uuid;

-- Escenario 3: Liberar a Juan (completó la tarea)
UPDATE drivers 
SET status = 'DISPONIBLE' 
WHERE id = '550e8400-e29b-41d4-a716-446655440000'::uuid;

-- Escenario 4: Terminar turno de Juan
UPDATE active_shifts
SET is_active = false, end_time = NOW()
WHERE driver_id = '550e8400-e29b-41d4-a716-446655440000'::uuid
  AND is_active = true;

UPDATE drivers 
SET status = 'OFFLINE' 
WHERE id = '550e8400-e29b-41d4-a716-446655440000'::uuid;

UPDATE trucks 
SET status = 'DISPONIBLE' 
WHERE plate = 'ABC-1234';

-- ======================
-- 7. VERIFICACIONES DE INTEGRIDAD
-- ======================

-- Conductores sin perfil de operador
SELECT d.id, d.full_name
FROM drivers d
LEFT JOIN operator_profiles op ON d.id = op.driver_id
WHERE op.id IS NULL;

-- Turnos activos con conductores OFFLINE (inconsistencia)
SELECT 
    a_s.id as shift_id,
    d.id as driver_id,
    d.full_name,
    d.status as driver_status
FROM active_shifts a_s
JOIN drivers d ON a_s.driver_id = d.id
WHERE a_s.is_active = true AND d.status = 'OFFLINE';

-- Camiones EN_USO sin turno activo (inconsistencia)
SELECT t.id, t.plate, t.type, t.status
FROM trucks t
LEFT JOIN active_shifts a_s ON t.id = a_s.truck_id AND a_s.is_active = true
WHERE t.status = 'EN_USO' AND a_s.id IS NULL;

-- Conductores con múltiples turnos activos (inconsistencia)
SELECT 
    driver_id,
    COUNT(*) as turnos_activos
FROM active_shifts
WHERE is_active = true
GROUP BY driver_id
HAVING COUNT(*) > 1;
