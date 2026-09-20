CREATE TABLE IF NOT EXISTS group_message_read_state (
    group_id TEXT NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    last_read_message_id TEXT NULL REFERENCES group_messages(message_id) ON DELETE SET NULL,
    last_read_sent_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_message_read_state_user_updated_at
    ON group_message_read_state (user_id, updated_at DESC);
