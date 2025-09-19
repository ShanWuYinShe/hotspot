-- ============================================================================
-- HotSpot 完整数据库架构 - 全新设计
-- ============================================================================
-- 
-- 功能职责：
-- - 完全按照 backend_doc.mdc 需求文档设计的数据库架构
-- - 统一消息架构（群聊和私聊使用相同的 groups 表）
-- - 用户ID编码规范（64位整数，高8位编码用户类型）
-- - 软删除机制和触发器
-- - 高性能索引和分区设计
-- - 不考虑向后兼容性，全新设计
--
-- ============================================================================

-- 启用必要的扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- 1. 基础枚举类型定义
-- ============================================================================

-- 群组类型枚举
CREATE TYPE group_type_enum AS ENUM ('public_chat', 'private_chat');

-- 成员角色枚举
CREATE TYPE member_role_enum AS ENUM ('owner', 'admin', 'member');

-- 成员状态枚举
CREATE TYPE member_status_enum AS ENUM ('active', 'muted', 'banned', 'left');

-- 消息类型枚举
CREATE TYPE message_type_enum AS ENUM ('text', 'image', 'video', 'audio', 'file', 'location', 'system');

-- 媒体类型枚举
CREATE TYPE media_type_enum AS ENUM ('image', 'video', 'audio', 'file', 'location');

-- 反应类型枚举
CREATE TYPE reaction_type_enum AS ENUM ('like', 'love', 'laugh', 'angry', 'sad', 'wow');

-- 可见性枚举（用于生活瞬间）
CREATE TYPE visibility_enum AS ENUM ('public', 'friends', 'private');

-- 用户关系类型枚举
CREATE TYPE relationship_type_enum AS ENUM ('friend', 'follow', 'block');

-- 用户关系状态枚举
CREATE TYPE relationship_status_enum AS ENUM ('pending', 'accepted', 'rejected', 'active');

-- 举报目标类型枚举
CREATE TYPE report_target_enum AS ENUM ('user', 'moment', 'comment', 'message', 'group');

-- 举报原因枚举
CREATE TYPE report_reason_enum AS ENUM ('spam', 'harassment', 'inappropriate_content', 'fake_account', 'violence', 'other');

-- 举报状态枚举
CREATE TYPE report_status_enum AS ENUM ('pending', 'reviewing', 'resolved', 'dismissed');

-- 通知类型枚举
CREATE TYPE notification_type_enum AS ENUM (
    'friend_request', 'friend_accepted', 'new_follower',
    'moment_liked', 'moment_commented', 'comment_replied',
    'group_invited', 'group_joined', 'group_message',
    'system_announcement', 'account_warning'
);

-- ============================================================================
-- 2. 基础函数定义
-- ============================================================================

-- 生成唯一ID函数
CREATE OR REPLACE FUNCTION generate_unique_id() RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(16), 'hex');
END;
$$ LANGUAGE plpgsql;

-- 用户ID序列管理表
CREATE TABLE user_id_sequences (
    user_type INTEGER PRIMARY KEY,
    sequence_name TEXT NOT NULL,
    description TEXT
);

-- 插入用户类型序列
INSERT INTO user_id_sequences (user_type, sequence_name, description) VALUES
    (1, 'temp_user_seq', '临时用户序列'),
    (2, 'regular_user_seq', '普通用户序列'),
    (3, 'system_user_seq', '系统用户序列');

-- 创建序列
CREATE SEQUENCE temp_user_seq START 1;
CREATE SEQUENCE regular_user_seq START 1;
CREATE SEQUENCE system_user_seq START 1;

-- 生成带用户类型编码的用户ID
CREATE OR REPLACE FUNCTION generate_user_id(user_type INTEGER) RETURNS BIGINT AS $$
DECLARE
    sequence_id BIGINT;
    type_prefix BIGINT;
    sequence_name TEXT;
BEGIN
    -- 获取对应的序列名
    SELECT us.sequence_name INTO sequence_name
    FROM user_id_sequences us
    WHERE us.user_type = generate_user_id.user_type;
    
    IF sequence_name IS NULL THEN
        RAISE EXCEPTION '无效的用户类型: %', user_type;
    END IF;
    
    -- 获取序列号（56位）
    EXECUTE format('SELECT nextval(%L)', sequence_name) INTO sequence_id;
    
    -- 用户类型左移56位作为前缀（高8位）
    type_prefix := user_type::BIGINT << 56;
    
    -- 合并类型前缀和序列号
    RETURN type_prefix | sequence_id;
END;
$$ LANGUAGE plpgsql;

