-- 收件箱提醒表
CREATE TABLE inbox_items (
    item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    category VARCHAR(20) NOT NULL CHECK (category IN ('social', 'system', 'content')),
    item_type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    content TEXT NOT NULL,
    data JSONB,
    action_url TEXT,
    severity VARCHAR(20) CHECK (severity IN ('info', 'warning', 'error')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_read BOOLEAN DEFAULT FALSE
);

-- 收件箱索引
CREATE INDEX idx_inbox_items_user ON inbox_items(user_id, is_read, created_at DESC);
CREATE INDEX idx_inbox_items_category ON inbox_items(user_id, category, created_at DESC);
CREATE INDEX idx_inbox_items_type ON inbox_items(item_type);

-- 设备推送Token表
CREATE TABLE device_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    token TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (user_id, platform, token)
);

-- 设备Token索引
CREATE INDEX idx_device_tokens_user ON device_tokens(user_id);

-- 事件订阅配置表
CREATE TABLE event_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    friend_request_enabled BOOLEAN DEFAULT TRUE,
    friend_accepted_enabled BOOLEAN DEFAULT TRUE,
    group_invite_enabled BOOLEAN DEFAULT TRUE,
    group_invite_response_enabled BOOLEAN DEFAULT TRUE,
    message_mention_enabled BOOLEAN DEFAULT TRUE,
    moment_like_enabled BOOLEAN DEFAULT TRUE,
    moment_comment_enabled BOOLEAN DEFAULT TRUE,
    comment_reply_enabled BOOLEAN DEFAULT TRUE,
    system_announcement_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (user_id)
);

-- 订阅配置索引
CREATE INDEX idx_event_subscriptions_user ON event_subscriptions(user_id);

-- 信息流反馈表
CREATE TABLE feed_feedback (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    item_id VARCHAR(255) NOT NULL,
    item_type VARCHAR(20) NOT NULL CHECK (item_type IN ('moment', 'group_card', 'system')),
    action VARCHAR(20) NOT NULL CHECK (action IN ('hide', 'not_interested', 'clicked', 'viewed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 信息流反馈索引
CREATE INDEX idx_feed_feedback_user ON feed_feedback(user_id, created_at DESC);
CREATE INDEX idx_feed_feedback_item ON feed_feedback(item_id, item_type);

-- 媒体文件元数据表
CREATE TABLE media_files (
    file_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    category VARCHAR(20) NOT NULL CHECK (category IN ('avatar', 'group_image', 'moment_image', 'file')),
    file_url TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 媒体文件索引
CREATE INDEX idx_media_files_user ON media_files(user_id, created_at DESC);
CREATE INDEX idx_media_files_category ON media_files(category);

