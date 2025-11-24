-- Crear camión LAA-1020 para pruebas E2E
INSERT INTO trucks (plate, type, status, capacity_kg, created_at, updated_at)
VALUES ('LAA-1020', 'CARGA_LATERAL', 'DISPONIBLE', 5000, NOW(), NOW())
ON CONFLICT (plate) DO NOTHING;
