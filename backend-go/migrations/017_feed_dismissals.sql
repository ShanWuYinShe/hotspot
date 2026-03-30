CREATE TABLE IF NOT EXISTS feed_dismissals (
    user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, target_type, target_id)
);

CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user_created_at
    ON feed_dismissals (user_id, created_at DESC);
