-- ============================================================
-- 005_interaction_module.sql
-- 互动模块：标签、通知、反馈、举报、分享
-- ============================================================

-- 1. 标签表 (无外键)
CREATE TABLE IF NOT EXISTS tags (
    tag_id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(50) UNIQUE NOT NULL,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tags_usage ON tags(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_tags_name_gin ON tags USING GIN(name gin_trgm_ops);

-- 2. 用户标签关联 (无外键)
CREATE TABLE IF NOT EXISTS user_tags (
    user_id BIGINT NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, tag_id)
);

-- 3. 瞬间标签关联 (无外键)
CREATE TABLE IF NOT EXISTS moment_tags (
    moment_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (moment_id, tag_id)
);

-- 4. 群组标签关联 (无外键)
CREATE TABLE IF NOT EXISTS group_tags (
    group_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, tag_id)
);

-- 5. 标签搜索统计 (无外键)
CREATE TABLE IF NOT EXISTS tag_search_stats (
    tag_id UUID NOT NULL,
    search_date DATE NOT NULL,
    search_count INTEGER DEFAULT 0,
    PRIMARY KEY (tag_id, search_date)
);

-- 6. 通知表 (分区表)
CREATE TABLE IF NOT EXISTS notifications_partitioned (
    notification_id UUID NOT NULL DEFAULT uuidv7(),
    user_id BIGINT NOT NULL,
    notification_type VARCHAR(50) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (notification_id, created_at),
    CONSTRAINT check_payload_is_object CHECK (jsonb_typeof(payload) = 'object')
) PARTITION BY RANGE (created_at);

-- 创建默认分区
CREATE TABLE IF NOT EXISTS notifications_default PARTITION OF notifications_partitioned DEFAULT;

-- 创建初始分区 (2025, 2026)
DO $$
BEGIN
    PERFORM create_partition('notifications_2025', 'notifications_partitioned', '2025-01-01', '2026-01-01');
    PERFORM create_partition('notifications_2026', 'notifications_partitioned', '2026-01-01', '2027-01-01');
END $$;

-- 索引
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications_partitioned(user_id, created_at DESC, is_read) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications_partitioned(created_at DESC);

-- 重命名
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'notifications') THEN
        ALTER TABLE notifications_partitioned RENAME TO notifications;
    END IF;
END $$;

-- 7. 推荐反馈表 (分区表)
CREATE TABLE IF NOT EXISTS feed_feedback_partitioned (
    feedback_id UUID NOT NULL DEFAULT uuidv7(),
    user_id BIGINT NOT NULL,
    content_type VARCHAR(20) NOT NULL, -- moment, user, group
    content_id UUID NOT NULL, -- ID 统一存为 UUID? 
    -- 注意: user_id 是 BIGINT, moment_id 是 UUID. content_id 混合类型比较麻烦.
    -- 通常做法: 统一转字符串存储, 或者分列存储.
    -- 这里假设 content_id 存储 UUID (moments, groups). 如果是对 user 的反馈, 需要 user_target_id BIGINT?
    -- 为了通用性, 我们可以定义 content_id 为 VARCHAR(100).
    -- 024.sql 中定义为 `content_id UUID`. 这意味着只能反馈 moments/groups? 
    -- 024.sql: `content_id UUID NOT NULL`.
    -- 如果我们要反馈 user (BIGINT), 这会有问题. 
    -- 鉴于 `moments` 和 `groups` 都是 UUID, 且是主要内容, 暂时保留 UUID. 如果需要 user, 可能会有问题.
    -- 无论如何, 我按 024 的定义保留 UUID.
    
    action VARCHAR(20) NOT NULL, -- like, dislike, click, view
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (feedback_id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建默认分区
CREATE TABLE IF NOT EXISTS feed_feedback_default PARTITION OF feed_feedback_partitioned DEFAULT;

-- 创建初始分区 (2025, 2026)
DO $$
BEGIN
    PERFORM create_partition('feed_feedback_2025', 'feed_feedback_partitioned', '2025-01-01', '2026-01-01');
    PERFORM create_partition('feed_feedback_2026', 'feed_feedback_partitioned', '2026-01-01', '2027-01-01');
END $$;

-- 索引
CREATE INDEX IF NOT EXISTS idx_feedback_user ON feed_feedback_partitioned(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_content ON feed_feedback_partitioned(content_id, action);

-- 重命名
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'feed_feedback') THEN
        ALTER TABLE feed_feedback_partitioned RENAME TO feed_feedback;
    END IF;
END $$;

-- 8. 举报表 (无外键)
CREATE TABLE IF NOT EXISTS reports (
    report_id UUID PRIMARY KEY DEFAULT uuidv7(),
    reporter_id BIGINT NOT NULL,
    target_type VARCHAR(20) NOT NULL, -- user, moment, group, comment
    target_id VARCHAR(100) NOT NULL, -- 兼容 UUID 和 BIGINT
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending', -- pending, processing, resolved, rejected
    result TEXT,
    handler_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reports_reporter ON reports(reporter_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_target ON reports(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status, created_at);

-- 9. 分享记录表 (无外键)
CREATE TABLE IF NOT EXISTS shares (
    share_id UUID PRIMARY KEY DEFAULT uuidv7(),
    sharer_id BIGINT NOT NULL,
    content_type VARCHAR(20) NOT NULL,
    content_id VARCHAR(100) NOT NULL,
    platform VARCHAR(20), -- wechat, moment, link
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shares_sharer ON shares(sharer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_shares_content ON shares(content_type, content_id);
