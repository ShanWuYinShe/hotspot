CREATE INDEX IF NOT EXISTS idx_moments_non_private_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level <> 'private';
