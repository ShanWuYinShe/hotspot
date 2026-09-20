CREATE TABLE IF NOT EXISTS user_profile_tags (
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  tag_name TEXT NOT NULL,
  PRIMARY KEY (user_id, tag_name)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_profile_tags_name_user
  ON user_profile_tags(tag_name, user_id ASC);
