CREATE TABLE IF NOT EXISTS private_chats (
  chat_key TEXT PRIMARY KEY,
  user_a_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  user_b_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS groups (
  group_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  avatar_url TEXT,
  description TEXT,
  has_password BOOLEAN NOT NULL DEFAULT FALSE,
  location_name TEXT NOT NULL,
  latitude DOUBLE PRECISION NOT NULL,
  longitude DOUBLE PRECISION NOT NULL,
  join_method TEXT NOT NULL DEFAULT 'open',
  is_searchable BOOLEAN NOT NULL DEFAULT TRUE,
  max_members INTEGER NOT NULL DEFAULT 200,
  owner_id TEXT REFERENCES users(user_id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  tags TEXT[] NOT NULL DEFAULT '{}',
  s2_cell_token TEXT NOT NULL DEFAULT '',
  s2_cell_id BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_groups_owner_created_at ON groups(owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_groups_active_created_at ON groups(created_at DESC, group_id ASC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_groups_s2_cell_id ON groups(s2_cell_id);
CREATE INDEX IF NOT EXISTS idx_groups_visible_s2_cell_id ON groups(s2_cell_id) WHERE deleted_at IS NULL AND is_searchable = TRUE;

CREATE TABLE IF NOT EXISTS private_messages (
  message_id TEXT PRIMARY KEY,
  chat_key TEXT NOT NULL REFERENCES private_chats(chat_key) ON DELETE CASCADE,
  sender_user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  content TEXT,
  share_code TEXT,
  sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_recalled BOOLEAN NOT NULL DEFAULT FALSE,
  recalled_at TIMESTAMPTZ,
  content_prev TEXT,
  edited_at TIMESTAMPTZ,
  file_id TEXT,
  reply_to_message_id TEXT NULL REFERENCES private_messages(message_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_private_messages_chat_key_sent_at ON private_messages(chat_key, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_private_messages_chat_key_reply_to ON private_messages(chat_key, reply_to_message_id);

CREATE TABLE IF NOT EXISTS group_messages (
  message_id TEXT PRIMARY KEY,
  group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
  sender_user_id TEXT NULL REFERENCES users(user_id) ON DELETE SET NULL,
  type TEXT NOT NULL,
  content TEXT,
  sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_recalled BOOLEAN NOT NULL DEFAULT FALSE,
  recalled_at TIMESTAMPTZ,
  content_prev TEXT,
  edited_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_group_messages_group_id_sent_at ON group_messages(group_id, sent_at DESC);
