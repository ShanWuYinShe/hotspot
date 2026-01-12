-- 010_notification_inbox.sql
-- 通知中心：持久化存储通知历史
-- 支持 "Zero-Lookup" 体验，存储必要的快照信息

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
