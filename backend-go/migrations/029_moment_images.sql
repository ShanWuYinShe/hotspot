CREATE TABLE IF NOT EXISTS moment_images (
  moment_id TEXT NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
  file_id TEXT NOT NULL,
  position SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (moment_id, file_id)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moment_images_moment_position
  ON moment_images(moment_id, position);
