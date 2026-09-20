-- feed_dismissals：spec 预留（feed/dismiss「标记不感兴趣」未实现，见 api_spec_index.md 预留节）。
-- 当前无任何消费方，保留建表以便未来实现；删除本文件不影响已建库。
CREATE TABLE IF NOT EXISTS feed_dismissals (
    user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, target_type, target_id)
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feed_dismissals_user_created_at
    ON feed_dismissals (user_id, created_at DESC);
