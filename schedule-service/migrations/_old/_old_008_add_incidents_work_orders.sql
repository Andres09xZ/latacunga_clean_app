-- Create incidents and work_orders tables for persistence of trigger logic
-- Incidents: individual events contributing to zone score
-- WorkOrders: generated when threshold reached or manual trigger

CREATE TABLE IF NOT EXISTS work_orders (
    id BIGSERIAL PRIMARY KEY,
    zone_id INT NOT NULL REFERENCES cleaning_zones(id) ON DELETE CASCADE,
    zone_name VARCHAR(100) NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_points INT NOT NULL DEFAULT 0,
    threshold INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS incidents (
    id BIGSERIAL PRIMARY KEY,
    zone_id INT REFERENCES cleaning_zones(id) ON DELETE SET NULL,
    zone_name VARCHAR(100),
    incident_type VARCHAR(50) NOT NULL,
    points INT NOT NULL DEFAULT 0,
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    work_order_id BIGINT REFERENCES work_orders(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_incidents_zone_pending ON incidents(zone_id) WHERE work_order_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_work_orders_zone ON work_orders(zone_id);
