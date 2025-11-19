-- Migration: 002_enhance_task_service_schema.sql
-- Description: Add support for novedad integration and improved task management
-- Date: 2025-01-13

-- ============================================================
-- Add new columns to tasks table
-- ============================================================

ALTER TABLE tasks 
ADD COLUMN IF NOT EXISTS novedad_id UUID UNIQUE,
ADD COLUMN IF NOT EXISTS source VARCHAR(20) DEFAULT 'novedad',
ADD COLUMN IF NOT EXISTS description TEXT,
ADD COLUMN IF NOT EXISTS latitude NUMERIC(10, 8),
ADD COLUMN IF NOT EXISTS longitude NUMERIC(11, 8),
ADD COLUMN IF NOT EXISTS photo_url TEXT,
ADD COLUMN IF NOT EXISTS evidence JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS version INT DEFAULT 1;

-- Create indexes for frequently queried columns
CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks(state);
CREATE INDEX IF NOT EXISTS idx_tasks_actor_id ON tasks(actor_id);
CREATE INDEX IF NOT EXISTS idx_tasks_novedad_id ON tasks(novedad_id);
CREATE INDEX IF NOT EXISTS idx_tasks_source ON tasks(source);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC);

-- ============================================================
-- Create task_histories table for audit trail
-- ============================================================

CREATE TABLE IF NOT EXISTS task_histories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL,
    old_state VARCHAR(50) NOT NULL,
    new_state VARCHAR(50) NOT NULL,
    reason TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_histories_task_id ON task_histories(task_id);
CREATE INDEX IF NOT EXISTS idx_task_histories_actor_id ON task_histories(actor_id);
CREATE INDEX IF NOT EXISTS idx_task_histories_created_at ON task_histories(created_at DESC);

-- ============================================================
-- Update processed_events table for better event tracking
-- ============================================================

ALTER TABLE processed_events 
ADD COLUMN IF NOT EXISTS event_type VARCHAR(50),
ADD COLUMN IF NOT EXISTS source_id UUID;

-- Drop existing unique constraint and create new one
ALTER TABLE processed_events 
DROP CONSTRAINT IF EXISTS processed_events_pkey CASCADE,
DROP CONSTRAINT IF EXISTS processed_events_event_id_key CASCADE;

ALTER TABLE processed_events 
ADD PRIMARY KEY (event_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_processed_events_idempotency 
ON processed_events(event_id, consumer, event_type);

CREATE INDEX IF NOT EXISTS idx_processed_events_consumer 
ON processed_events(consumer);

CREATE INDEX IF NOT EXISTS idx_processed_events_event_type 
ON processed_events(event_type);

CREATE INDEX IF NOT EXISTS idx_processed_events_source_id 
ON processed_events(source_id);

CREATE INDEX IF NOT EXISTS idx_processed_events_processed_at 
ON processed_events(processed_at DESC);

-- ============================================================
-- Add comments for documentation
-- ============================================================

COMMENT ON TABLE task_histories IS 'Audit trail tracking all state changes for tasks';
COMMENT ON COLUMN task_histories.old_state IS 'Previous state before transition';
COMMENT ON COLUMN task_histories.new_state IS 'New state after transition';
COMMENT ON COLUMN task_histories.actor_id IS 'User ID who made the state change';
COMMENT ON COLUMN task_histories.reason IS 'Optional reason for the state change';

COMMENT ON TABLE processed_events IS 'Idempotency tracking for event processing';
COMMENT ON COLUMN processed_events.event_type IS 'Type of event: novedad, report, etc.';
COMMENT ON COLUMN processed_events.source_id IS 'ID of the novedad or report that triggered the event';

COMMENT ON COLUMN tasks.source IS 'Source of the task: novedad or report';
COMMENT ON COLUMN tasks.novedad_id IS 'Reference to novedad if source=novedad';
COMMENT ON COLUMN tasks.description IS 'Detailed description of what needs to be done';
COMMENT ON COLUMN tasks.latitude IS 'Latitude coordinate of task location';
COMMENT ON COLUMN tasks.longitude IS 'Longitude coordinate of task location';
COMMENT ON COLUMN tasks.photo_url IS 'URL of photo taken during task completion';
COMMENT ON COLUMN tasks.evidence IS 'Array of photo URLs as evidence of completion';
COMMENT ON COLUMN tasks.version IS 'Version for optimistic locking (future use)';

-- ============================================================
-- Verification queries
-- ============================================================

-- Verify table structure
-- SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'tasks';
-- SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'task_histories';
-- SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'processed_events';
