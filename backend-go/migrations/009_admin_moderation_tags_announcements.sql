CREATE TABLE IF NOT EXISTS moment_tags (
  moment_id TEXT NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
  tag_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (moment_id, tag_name)
);

CREATE INDEX IF NOT EXISTS idx_moment_tags_name_created_at ON moment_tags(tag_name, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_blocked_tags (
  name TEXT PRIMARY KEY,
  blocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS system_announcements (
  announcement_id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  severity TEXT NOT NULL,
  action_url TEXT,
  status TEXT NOT NULL DEFAULT 'published',
  send_push BOOLEAN NOT NULL DEFAULT FALSE,
  push_sent_at TIMESTAMPTZ,
  view_count INTEGER NOT NULL DEFAULT 0,
  created_by_user_id TEXT NOT NULL REFERENCES users(user_id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_system_announcements_public_created_at
  ON system_announcements(created_at DESC, announcement_id DESC)
  WHERE deleted_at IS NULL AND status = 'published';

CREATE INDEX IF NOT EXISTS idx_system_announcements_admin_created_at
  ON system_announcements(created_at DESC, announcement_id DESC)
  WHERE deleted_at IS NULL;
