-- Create actors table
CREATE TABLE IF NOT EXISTS actors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  actor_type TEXT NOT NULL,
  location TEXT,
  shift_start TIMESTAMPTZ,
  shift_end TIMESTAMPTZ,
  last_seen_at TIMESTAMPTZ,
  status TEXT DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_actors_type_status ON actors (actor_type, status);

-- Create tasks table
CREATE TABLE IF NOT EXISTS tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id UUID REFERENCES reports(id) ON DELETE SET NULL,
  actor_id UUID REFERENCES actors(id) ON DELETE SET NULL,
  type TEXT NOT NULL,
  state TEXT NOT NULL DEFAULT 'PENDIENTE_ASIGNAR',
  priority INT DEFAULT 0,
  instructions TEXT,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  version INT NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_tasks_report ON tasks (report_id);
CREATE INDEX IF NOT EXISTS idx_tasks_actor ON tasks (actor_id);
CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks (state);

-- Create processed_events table
CREATE TABLE IF NOT EXISTS processed_events (
  event_id UUID PRIMARY KEY,
  consumer TEXT NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_consumer ON processed_events (consumer);