-- 获取用户类型函数
CREATE OR REPLACE FUNCTION get_user_type(user_id BIGINT) RETURNS INTEGER AS $$
BEGIN
    RETURN (user_id >> 56)::INTEGER;
END;
$$ LANGUAGE plpgsql;

-- 验证用户ID有效性
CREATE OR REPLACE FUNCTION is_valid_user_id(user_id BIGINT) RETURNS BOOLEAN AS $$
DECLARE
    user_type INTEGER;
BEGIN
    user_type := get_user_type(user_id);
    RETURN user_type IN (1, 2, 3);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 3. 用户系统表
-- ============================================================================

-- 用户表
CREATE TABLE users (
    id BIGINT PRIMARY KEY DEFAULT generate_user_id(1), -- 编码用户类型的主键（默认临时用户）
    login_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 登录标识符（临时用户为UUID，正式用户为邮箱/手机号）
    nickname TEXT NOT NULL,                      -- 用户昵称
    avatar_url TEXT,                             -- 头像URL
    bio TEXT,                                    -- 个人简介
    password_hash TEXT,                          -- 密码哈希（临时用户为NULL）
    is_verified BOOLEAN NOT NULL DEFAULT false, -- 是否已验证
    verified_at TIMESTAMP WITH TIME ZONE,       -- 验证时间
    is_active BOOLEAN NOT NULL DEFAULT true,    -- 账户是否激活
    is_deleted BOOLEAN NOT NULL DEFAULT false,  -- 软删除标记
    -- 位置信息
    latitude DOUBLE PRECISION,                   -- 当前纬度
    longitude DOUBLE PRECISION,                  -- 当前经度
    location GEOGRAPHY(POINT, 4326),            -- PostGIS地理位置点
    location_name TEXT,                          -- 当前位置名称/地址
    location_accuracy DOUBLE PRECISION,         -- 位置精度（米）
    location_updated_at TIMESTAMP WITH TIME ZONE, -- 位置更新时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE         -- 删除时间
);

-- 用户设置表
CREATE TABLE user_settings (
    id BIGSERIAL PRIMARY KEY,                    -- 设置ID（自增主键，用于高性能分页）
    setting_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 设置唯一标识符（对外API使用，保护数据隐私）
    user_id BIGINT NOT NULL,                    -- 用户ID
    privacy_level TEXT DEFAULT 'public',        -- 隐私级别：'public'|'friends'|'private'
    location_sharing BOOLEAN DEFAULT true,      -- 位置分享开关
    notification_enabled BOOLEAN DEFAULT true,  -- 通知开关
    mention_notifications BOOLEAN DEFAULT true, -- 提及通知开关
    group_invite_notifications BOOLEAN DEFAULT true, -- 群组邀请通知开关
    moment_notifications BOOLEAN DEFAULT true,  -- 生活瞬间通知开关
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    
    UNIQUE(user_id)                             -- 唯一约束
);

-- 用户活动记录表（分区表）
CREATE TABLE user_activity (
    id BIGSERIAL,                               -- 活动记录ID（自增主键，用于高性能分页）
    activity_id TEXT NOT NULL DEFAULT generate_unique_id(), -- 活动唯一标识符（对外API使用，保护数据隐私）
    user_id BIGINT NOT NULL,                    -- 用户ID
    activity_type TEXT NOT NULL,               -- 活动类型：'login'|'logout'|'create_group'|'join_group'|'leave_group'|'create_moment'|'like_moment'|'send_message'|'update_profile'|'update_location'等
    target_type TEXT,                          -- 目标类型：'group'|'moment'|'message'|'user'
    target_id TEXT,                            -- 目标ID（群组ID、瞬间ID、消息ID等）
    description TEXT,                          -- 活动描述
    metadata JSONB NOT NULL DEFAULT '{}',     -- 完整元数据（包含IP地址、设备信息、位置信息、用户代理等）
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 活动时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    PRIMARY KEY (id, created_at),              -- 复合主键包含分区键
    UNIQUE (activity_id, created_at)           -- 唯一约束包含分区键
) PARTITION BY RANGE (created_at);

-- 用户关系表
CREATE TABLE user_relationships (
    id BIGSERIAL PRIMARY KEY,
    relationship_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    from_user_id BIGINT NOT NULL,              -- 发起用户ID
    to_user_id BIGINT NOT NULL,                -- 目标用户ID
    relationship_type relationship_type_enum NOT NULL, -- 关系类型
    status relationship_status_enum NOT NULL DEFAULT 'pending', -- 关系状态
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(from_user_id, to_user_id, relationship_type)
);

