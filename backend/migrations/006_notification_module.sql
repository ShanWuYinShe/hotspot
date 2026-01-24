-- 006_notification_module.sql
-- 通知模块：通知历史、推送订阅设置、设备令牌
-- 通知中心：持久化存储通知历史，支持 "Zero-Lookup" 体验

-- ============================================================
-- 通知类型枚举
-- ============================================================
CREATE TYPE notification_type AS ENUM (
    'like',             -- 点赞 (moment, comment)
    'comment',          -- 评论 (moment)
    'reply',            -- 回复 (comment)
    'follow',           -- 关注
    'mention',          -- @提及
    'system',           -- 系统通知
    'group_invite',     -- 群组邀请
    'group_apply',      -- 入群申请
    'group_kick',       -- 被移出群
    'group_quit',       -- 成员退群
    'group_dismiss',    -- 群解散
    'group_admin'       -- 设置管理员
);

-- ============================================================
-- 通知表
-- ============================================================
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    notification_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,                    -- 接收者ID
    type notification_type NOT NULL,            -- 通知类型
    title VARCHAR(255) NOT NULL,                -- 标题
    content TEXT NOT NULL,                      -- 内容
    
    -- 触发者信息 (快照)
    sender_id BIGINT,                           -- 触发者ID (系统通知可为空)
    sender_nickname VARCHAR(255),               -- 触发者昵称快照
    sender_avatar_url VARCHAR(255),             -- 触发者头像快照
    
    -- 目标资源信息 (用于跳转)
    target_type VARCHAR(32),                    -- 'moment', 'comment', 'user', 'group', 'web'
    target_id VARCHAR(255),                     -- 资源ID
    reference_id VARCHAR(255),                  -- 引用ID (如评论所属的动态ID)
    target_preview VARCHAR(255),                -- 目标摘要/封面图 (Zero-Lookup)
    
    is_read BOOLEAN NOT NULL DEFAULT FALSE,     -- 已读状态
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id) WHERE is_read = FALSE;

-- ============================================================
-- 推送订阅设置（用户推送偏好）
-- 说明：好友 = 互相关注，无独立好友申请流程
-- ============================================================
CREATE TABLE notification_subscriptions (
    user_id BIGINT PRIMARY KEY,
    new_follower_enabled BOOLEAN NOT NULL DEFAULT TRUE,        -- 有人关注我
    mutual_follow_enabled BOOLEAN NOT NULL DEFAULT TRUE,       -- 互相关注（成为好友）
    follow_request_enabled BOOLEAN NOT NULL DEFAULT TRUE,      -- 收到关注请求（需审批时）
    follow_request_handled_enabled BOOLEAN NOT NULL DEFAULT TRUE, -- 关注请求被处理
    group_invite_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_invite_response_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_application_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_application_response_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    group_announcement_enabled BOOLEAN NOT NULL DEFAULT TRUE,
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
