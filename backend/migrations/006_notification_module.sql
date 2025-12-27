-- 006_notification_module.sql
-- 通知模块：推送订阅设置、设备令牌
-- 注：通知是实时推送机制，不存储。冷启动通过各业务 API 获取待处理数据。

-- ============================================================
-- 推送订阅设置（用户推送偏好）
-- 说明：好友 = 互相关注，无独立好友申请流程
-- ============================================================
CREATE TABLE notification_subscriptions (
    user_id BIGINT PRIMARY KEY,
    new_follower_enabled BOOLEAN NOT NULL DEFAULT TRUE,        -- 有人关注我
    mutual_follow_enabled BOOLEAN NOT NULL DEFAULT TRUE,       -- 互相关注（成为好友）
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
    token_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    platform platform NOT NULL,
    device_token VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_device_token UNIQUE (user_id, platform, device_token)
);

CREATE INDEX idx_device_tokens_token_id ON device_tokens (token_id);
CREATE INDEX idx_device_tokens_user_id ON device_tokens (user_id, created_at DESC);
CREATE INDEX idx_device_tokens_platform ON device_tokens (platform);


