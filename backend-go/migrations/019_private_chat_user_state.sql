CREATE TABLE IF NOT EXISTS private_chat_user_state (
    chat_key TEXT NOT NULL REFERENCES private_chats(chat_key) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    visible_since TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chat_key, user_id)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_private_chat_user_state_user_updated_at
    ON private_chat_user_state (user_id, updated_at DESC);
