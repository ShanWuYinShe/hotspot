CREATE TABLE IF NOT EXISTS users (
  user_id TEXT PRIMARY KEY,
  nickname TEXT NOT NULL,
  avatar_url TEXT,
  user_type TEXT NOT NULL DEFAULT 'normal',
  bio TEXT,
  gender TEXT,
  email TEXT,
  phone TEXT,
  location_name TEXT NOT NULL,
  latitude DOUBLE PRECISION NOT NULL,
  longitude DOUBLE PRECISION NOT NULL,
  presence_status TEXT NOT NULL DEFAULT 'offline',
  last_online TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status TEXT NOT NULL DEFAULT 'normal',
  banned_reason TEXT,
  banned_duration_days INTEGER,
  banned_at TIMESTAMPTZ,
  register_ip TEXT,
  register_location TEXT,
  last_login_ip TEXT,
  last_login_location TEXT,
  s2_cell_id BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_s2_cell_id ON users(s2_cell_id);
CREATE INDEX IF NOT EXISTS idx_users_status_created_at ON users(status, created_at DESC, user_id ASC);

CREATE TABLE IF NOT EXISTS user_settings (
  user_id TEXT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
  privacy_level TEXT NOT NULL DEFAULT 'public',
  location_visible BOOLEAN NOT NULL DEFAULT TRUE,
  presence_visible BOOLEAN NOT NULL DEFAULT TRUE,
  social_lists_visible BOOLEAN NOT NULL DEFAULT TRUE,
  groups_visible BOOLEAN NOT NULL DEFAULT TRUE,
  reacted_moments_visible BOOLEAN NOT NULL DEFAULT TRUE,
  stats_visible BOOLEAN NOT NULL DEFAULT TRUE,
  allow_follow BOOLEAN NOT NULL DEFAULT TRUE,
  allow_private_chat TEXT NOT NULL DEFAULT 'everyone',
  searchable BOOLEAN NOT NULL DEFAULT TRUE,
  phone_searchable BOOLEAN NOT NULL DEFAULT FALSE,
  nickname_searchable BOOLEAN NOT NULL DEFAULT TRUE,
  location_searchable BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS user_relations (
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  target_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  is_following BOOLEAN NOT NULL DEFAULT FALSE,
  is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
  remark TEXT,
  followed_at TIMESTAMPTZ,
  blocked_at TIMESTAMPTZ,
  block_reason TEXT,
  PRIMARY KEY (user_id, target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_relations_target ON user_relations(target_user_id);
CREATE INDEX IF NOT EXISTS idx_user_relations_user_followed_at ON user_relations(user_id, followed_at DESC, target_user_id ASC) WHERE is_following = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_relations_target_followed_at ON user_relations(target_user_id, followed_at DESC, user_id ASC) WHERE is_following = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_relations_user_blocked_at ON user_relations(user_id, blocked_at DESC, target_user_id ASC) WHERE is_blocked = TRUE;
