-- Add scoring columns for threshold-based route triggering
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS threshold INT DEFAULT 0;
ALTER TABLE cleaning_zones ADD COLUMN IF NOT EXISTS current_score INT DEFAULT 0;

-- Set threshold for URBANO_CENTRAL as per feature (50)
UPDATE cleaning_zones SET threshold = 50 WHERE zone_name = 'URBANO_CENTRAL';
