CREATE TABLE IF NOT EXISTS moment_likes (
  moment_id TEXT NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  liked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (moment_id, user_id)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moment_likes_moment_liked_at
  ON moment_likes(moment_id, liked_at DESC, user_id ASC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moment_likes_user_liked_at
  ON moment_likes(user_id, liked_at DESC, moment_id DESC);

CREATE TABLE IF NOT EXISTS moment_favorites (
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  moment_id TEXT NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
  favorited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, moment_id)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moment_favorites_user_favorited_at
  ON moment_favorites(user_id, favorited_at DESC, moment_id DESC);
