-- Add revoked status columns for shares
ALTER TABLE shares
    ADD COLUMN IF NOT EXISTS is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;

-- Ensure revoked rows are excluded from share_code uniqueness (no change needed, but future queries should filter)
