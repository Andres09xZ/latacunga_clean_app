-- Migration: Add id_user column to users table
ALTER TABLE users
ADD COLUMN IF NOT EXISTS id_user VARCHAR(100) UNIQUE;

-- Create index for id_user
CREATE INDEX IF NOT EXISTS idx_users_id_user ON users(id_user);
