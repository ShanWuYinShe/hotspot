-- ============================================================================
-- 09. 通知系统表
-- ============================================================================
-- 
-- 功能职责：
-- - 定义通知系统相关的所有表结构
-- - 支持多种通知类型和推送方式
-- - 提供通知设置和偏好管理
-- - 支持批量通知和模板系统
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql, 04_users.sql
--
-- ============================================================================

-- 通知表
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,                   -- 通知ID（自增主键，用于高性能分页）
    notification_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 通知唯一标识符（对外API使用，保护数据隐私）
    user_id BIGINT NOT NULL,                   -- 接收用户ID
    sender_id BIGINT,                          -- 发送者ID（系统通知为NULL）
    notification_type notification_type_enum NOT NULL, -- 通知类型
    title TEXT NOT NULL,                       -- 通知标题
    content TEXT NOT NULL,                     -- 通知内容
    -- 关联信息
    related_type TEXT,                         -- 关联对象类型：'group'|'moment'|'message'|'user'|'comment'
    related_id TEXT,                           -- 关联对象ID
    -- 推送信息
    push_data JSONB NOT NULL DEFAULT '{}',    -- 推送数据（包含跳转链接、额外参数等）
    image_url TEXT,                            -- 通知图片URL
    action_url TEXT,                           -- 点击跳转URL
    -- 状态和时间
    status notification_status_enum NOT NULL DEFAULT 'unread', -- 通知状态
    priority INTEGER NOT NULL DEFAULT 1,       -- 优先级（1-5，5最高）
    is_push_sent BOOLEAN NOT NULL DEFAULT false, -- 是否已推送
    push_sent_at TIMESTAMP WITH TIME ZONE,     -- 推送时间
    read_at TIMESTAMP WITH TIME ZONE,          -- 阅读时间
    expires_at TIMESTAMP WITH TIME ZONE,       -- 过期时间
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 通知设置表
CREATE TABLE notification_settings (
    id BIGSERIAL PRIMARY KEY,                   -- 设置ID
    user_id BIGINT UNIQUE NOT NULL,            -- 用户ID
    -- 全局设置
    push_enabled BOOLEAN NOT NULL DEFAULT true, -- 推送总开关
    email_enabled BOOLEAN NOT NULL DEFAULT true, -- 邮件通知开关
    sms_enabled BOOLEAN NOT NULL DEFAULT false, -- 短信通知开关
    -- 具体通知类型设置
    system_notifications BOOLEAN NOT NULL DEFAULT true,    -- 系统通知
    message_notifications BOOLEAN NOT NULL DEFAULT true,   -- 消息通知
    group_invite_notifications BOOLEAN NOT NULL DEFAULT true, -- 群组邀请通知
    friend_request_notifications BOOLEAN NOT NULL DEFAULT true, -- 好友请求通知
    moment_like_notifications BOOLEAN NOT NULL DEFAULT true,    -- 瞬间点赞通知
    moment_comment_notifications BOOLEAN NOT NULL DEFAULT true, -- 瞬间评论通知
    mention_notifications BOOLEAN NOT NULL DEFAULT true,        -- 提及通知
    -- 时间设置
    quiet_hours_enabled BOOLEAN NOT NULL DEFAULT false, -- 免打扰时间开关
    quiet_start_time TIME,                     -- 免打扰开始时间
    quiet_end_time TIME,                       -- 免打扰结束时间
    timezone TEXT DEFAULT 'UTC',               -- 时区
    -- 频率控制
    max_notifications_per_hour INTEGER DEFAULT 50, -- 每小时最大通知数
    batch_notifications BOOLEAN NOT NULL DEFAULT true, -- 批量通知开关
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 通知模板表
CREATE TABLE notification_templates (
    id BIGSERIAL PRIMARY KEY,                   -- 模板ID
    template_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 模板唯一标识符
    name TEXT UNIQUE NOT NULL,                 -- 模板名称
    notification_type notification_type_enum NOT NULL, -- 通知类型
    title_template TEXT NOT NULL,              -- 标题模板
    content_template TEXT NOT NULL,            -- 内容模板
    push_template JSONB,                       -- 推送模板
    variables TEXT[],                          -- 模板变量列表
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否启用
    created_by BIGINT,                         -- 创建人ID
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 批量通知任务表
CREATE TABLE notification_batches (
    id BIGSERIAL PRIMARY KEY,                   -- 批量任务ID
    batch_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 批量任务唯一标识符
    name TEXT NOT NULL,                        -- 任务名称
    template_id TEXT,                          -- 使用的模板ID
    target_criteria JSONB NOT NULL,           -- 目标用户筛选条件
    notification_data JSONB NOT NULL,         -- 通知数据
    -- 执行状态
    status TEXT NOT NULL DEFAULT 'pending',    -- 状态：'pending'|'processing'|'completed'|'failed'|'cancelled'
    total_targets INTEGER NOT NULL DEFAULT 0,  -- 目标用户总数
    sent_count INTEGER NOT NULL DEFAULT 0,     -- 已发送数量
    failed_count INTEGER NOT NULL DEFAULT 0,   -- 失败数量
    -- 时间控制
    scheduled_at TIMESTAMP WITH TIME ZONE,     -- 计划执行时间
    started_at TIMESTAMP WITH TIME ZONE,       -- 开始执行时间
    completed_at TIMESTAMP WITH TIME ZONE,     -- 完成时间
    created_by BIGINT NOT NULL,                -- 创建人ID
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 推送设备表
CREATE TABLE push_devices (
    id BIGSERIAL PRIMARY KEY,                   -- 设备ID
    device_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 设备唯一标识符
    user_id BIGINT NOT NULL,                   -- 用户ID
    device_token TEXT NOT NULL,                -- 设备推送令牌
    device_type TEXT NOT NULL,                 -- 设备类型：'ios'|'android'|'web'|'harmony'
    device_info JSONB NOT NULL DEFAULT '{}',  -- 设备信息
    app_version TEXT,                          -- 应用版本
    os_version TEXT,                           -- 系统版本
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否活跃
    last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 最后使用时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    
    UNIQUE(user_id, device_token)              -- 防止重复设备
);

-- 通知发送日志表
CREATE TABLE notification_logs (
    id BIGSERIAL PRIMARY KEY,                   -- 日志ID
    notification_id TEXT NOT NULL,            -- 通知ID
    user_id BIGINT NOT NULL,                   -- 用户ID
    channel TEXT NOT NULL,                     -- 发送渠道：'push'|'email'|'sms'|'websocket'
    status TEXT NOT NULL,                      -- 发送状态：'pending'|'sent'|'delivered'|'failed'|'clicked'
    provider TEXT,                             -- 服务提供商
    provider_message_id TEXT,                  -- 提供商消息ID
    error_message TEXT,                        -- 错误信息
    retry_count INTEGER NOT NULL DEFAULT 0,    -- 重试次数
    metadata JSONB NOT NULL DEFAULT '{}',     -- 发送元数据
    sent_at TIMESTAMP WITH TIME ZONE,          -- 发送时间
    delivered_at TIMESTAMP WITH TIME ZONE,     -- 送达时间
    clicked_at TIMESTAMP WITH TIME ZONE,       -- 点击时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() -- 创建时间
);

-- 通知统计表（用于性能优化）
CREATE TABLE notification_statistics (
    id BIGSERIAL PRIMARY KEY,                   -- 统计ID
    user_id BIGINT UNIQUE NOT NULL,            -- 用户ID
    total_notifications INTEGER NOT NULL DEFAULT 0,    -- 总通知数
    unread_notifications INTEGER NOT NULL DEFAULT 0,   -- 未读通知数
    total_push_sent INTEGER NOT NULL DEFAULT 0,        -- 总推送数
    total_push_clicked INTEGER NOT NULL DEFAULT 0,     -- 总点击数
    last_notification_at TIMESTAMP WITH TIME ZONE,     -- 最后通知时间
    last_read_at TIMESTAMP WITH TIME ZONE,             -- 最后阅读时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 添加外键约束
ALTER TABLE notifications ADD CONSTRAINT fk_notifications_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE notifications ADD CONSTRAINT fk_notifications_sender_id 
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE notification_settings ADD CONSTRAINT fk_notification_settings_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE notification_templates ADD CONSTRAINT fk_notification_templates_created_by 
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE notification_batches ADD CONSTRAINT fk_notification_batches_template_id 
    FOREIGN KEY (template_id) REFERENCES notification_templates(template_id) ON DELETE SET NULL;

ALTER TABLE notification_batches ADD CONSTRAINT fk_notification_batches_created_by 
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE push_devices ADD CONSTRAINT fk_push_devices_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE notification_logs ADD CONSTRAINT fk_notification_logs_notification_id 
    FOREIGN KEY (notification_id) REFERENCES notifications(notification_id) ON DELETE CASCADE;

ALTER TABLE notification_logs ADD CONSTRAINT fk_notification_logs_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE notification_statistics ADD CONSTRAINT fk_notification_statistics_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 添加检查约束
ALTER TABLE notifications ADD CONSTRAINT check_notifications_title_length 
    CHECK (length(title) >= 1 AND length(title) <= 200);

ALTER TABLE notifications ADD CONSTRAINT check_notifications_content_length 
    CHECK (length(content) >= 1 AND length(content) <= 1000);

ALTER TABLE notifications ADD CONSTRAINT check_notifications_priority 
    CHECK (priority >= 1 AND priority <= 5);

ALTER TABLE notifications ADD CONSTRAINT check_notifications_expires_at 
    CHECK (expires_at IS NULL OR expires_at > created_at);

ALTER TABLE notification_settings ADD CONSTRAINT check_notification_settings_max_notifications 
    CHECK (max_notifications_per_hour IS NULL OR max_notifications_per_hour > 0);

ALTER TABLE notification_settings ADD CONSTRAINT check_notification_settings_quiet_hours 
    CHECK ((quiet_hours_enabled = false) OR 
           (quiet_hours_enabled = true AND quiet_start_time IS NOT NULL AND quiet_end_time IS NOT NULL));

ALTER TABLE notification_batches ADD CONSTRAINT check_notification_batches_counts 
    CHECK (sent_count >= 0 AND failed_count >= 0 AND sent_count + failed_count <= total_targets);

ALTER TABLE notification_logs ADD CONSTRAINT check_notification_logs_retry_count 
    CHECK (retry_count >= 0 AND retry_count <= 10);

-- 添加表注释
COMMENT ON TABLE notifications IS '通知表，存储所有类型的用户通知';
COMMENT ON TABLE notification_settings IS '通知设置表，管理用户的通知偏好和配置';
COMMENT ON TABLE notification_templates IS '通知模板表，定义各种通知类型的模板';
COMMENT ON TABLE notification_batches IS '批量通知任务表，管理批量通知的执行';
COMMENT ON TABLE push_devices IS '推送设备表，管理用户的推送设备信息';
COMMENT ON TABLE notification_logs IS '通知发送日志表，记录通知的发送状态和结果';
COMMENT ON TABLE notification_statistics IS '通知统计表，用于性能优化和数据分析';

-- 添加列注释
COMMENT ON COLUMN notifications.notification_id IS '通知唯一标识符，对外API使用';
COMMENT ON COLUMN notifications.notification_type IS '通知类型：system=系统通知, message=消息通知, group_invite=群组邀请等';
COMMENT ON COLUMN notifications.push_data IS '推送数据，JSON格式，包含跳转链接和额外参数';
COMMENT ON COLUMN notifications.status IS '通知状态：unread=未读, read=已读, archived=已归档';
COMMENT ON COLUMN notification_settings.quiet_hours_enabled IS '免打扰时间开关，启用后在指定时间段内不发送推送';
COMMENT ON COLUMN notification_templates.variables IS '模板变量列表，定义模板中可使用的变量';
COMMENT ON COLUMN notification_batches.target_criteria IS '目标用户筛选条件，JSON格式，定义批量通知的目标用户';
COMMENT ON COLUMN push_devices.device_token IS '设备推送令牌，用于推送服务识别设备';
COMMENT ON COLUMN notification_logs.channel IS '发送渠道：push=推送通知, email=邮件, sms=短信, websocket=WebSocket';
