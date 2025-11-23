-- Migration 011: Minimal Planning Core schema adjustments
-- Adds schedule_config column and zone_metrics table separating accumulation state.

-- Add schedule_config if missing
DO $$ BEGIN
    BEGIN
        ALTER TABLE cleaning_zones ADD COLUMN schedule_config TEXT;
    EXCEPTION WHEN duplicate_column THEN
        -- ignore
    END;
END $$;
-- Optional: drop legacy columns no longer used by the Planning Core (safe in dev; comment if keeping history)
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS route_name;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS route_type;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS schedule_day;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS points_count;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS area_km2;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS schedule_time;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS threshold;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS current_score;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS last_updated;
-- ALTER TABLE cleaning_zones DROP COLUMN IF EXISTS status; -- Re-added below with simplified semantics

DO $$ BEGIN
    BEGIN
        ALTER TABLE cleaning_zones ADD COLUMN status VARCHAR(40) DEFAULT 'ACUMULANDO';
    EXCEPTION WHEN duplicate_column THEN
    END;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='zone_metrics') THEN
        CREATE TABLE zone_metrics (
            zone_id INT PRIMARY KEY REFERENCES cleaning_zones(id) ON DELETE CASCADE,
            current_score INT NOT NULL DEFAULT 0,
            threshold INT NOT NULL DEFAULT 50,
            last_trigger TIMESTAMPTZ NULL,
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_zone_metrics_score ON zone_metrics(current_score);
CREATE INDEX IF NOT EXISTS idx_zone_metrics_threshold ON zone_metrics(threshold);

-- Initialize metrics rows for existing zones if missing
INSERT INTO zone_metrics (zone_id, current_score, threshold)
SELECT id, 0, 50 FROM cleaning_zones cz
WHERE NOT EXISTS (SELECT 1 FROM zone_metrics zm WHERE zm.zone_id = cz.id);
