ALTER TABLE auth_sessions ADD COLUMN IF NOT EXISTS device_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_device ON auth_sessions(user_id, device_id);
