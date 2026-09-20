-- 011_comment_likes.sql
-- Add comment likes support

CREATE TABLE IF NOT EXISTS comment_likes (
  comment_id TEXT NOT NULL REFERENCES comments(comment_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  liked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (comment_id, user_id)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_comment_likes_comment_liked_at
  ON comment_likes(comment_id, liked_at DESC);