-- ============================================================================
-- 4. 群组系统表（统一消息架构）
-- ============================================================================

-- 群组表（包含私聊群组）
CREATE TABLE groups (
    id BIGSERIAL PRIMARY KEY,                    -- 群组ID（自增主键，用于高性能分页）
    group_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 群组唯一标识符（对外API使用，保护数据隐私）
    name TEXT,                                  -- 群组名称（私聊群组可为NULL）
    description TEXT,                           -- 群组描述
    avatar_url TEXT,                            -- 群组头像URL
    creator_id BIGINT NOT NULL,                 -- 创建者ID
    owner_id BIGINT,                            -- 群主ID（私聊群组可为NULL）
    latitude DOUBLE PRECISION,                  -- 纬度（私聊群组可为NULL）
    longitude DOUBLE PRECISION,                 -- 经度（私聊群组可为NULL）
    location GEOGRAPHY(POINT, 4326),           -- PostGIS地理位置点（私聊群组可为NULL）
    location_name TEXT,                         -- 位置名称（私聊群组可为NULL）
    group_type group_type_enum NOT NULL DEFAULT 'public_chat', -- 群组类型：'public_chat'（可搜索群聊）|'private_chat'（不可搜索私聊）
    max_members INTEGER DEFAULT 500,            -- 最大成员数
    password_hash TEXT,                         -- 群组密码哈希（可选）
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否活跃
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),      -- 最后活跃时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 群组成员表（包含私聊参与者）
CREATE TABLE group_members (
    id BIGSERIAL PRIMARY KEY,                    -- 成员关系ID（自增主键，用于高性能分页）
    member_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 成员关系唯一标识符（对外API使用，保护数据隐私）
    group_id BIGINT NOT NULL,                   -- 群组ID
    user_id BIGINT NOT NULL,                    -- 用户ID
    role member_role_enum NOT NULL DEFAULT 'member', -- 成员角色：'owner'|'admin'|'member'
    status member_status_enum NOT NULL DEFAULT 'active', -- 成员状态：'active'|'muted'|'banned'|'left'
    permissions TEXT[] DEFAULT '{}',             -- 权限列表
    can_send_messages BOOLEAN NOT NULL DEFAULT true, -- 发送消息权限
    can_add_participants BOOLEAN NOT NULL DEFAULT false, -- 添加参与者权限
    muted_until TIMESTAMP WITH TIME ZONE,       -- 静音截止时间
    last_read_message_id BIGINT,                -- 最后已读消息ID
    last_read_at TIMESTAMP WITH TIME ZONE,     -- 最后已读时间
    unread_count INTEGER NOT NULL DEFAULT 0,   -- 未读消息数
    notifications_enabled BOOLEAN NOT NULL DEFAULT true, -- 通知开关
    mention_notifications BOOLEAN NOT NULL DEFAULT true, -- 提及通知开关
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 加入时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    left_at TIMESTAMP WITH TIME ZONE,          -- 离开时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    invited_by BIGINT,                          -- 邀请者ID
    
    UNIQUE(group_id, user_id)                   -- 唯一约束
);

-- ============================================================================
-- 5. 统一消息系统表
-- ============================================================================

-- 消息表（分区表，统一架构）
CREATE TABLE messages (
    id BIGSERIAL,                               -- 消息ID（自增主键，用于高性能分页）
    message_id TEXT NOT NULL DEFAULT generate_unique_id(), -- 消息唯一标识符（对外API使用，保护数据隐私）
    group_id BIGINT NOT NULL,                   -- 群组ID（统一架构：直接关联群组）
    sender_id BIGINT,                           -- 发送者ID（系统消息可为NULL）
    content TEXT NOT NULL,                      -- 消息内容
    message_type message_type_enum NOT NULL DEFAULT 'text', -- 消息类型：'text'|'image'|'video'|'audio'|'file'|'location'|'system'
    system_event_type TEXT,                    -- 系统事件类型：'user_joined'|'user_left'|'user_kicked'|'role_changed'|'group_created'等
    reply_to_message_id BIGINT,                -- 回复的消息ID
    thread_id BIGINT,                          -- 话题ID
    is_edited BOOLEAN NOT NULL DEFAULT false,  -- 是否已编辑
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    deleted_for_everyone BOOLEAN NOT NULL DEFAULT false, -- 全员删除标识
    reaction_count INTEGER NOT NULL DEFAULT 0, -- 反应数量
    reply_count INTEGER NOT NULL DEFAULT 0,    -- 回复数量
    metadata JSONB DEFAULT '{}',               -- 扩展元数据（系统消息的详细信息）
    mentioned_users BIGINT[] DEFAULT '{}',      -- 提及的用户ID数组
    affected_users BIGINT[] DEFAULT '{}',       -- 受影响的用户ID数组（系统消息用）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    edited_at TIMESTAMP WITH TIME ZONE,        -- 编辑时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    PRIMARY KEY (id, created_at),              -- 复合主键包含分区键
    UNIQUE (message_id, created_at)            -- 唯一约束包含分区键
) PARTITION BY RANGE (created_at);

