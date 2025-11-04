-- 收件箱提醒表
CREATE TABLE inbox_items (
    id BIGSERIAL PRIMARY KEY,
    item_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    category inbox_category NOT NULL CHECK (category != 'any'::inbox_category),
    item_type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    content TEXT NOT NULL,
    data JSONB,
    action_url TEXT,
    severity VARCHAR(20) CHECK (severity IN ('info', 'warning', 'error')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    is_read BOOLEAN DEFAULT FALSE NOT NULL
);

-- 收件箱索引
CREATE INDEX idx_inbox_items_id ON inbox_items(id DESC);
CREATE UNIQUE INDEX idx_inbox_items_item_id ON inbox_items(item_id);
CREATE INDEX idx_inbox_items_user ON inbox_items(user_id, is_read, id DESC);
CREATE INDEX idx_inbox_items_category ON inbox_items(user_id, category, id DESC);
CREATE INDEX idx_inbox_items_type ON inbox_items(item_type);

-- 设备推送Token表
CREATE TABLE device_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('ios', 'android', 'web')),
    token TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (user_id, platform, token)
);

-- 设备Token索引
CREATE INDEX idx_device_tokens_id ON device_tokens(id DESC);
CREATE INDEX idx_device_tokens_user ON device_tokens(user_id);

-- 事件订阅配置表
CREATE TABLE event_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    friend_request_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    friend_accepted_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    group_invite_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    group_invite_response_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    message_mention_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    moment_like_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    moment_comment_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    comment_reply_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    system_announcement_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (user_id)
);

-- 订阅配置索引
CREATE INDEX idx_event_subscriptions_id ON event_subscriptions(id DESC);
CREATE INDEX idx_event_subscriptions_user ON event_subscriptions(user_id);

