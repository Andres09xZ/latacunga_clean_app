-- Migration: Allow NULL email and password_hash for OTP users
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
-- Optionally, if email had a NOT NULL constraint with default empty string, consider updating existing empty strings to NULL:
-- UPDATE users SET email = NULL WHERE email = '';
-- UPDATE users SET password_hash = NULL WHERE password_hash = '';