-- 消息媒体附件表
CREATE TABLE message_media (
    id BIGSERIAL PRIMARY KEY,                    -- 媒体ID（自增主键，用于高性能分页）
    media_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 媒体唯一标识符（对外API使用，保护数据隐私）
    message_id BIGINT NOT NULL,                 -- 消息ID
    media_type media_type_enum NOT NULL,        -- 媒体类型
    file_name TEXT,                             -- 文件名
    file_size BIGINT,                           -- 文件大小（字节）
    mime_type TEXT,                             -- MIME类型
    file_url TEXT NOT NULL,                     -- 文件URL
    thumbnail_url TEXT,                         -- 缩略图URL
    width INTEGER,                              -- 宽度（图片/视频）
    height INTEGER,                             -- 高度（图片/视频）
    duration INTEGER,                           -- 时长（音频/视频，秒）
    latitude DOUBLE PRECISION,                  -- 纬度（位置类型）
    longitude DOUBLE PRECISION,                 -- 经度（位置类型）
    location_name TEXT,                         -- 位置名称（位置类型）
    metadata JSONB,                             -- 扩展元数据
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 消息反应表
CREATE TABLE message_reactions (
    id BIGSERIAL PRIMARY KEY,                    -- 反应ID（自增主键，用于高性能分页）
    reaction_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 反应唯一标识符（对外API使用，保护数据隐私）
    message_id BIGINT NOT NULL,                 -- 消息ID
    user_id BIGINT NOT NULL,                    -- 用户ID
    reaction_type reaction_type_enum NOT NULL,  -- 反应类型
    emoji TEXT,                                 -- 表情符号
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(message_id, user_id, reaction_type)  -- 唯一约束
);

-- ============================================================================
-- 6. 生活瞬间系统表
-- ============================================================================

-- 生活瞬间表
CREATE TABLE moments (
    id BIGSERIAL PRIMARY KEY,                    -- 瞬间ID（自增主键，用于高性能分页）
    moment_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 瞬间唯一标识符（对外API使用，保护数据隐私）
    user_id BIGINT NOT NULL,                    -- 用户ID
    description TEXT NOT NULL,                  -- 瞬间描述
    latitude DOUBLE PRECISION NOT NULL,         -- 纬度
    longitude DOUBLE PRECISION NOT NULL,        -- 经度
    location GEOGRAPHY(POINT, 4326) NOT NULL,   -- PostGIS地理位置点
    location_name TEXT NOT NULL,                -- 位置名称
    images TEXT[] DEFAULT '{}',                 -- 图片URL数组
    visibility visibility_enum NOT NULL DEFAULT 'public', -- 可见性
    likes_count INTEGER NOT NULL DEFAULT 0,     -- 点赞数
    comments_count INTEGER NOT NULL DEFAULT 0,  -- 评论数
    shares_count INTEGER NOT NULL DEFAULT 0,    -- 分享数
    is_active BOOLEAN NOT NULL DEFAULT true,    -- 是否活跃
    is_deleted BOOLEAN NOT NULL DEFAULT false,  -- 软删除标记
    expires_at TIMESTAMP WITH TIME ZONE,        -- 过期时间（可选）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE         -- 删除时间
);

-- 标签表
CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,                    -- 标签ID（自增主键，用于高性能分页）
    tag_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 标签唯一标识符（对外API使用，保护数据隐私）
    name TEXT UNIQUE NOT NULL,                  -- 标签名称
    description TEXT,                           -- 标签描述
    color TEXT DEFAULT '#007bff',               -- 标签颜色
    usage_count INTEGER NOT NULL DEFAULT 0,    -- 使用计数
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否活跃
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_by BIGINT,                          -- 创建者ID
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 生活瞬间标签关联表
CREATE TABLE moment_tags (
    id BIGSERIAL PRIMARY KEY,                    -- 关联ID（自增主键，用于高性能分页）
    moment_tag_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 关联唯一标识符（对外API使用，保护数据隐私）
    moment_id BIGINT NOT NULL,                  -- 瞬间ID
    tag_id BIGINT NOT NULL,                     -- 标签ID
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(moment_id, tag_id)                   -- 唯一约束
);

-- 生活瞬间点赞表
CREATE TABLE moment_likes (
    id BIGSERIAL PRIMARY KEY,                    -- 点赞ID（自增主键，用于高性能分页）
    like_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 点赞唯一标识符（对外API使用，保护数据隐私）
    moment_id BIGINT NOT NULL,                  -- 瞬间ID
    user_id BIGINT NOT NULL,                    -- 用户ID
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记（取消点赞）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 点赞时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 取消点赞时间
    
    UNIQUE(moment_id, user_id)                  -- 唯一约束
);

-- 生活瞬间评论表
CREATE TABLE moment_comments (
    id BIGSERIAL PRIMARY KEY,
    comment_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    moment_id BIGINT NOT NULL,                 -- 瞬间ID
    user_id BIGINT NOT NULL,                   -- 评论用户ID
    parent_comment_id BIGINT,                  -- 父评论ID（回复功能）
    content TEXT NOT NULL,                     -- 评论内容
    likes_count INTEGER NOT NULL DEFAULT 0,   -- 评论点赞数
    replies_count INTEGER NOT NULL DEFAULT 0, -- 回复数
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- ============================================================================
-- 7. 内容安全系统表
-- ============================================================================

-- 举报表
CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,
    report_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    reporter_id BIGINT NOT NULL,               -- 举报人ID
    reported_user_id BIGINT,                   -- 被举报用户ID
    target_type report_target_enum NOT NULL,   -- 举报目标类型
    target_id BIGINT NOT NULL,                 -- 举报目标ID
    reason report_reason_enum NOT NULL,        -- 举报原因
    description TEXT,                          -- 详细描述
    status report_status_enum NOT NULL DEFAULT 'pending', -- 处理状态
    handled_by BIGINT,                         -- 处理人ID
    handled_at TIMESTAMP WITH TIME ZONE,      -- 处理时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- 8. 通知系统表
-- ============================================================================

-- 通知表
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    notification_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    user_id BIGINT NOT NULL,                   -- 接收用户ID
    type notification_type_enum NOT NULL,      -- 通知类型
    title TEXT NOT NULL,                       -- 通知标题
    content TEXT NOT NULL,                     -- 通知内容
    data JSONB,                               -- 扩展数据
    is_read BOOLEAN NOT NULL DEFAULT false,   -- 是否已读
    is_pushed BOOLEAN NOT NULL DEFAULT false, -- 是否已推送
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE,         -- 阅读时间
    expires_at TIMESTAMP WITH TIME ZONE       -- 过期时间
);

-- ============================================================================
-- 9. 创建分区表
-- ============================================================================

-- 创建用户活动分区表（按月分区）
CREATE TABLE user_activity_2024_01 PARTITION OF user_activity
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE user_activity_2024_02 PARTITION OF user_activity
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE user_activity_2024_03 PARTITION OF user_activity
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

-- 创建消息分区表（按月分区）
CREATE TABLE messages_2024_01 PARTITION OF messages
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE messages_2024_02 PARTITION OF messages
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE messages_2024_03 PARTITION OF messages
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

-- 为消息分区表创建索引
CREATE INDEX CONCURRENTLY idx_messages_2024_01_group_time_covering 
ON messages_2024_01 (group_id, created_at DESC) 
INCLUDE (id, message_id, sender_id, content, message_type) 
WHERE is_deleted = false;

CREATE INDEX CONCURRENTLY idx_messages_2024_01_sender_time 
ON messages_2024_01 (sender_id, created_at DESC) 
WHERE sender_id IS NOT NULL AND is_deleted = false;

CREATE INDEX CONCURRENTLY idx_messages_2024_01_message_id 
ON messages_2024_01 (message_id) 
WHERE is_deleted = false;

-- 为用户活动分区表创建索引
CREATE INDEX CONCURRENTLY idx_user_activity_2024_01_activity_id 
ON user_activity_2024_01 (activity_id) 
WHERE is_deleted = false;

CREATE INDEX CONCURRENTLY idx_user_activity_2024_01_user_id 
ON user_activity_2024_01 (user_id, created_at DESC) 
WHERE is_deleted = false;

-- ============================================================================
-- 10. 创建索引
-- ============================================================================

-- 用户表索引
CREATE INDEX CONCURRENTLY idx_users_login_id ON users(login_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_users_active ON users(updated_at DESC) WHERE is_deleted = false AND is_active = true;
CREATE INDEX CONCURRENTLY idx_users_location ON users USING GIST(location) WHERE is_deleted = false AND location IS NOT NULL;

-- 用户设置表索引
CREATE INDEX CONCURRENTLY idx_user_settings_setting_id ON user_settings(setting_id);
CREATE INDEX CONCURRENTLY idx_user_settings_user_id ON user_settings(user_id);

-- 用户关系表索引
CREATE INDEX CONCURRENTLY idx_user_relationships_from_user ON user_relationships(from_user_id, relationship_type) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_user_relationships_to_user ON user_relationships(to_user_id, relationship_type) WHERE is_deleted = false;

-- 群组表索引
CREATE INDEX CONCURRENTLY idx_groups_group_id ON groups(group_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_groups_creator ON groups(creator_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_groups_location_active ON groups USING GIST(location) WHERE is_deleted = false AND is_active = true;
CREATE INDEX CONCURRENTLY idx_groups_type_active ON groups(group_type, is_active, last_active_at DESC) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_groups_searchable ON groups(group_type, latitude, longitude) WHERE is_deleted = false AND group_type = 'public_chat';

-- 群组成员表索引
CREATE INDEX CONCURRENTLY idx_group_members_member_id ON group_members(member_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_group_members_group_user ON group_members(group_id, user_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_group_members_user_active ON group_members(user_id, status, joined_at DESC) WHERE is_deleted = false AND status = 'active';
CREATE INDEX CONCURRENTLY idx_group_members_group_role ON group_members(group_id, role) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_group_members_unread ON group_members(user_id, unread_count DESC) WHERE is_deleted = false AND unread_count > 0;

-- 消息表索引（分区表索引将在分区创建后添加）

-- 消息媒体表索引
CREATE INDEX CONCURRENTLY idx_message_media_media_id ON message_media(media_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_message_media_message_id ON message_media(message_id) WHERE is_deleted = false;

-- 消息反应表索引
CREATE INDEX CONCURRENTLY idx_message_reactions_message_id ON message_reactions(message_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_message_reactions_user_id ON message_reactions(user_id) WHERE is_deleted = false;

-- 生活瞬间表索引
CREATE INDEX CONCURRENTLY idx_moments_moment_id ON moments(moment_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_moments_user_id ON moments(user_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_moments_location_active ON moments USING GIST(location) WHERE is_deleted = false AND is_active = true;
CREATE INDEX CONCURRENTLY idx_moments_created_at ON moments(created_at DESC) WHERE is_deleted = false;

-- 标签表索引
CREATE INDEX CONCURRENTLY idx_tags_name ON tags(name) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_tags_usage_count ON tags(usage_count DESC) WHERE is_deleted = false AND is_active = true;

-- 生活瞬间标签关联表索引
CREATE INDEX CONCURRENTLY idx_moment_tags_moment_id ON moment_tags(moment_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_moment_tags_tag_id ON moment_tags(tag_id) WHERE is_deleted = false;

-- 生活瞬间点赞表索引
CREATE INDEX CONCURRENTLY idx_moment_likes_moment_id ON moment_likes(moment_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_moment_likes_user_id ON moment_likes(user_id) WHERE is_deleted = false;

-- 生活瞬间评论表索引
CREATE INDEX CONCURRENTLY idx_moment_comments_moment_id ON moment_comments(moment_id, created_at) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY idx_moment_comments_user_id ON moment_comments(user_id) WHERE is_deleted = false;

-- 举报表索引
CREATE INDEX CONCURRENTLY idx_reports_reporter ON reports(reporter_id, created_at DESC);
CREATE INDEX CONCURRENTLY idx_reports_target ON reports(target_type, target_id);
CREATE INDEX CONCURRENTLY idx_reports_status ON reports(status, created_at DESC);

-- 通知表索引
CREATE INDEX CONCURRENTLY idx_notifications_user_unread ON notifications(user_id, created_at DESC) WHERE is_read = false;
CREATE INDEX CONCURRENTLY idx_notifications_type ON notifications(type, created_at DESC);

-- ============================================================================
-- 11. 创建触发器函数
-- ============================================================================

-- 自动更新 updated_at 字段的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为需要的表创建 updated_at 触发器
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_group_members_updated_at
    BEFORE UPDATE ON group_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_moments_updated_at
    BEFORE UPDATE ON moments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 自动维护统计数据的触发器函数
CREATE OR REPLACE FUNCTION update_group_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- 处理消息插入
    IF TG_OP = 'INSERT' AND TG_TABLE_NAME = 'messages' THEN
        -- 维护群组最后活跃时间
        UPDATE groups 
        SET last_active_at = NEW.created_at,
            updated_at = NOW()
        WHERE id = NEW.group_id;
        
        -- 维护群组成员未读数量（除发送者外）
        UPDATE group_members 
        SET unread_count = unread_count + 1,
            updated_at = NOW()
        WHERE group_id = NEW.group_id 
          AND user_id != COALESCE(NEW.sender_id, 0)
          AND is_deleted = false
          AND status = 'active';
        
        RETURN NEW;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 创建统计维护触发器
CREATE TRIGGER trigger_update_group_stats_on_message
    AFTER INSERT ON messages
    FOR EACH ROW EXECUTE FUNCTION update_group_stats();

-- 软删除触发器函数
CREATE OR REPLACE FUNCTION soft_delete_trigger_function()
RETURNS TRIGGER AS $$
BEGIN
    NEW.deleted_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为需要软删除的表创建触发器
CREATE TRIGGER trigger_soft_delete_users
    BEFORE UPDATE OF is_deleted ON users
    FOR EACH ROW 
    WHEN (OLD.is_deleted = false AND NEW.is_deleted = true)
    EXECUTE FUNCTION soft_delete_trigger_function();

CREATE TRIGGER trigger_soft_delete_groups
    BEFORE UPDATE OF is_deleted ON groups
    FOR EACH ROW 
    WHEN (OLD.is_deleted = false AND NEW.is_deleted = true)
    EXECUTE FUNCTION soft_delete_trigger_function();

CREATE TRIGGER trigger_soft_delete_messages
    BEFORE UPDATE OF is_deleted ON messages
    FOR EACH ROW 
    WHEN (OLD.is_deleted = false AND NEW.is_deleted = true)
    EXECUTE FUNCTION soft_delete_trigger_function();

-- ============================================================================
-- 12. 创建业务函数
-- ============================================================================

-- 创建私聊群组函数
CREATE OR REPLACE FUNCTION create_private_chat_group(
    user1_id BIGINT,
    user2_id BIGINT
) RETURNS TEXT AS $$
DECLARE
    new_group_id TEXT;
    group_internal_id BIGINT;
BEGIN
    -- 检查是否已存在私聊群组
    SELECT g.group_id INTO new_group_id
    FROM groups g
    JOIN group_members gm1 ON g.id = gm1.group_id AND gm1.user_id = user1_id
    JOIN group_members gm2 ON g.id = gm2.group_id AND gm2.user_id = user2_id
    WHERE g.group_type = 'private_chat' 
      AND g.is_deleted = false
      AND gm1.is_deleted = false 
      AND gm2.is_deleted = false;
    
    -- 如果已存在，直接返回
    IF new_group_id IS NOT NULL THEN
        RETURN new_group_id;
    END IF;
    
    -- 创建新的私聊群组
    INSERT INTO groups (
        group_id, name, creator_id, group_type, max_members
    ) VALUES (
        generate_unique_id(), 
        NULL,  -- 私聊群组没有名称
        user1_id, 
        'private_chat', 
        2  -- 私聊最多2个成员
    ) RETURNING id, group_id INTO group_internal_id, new_group_id;
    
    -- 添加两个用户为成员
    INSERT INTO group_members (group_id, user_id, role, status) VALUES
        (group_internal_id, user1_id, 'member', 'active'),
        (group_internal_id, user2_id, 'member', 'active');
    
    RETURN new_group_id;
END;
$$ LANGUAGE plpgsql;

-- 发送统一消息函数
CREATE OR REPLACE FUNCTION send_unified_message(
    group_id_param TEXT,
    sender_id_param BIGINT,
    content_param TEXT,
    message_type_param message_type_enum DEFAULT 'text',
    reply_to_message_id_param BIGINT DEFAULT NULL,
    mentioned_users_param BIGINT[] DEFAULT '{}'
) RETURNS TEXT AS $$
DECLARE
    new_message_id TEXT;
    group_internal_id BIGINT;
BEGIN
    -- 获取群组内部ID
    SELECT id INTO group_internal_id
    FROM groups
    WHERE group_id = group_id_param AND is_deleted = false;
    
    IF group_internal_id IS NULL THEN
        RAISE EXCEPTION '群组不存在或已删除: %', group_id_param;
    END IF;
    
    -- 验证发送者是群组成员
    IF NOT EXISTS (
        SELECT 1 FROM group_members
        WHERE group_id = group_internal_id 
          AND user_id = sender_id_param 
          AND status = 'active' 
          AND can_send_messages = true
          AND is_deleted = false
    ) THEN
        RAISE EXCEPTION '用户无权限在此群组发送消息: %', sender_id_param;
    END IF;
    
    -- 插入消息
    INSERT INTO messages (
        message_id, group_id, sender_id, content, message_type, 
        reply_to_message_id, mentioned_users
    ) VALUES (
        generate_unique_id(), group_internal_id, sender_id_param, 
        content_param, message_type_param, reply_to_message_id_param, 
        mentioned_users_param
    ) RETURNING message_id INTO new_message_id;
    
    RETURN new_message_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 13. 创建用户类型视图
-- ============================================================================

-- 临时用户视图
CREATE VIEW temp_users AS
SELECT * FROM users WHERE get_user_type(id) = 1 AND is_deleted = false;

-- 普通用户视图
CREATE VIEW regular_users AS
SELECT * FROM users WHERE get_user_type(id) = 2 AND is_deleted = false;

-- 系统用户视图
CREATE VIEW system_users AS
SELECT * FROM users WHERE get_user_type(id) = 3 AND is_deleted = false;

-- ============================================================================
-- 14. 添加表注释
-- ============================================================================

COMMENT ON TABLE users IS '用户表 - 支持用户ID编码规范（高8位编码用户类型）';
COMMENT ON TABLE user_settings IS '用户设置表 - 管理用户隐私和通知偏好';
COMMENT ON TABLE user_activity IS '用户活动记录表 - 分区表，记录用户行为日志';
COMMENT ON TABLE user_relationships IS '用户关系表 - 管理好友、关注、拉黑等关系';
COMMENT ON TABLE groups IS '群组表 - 统一消息架构，包含公开群聊和私聊群组';
COMMENT ON TABLE group_members IS '群组成员表 - 管理所有群组成员关系，包括私聊参与者';
COMMENT ON TABLE messages IS '消息表 - 分区表，统一存储群聊和私聊消息';
COMMENT ON TABLE message_media IS '消息媒体附件表 - 支持多媒体消息';
COMMENT ON TABLE message_reactions IS '消息反应表 - 支持表情回应';
COMMENT ON TABLE moments IS '生活瞬间表 - 基于地理位置的动态分享';
COMMENT ON TABLE tags IS '标签表 - 生活瞬间标签系统';
COMMENT ON TABLE moment_tags IS '生活瞬间标签关联表';
COMMENT ON TABLE moment_likes IS '生活瞬间点赞表';
COMMENT ON TABLE moment_comments IS '生活瞬间评论表 - 支持多级回复';
COMMENT ON TABLE reports IS '举报表 - 内容安全和用户举报系统';
COMMENT ON TABLE notifications IS '通知表 - 系统通知和推送管理';

COMMENT ON FUNCTION generate_user_id(INTEGER) IS '生成带用户类型编码的64位用户ID';
COMMENT ON FUNCTION get_user_type(BIGINT) IS '从用户ID中提取用户类型';
COMMENT ON FUNCTION create_private_chat_group(BIGINT, BIGINT) IS '创建私聊群组，如果已存在则返回现有群组ID';
COMMENT ON FUNCTION send_unified_message(TEXT, BIGINT, TEXT, message_type_enum, BIGINT, BIGINT[]) IS '发送统一消息，支持群聊和私聊';

-- ============================================================================
-- 完成提示
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✅ HotSpot 完整数据库架构创建完成！';
    RAISE NOTICE '📊 统计信息：';
    RAISE NOTICE '   - 用户系统：4个表（users, user_settings, user_activity, user_relationships）';
    RAISE NOTICE '   - 群组系统：2个表（groups, group_members）';
    RAISE NOTICE '   - 消息系统：3个表（messages, message_media, message_reactions）';
    RAISE NOTICE '   - 瞬间系统：4个表（moments, tags, moment_tags, moment_likes）';
    RAISE NOTICE '   - 评论系统：1个表（moment_comments）';
    RAISE NOTICE '   - 安全系统：1个表（reports）';
    RAISE NOTICE '   - 通知系统：1个表（notifications）';
    RAISE NOTICE '   - 总计：16个核心表 + 分区表 + 视图';
    RAISE NOTICE '🚀 架构特性：';
    RAISE NOTICE '   - ✅ 统一消息架构（群聊和私聊使用相同表结构）';
    RAISE NOTICE '   - ✅ 用户ID编码规范（64位整数，高8位编码用户类型）';
    RAISE NOTICE '   - ✅ 软删除机制和自动触发器';
    RAISE NOTICE '   - ✅ 高性能索引和分区设计';
    RAISE NOTICE '   - ✅ PostGIS地理位置支持';
    RAISE NOTICE '   - ✅ 完整的业务函数和视图';
    RAISE NOTICE '💡 下一步：执行后端代码同步更新以使用新的数据库架构';
END $$;
