ALTER TABLE users
  ADD COLUMN IF NOT EXISTS s2_cell_id BIGINT NOT NULL DEFAULT 0;

ALTER TABLE moments
  ADD COLUMN IF NOT EXISTS s2_cell_id BIGINT NOT NULL DEFAULT 0;

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS s2_cell_id BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_s2_cell_id ON users(s2_cell_id);
CREATE INDEX IF NOT EXISTS idx_moments_s2_cell_id ON moments(s2_cell_id);
CREATE INDEX IF NOT EXISTS idx_groups_s2_cell_id ON groups(s2_cell_id);
CREATE INDEX IF NOT EXISTS idx_moments_public_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level = 'public';
CREATE INDEX IF NOT EXISTS idx_groups_visible_s2_cell_id ON groups(s2_cell_id) WHERE deleted_at IS NULL AND is_searchable = TRUE;
