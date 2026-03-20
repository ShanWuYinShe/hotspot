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
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_settings (
  user_id TEXT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
  privacy_level TEXT NOT NULL DEFAULT 'public',
  location_visible BOOLEAN NOT NULL DEFAULT TRUE,
  presence_visible BOOLEAN NOT NULL DEFAULT TRUE,
  social_lists_visible BOOLEAN NOT NULL DEFAULT TRUE,
  groups_visible BOOLEAN NOT NULL DEFAULT TRUE,
  liked_moments_visible BOOLEAN NOT NULL DEFAULT TRUE,
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
  PRIMARY KEY (user_id, target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_relations_target ON user_relations(target_user_id);
