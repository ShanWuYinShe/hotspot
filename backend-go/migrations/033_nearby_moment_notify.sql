-- 033_nearby_moment_notify.sql
-- Add nearby_moment_notify toggle to user_settings so users can opt in to
-- receiving notifications when nearby users post new moments.

ALTER TABLE user_settings
  ADD COLUMN IF NOT EXISTS nearby_moment_notify BOOLEAN NOT NULL DEFAULT FALSE;
