CREATE TABLE IF NOT EXISTS moment_mentions (
  moment_id TEXT NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
  mentioned_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (moment_id, mentioned_user_id)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moment_mentions_user_id_created_at
  ON moment_mentions(mentioned_user_id, created_at DESC, moment_id DESC);
