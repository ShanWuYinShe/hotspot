-- 006_notification_module.sql
-- 通知模块：收件箱、订阅设置、设备令牌

-- ============================================================
-- 收件箱
-- ============================================================
CREATE TABLE inbox_items (
    id BIGSERIAL PRIMARY KEY,
    inbox_item_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    category notification_category NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    content TEXT NOT NULL,
    data JSONB,
    action_url TEXT,
    severity notification_severity,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_inbox_items_cursor_id ON inbox_items (cursor_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_inbox_items_user_id ON inbox_items (user_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_inbox_items_unread ON inbox_items (user_id, is_read, created_at DESC) WHERE is_deleted = FALSE AND is_read = FALSE;
CREATE INDEX idx_inbox_unread_by_category ON inbox_items (user_id, category, is_read) WHERE is_deleted = FALSE AND is_read = FALSE;
CREATE INDEX idx_inbox_by_type ON inbox_items (user_id, type, created_at DESC) WHERE is_deleted = FALSE;

-- ============================================================
-- 订阅设置
-- ============================================================
CREATE TABLE notification_subscriptions (
    user_id BIGINT PRIMARY KEY,
    friend_request_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    friend_accepted_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_invite_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_invite_response_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_application_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_application_response_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    message_mention_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    moment_like_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    moment_comment_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    comment_reply_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    system_announcement_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE TRIGGER trigger_notification_subscriptions_updated_at
    BEFORE UPDATE ON notification_subscriptions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 设备令牌
-- ============================================================
CREATE TABLE device_tokens (
    id BIGSERIAL PRIMARY KEY,
    token_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    platform platform NOT NULL,
    device_token VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_device_token UNIQUE (user_id, platform, device_token)
);

CREATE INDEX idx_device_tokens_token_id ON device_tokens (token_id);
CREATE INDEX idx_device_tokens_cursor_id ON device_tokens (cursor_id);
CREATE INDEX idx_device_tokens_user_id ON device_tokens (user_id, created_at DESC);
CREATE INDEX idx_device_tokens_platform ON device_tokens (platform);


