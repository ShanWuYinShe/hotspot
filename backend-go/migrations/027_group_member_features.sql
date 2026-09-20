-- Group announcements
CREATE TABLE IF NOT EXISTS group_announcements (
  announcement_id TEXT PRIMARY KEY,
  group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  created_by_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_group_announcements_group_pinned_created
  ON group_announcements(group_id, is_pinned DESC, created_at DESC, announcement_id DESC) WHERE deleted_at IS NULL;

-- Group member settings and moderation
ALTER TABLE group_members
  ADD COLUMN IF NOT EXISTS is_muted BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS muted_until TIMESTAMPTZ;

-- Group member bans
CREATE TABLE IF NOT EXISTS group_bans (
  group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  banned_by_user_id TEXT REFERENCES users(user_id) ON DELETE SET NULL,
  reason TEXT,
  banned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_bans_group_banned_at
  ON group_bans(group_id, banned_at DESC, user_id ASC);
