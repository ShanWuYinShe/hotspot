CREATE TABLE IF NOT EXISTS moments (
  moment_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  description TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  likes_count INTEGER NOT NULL DEFAULT 0,
  comments_count INTEGER NOT NULL DEFAULT 0,
  shares_count INTEGER NOT NULL DEFAULT 0,
  location_name TEXT NOT NULL,
  latitude DOUBLE PRECISION NOT NULL,
  longitude DOUBLE PRECISION NOT NULL,
  privacy_level TEXT NOT NULL DEFAULT 'public',
  s2_cell_id BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_moments_user_id_created_at ON moments(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_moments_s2_cell_id ON moments(s2_cell_id);
CREATE INDEX IF NOT EXISTS idx_moments_non_private_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level <> 'private';

CREATE TABLE IF NOT EXISTS comments (
  comment_id TEXT PRIMARY KEY,
  moment_id TEXT NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  parent_id TEXT NULL REFERENCES comments(comment_id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  likes_count INTEGER NOT NULL DEFAULT 0,
  replies_count INTEGER NOT NULL DEFAULT 0,
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_comments_moment_id_created_at ON comments(moment_id, created_at DESC);
