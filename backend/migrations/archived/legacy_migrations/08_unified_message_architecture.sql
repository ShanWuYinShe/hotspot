-- ============================================================================
-- HotSpot 统一消息架构迁移
-- ============================================================================
-- 
-- 功能职责：
-- - 实现统一消息架构，群聊和私聊使用相同的底层实现
-- - 创建统一的消息表和群组表
-- - 私聊也通过群组实现，但设置为不可搜索类型
--
-- 依赖关系：
-- - 依赖现有的表结构
-- - 依赖用户ID编码规范
--
-- ============================================================================

-- 1. 创建群组类型枚举
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'group_type_enum') THEN
        CREATE TYPE group_type_enum AS ENUM ('public_chat', 'private_chat');
    END IF;
END $$;

-- 2. 创建成员角色枚举
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'member_role_enum') THEN
        CREATE TYPE member_role_enum AS ENUM ('owner', 'admin', 'member');
    END IF;
END $$;

-- 3. 创建成员状态枚举
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'member_status_enum') THEN
        CREATE TYPE member_status_enum AS ENUM ('active', 'muted', 'banned', 'left');
    END IF;
END $$;

-- 4. 创建消息类型枚举
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'message_type_enum') THEN
        CREATE TYPE message_type_enum AS ENUM ('text', 'image', 'video', 'audio', 'file', 'location', 'system');
    END IF;
END $$;

-- 5. 创建统一群组表（包含私聊群组）
CREATE TABLE IF NOT EXISTS unified_groups (
    id BIGSERIAL PRIMARY KEY,
    group_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    name TEXT,                                  -- 群组名称（私聊群组可为NULL）
    description TEXT,                           -- 群组描述
    avatar_url TEXT,                            -- 群组头像URL
    creator_id BIGINT NOT NULL,                 -- 创建者ID（使用编码用户ID）
    owner_id BIGINT,                            -- 群主ID（私聊群组可为NULL）
    latitude DOUBLE PRECISION,                  -- 纬度（私聊群组可为NULL）
    longitude DOUBLE PRECISION,                 -- 经度（私聊群组可为NULL）
    location GEOGRAPHY(POINT, 4326),           -- PostGIS地理位置点（私聊群组可为NULL）
    location_name TEXT,                         -- 位置名称（私聊群组可为NULL）
    group_type group_type_enum NOT NULL DEFAULT 'public_chat', -- 群组类型
    max_members INTEGER DEFAULT 500,            -- 最大成员数
    password_hash TEXT,                         -- 群组密码哈希（可选）
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否活跃
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 6. 创建统一群组成员表（包含私聊参与者）
CREATE TABLE IF NOT EXISTS unified_group_members (
    id BIGSERIAL PRIMARY KEY,
    member_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    group_id BIGINT NOT NULL REFERENCES unified_groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,                    -- 用户ID（使用编码用户ID）
    role member_role_enum NOT NULL DEFAULT 'member',
    status member_status_enum NOT NULL DEFAULT 'active',
    permissions TEXT[] DEFAULT '{}',
    can_send_messages BOOLEAN NOT NULL DEFAULT true,
    can_add_participants BOOLEAN NOT NULL DEFAULT false,
    muted_until TIMESTAMP WITH TIME ZONE,
    last_read_message_id BIGINT,
    last_read_at TIMESTAMP WITH TIME ZONE,
    unread_count INTEGER NOT NULL DEFAULT 0,
    notifications_enabled BOOLEAN NOT NULL DEFAULT true,
    mention_notifications BOOLEAN NOT NULL DEFAULT true,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    invited_by BIGINT,                          -- 邀请者ID
    
    UNIQUE(group_id, user_id)
);

-- 7. 创建统一消息表（支持分区）
CREATE TABLE IF NOT EXISTS unified_messages (
    id BIGSERIAL,
    message_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    group_id BIGINT NOT NULL REFERENCES unified_groups(id) ON DELETE CASCADE,
    sender_id BIGINT,                           -- 发送者ID（系统消息可为NULL）
    content TEXT NOT NULL,
    message_type message_type_enum NOT NULL DEFAULT 'text',
    system_event_type TEXT,                     -- 系统事件类型
    reply_to_message_id BIGINT,                 -- 回复的消息ID
    thread_id BIGINT,                           -- 话题ID
    is_edited BOOLEAN NOT NULL DEFAULT false,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    deleted_for_everyone BOOLEAN NOT NULL DEFAULT false,
    reaction_count INTEGER NOT NULL DEFAULT 0,
    reply_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    mentioned_users BIGINT[] DEFAULT '{}',      -- 提及的用户ID数组
    affected_users BIGINT[] DEFAULT '{}',       -- 受影响的用户ID数组（系统消息用）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    PRIMARY KEY (id, created_at)  -- 复合主键支持分区
) PARTITION BY RANGE (created_at);

-- 8. 创建消息表分区（按月分区）
CREATE TABLE IF NOT EXISTS unified_messages_2024_01 PARTITION OF unified_messages
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE IF NOT EXISTS unified_messages_2024_02 PARTITION OF unified_messages
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE IF NOT EXISTS unified_messages_2024_03 PARTITION OF unified_messages
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE IF NOT EXISTS unified_messages_2024_04 PARTITION OF unified_messages
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

-- 9. 创建索引
-- 群组表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_groups_group_id ON unified_groups(group_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_groups_creator ON unified_groups(creator_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_groups_location_active ON unified_groups USING GIST(location) WHERE is_deleted = false AND is_active = true;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_groups_type_active ON unified_groups(group_type, is_active, last_active_at DESC) WHERE is_deleted = false;

-- 群组成员表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_group_members_member_id ON unified_group_members(member_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_group_members_group_user ON unified_group_members(group_id, user_id) WHERE is_deleted = false;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_group_members_user_active ON unified_group_members(user_id, status, joined_at DESC) WHERE is_deleted = false AND status = 'active';
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_group_members_unread ON unified_group_members(user_id, unread_count DESC) WHERE is_deleted = false AND unread_count > 0;

-- 消息表索引（每个分区都需要）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_messages_group_time_covering 
ON unified_messages (group_id, created_at DESC) 
INCLUDE (id, message_id, sender_id, content, message_type) 
WHERE is_deleted = false;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_messages_sender_time 
ON unified_messages (sender_id, created_at DESC) 
WHERE sender_id IS NOT NULL AND is_deleted = false;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_unified_messages_message_id 
ON unified_messages (message_id) 
WHERE is_deleted = false;

-- 10. 创建私聊群组创建函数
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
    FROM unified_groups g
    JOIN unified_group_members gm1 ON g.id = gm1.group_id AND gm1.user_id = user1_id
    JOIN unified_group_members gm2 ON g.id = gm2.group_id AND gm2.user_id = user2_id
    WHERE g.group_type = 'private_chat' 
      AND g.is_deleted = false
      AND gm1.is_deleted = false 
      AND gm2.is_deleted = false;
    
    -- 如果已存在，直接返回
    IF new_group_id IS NOT NULL THEN
        RETURN new_group_id;
    END IF;
    
    -- 创建新的私聊群组
    INSERT INTO unified_groups (
        group_id, name, creator_id, group_type, max_members
    ) VALUES (
        generate_unique_id(), 
        NULL,  -- 私聊群组没有名称
        user1_id, 
        'private_chat', 
        2  -- 私聊最多2个成员
    ) RETURNING id, group_id INTO group_internal_id, new_group_id;
    
    -- 添加两个用户为成员
    INSERT INTO unified_group_members (group_id, user_id, role, status) VALUES
        (group_internal_id, user1_id, 'member', 'active'),
        (group_internal_id, user2_id, 'member', 'active');
    
    RETURN new_group_id;
END;
$$ LANGUAGE plpgsql;

-- 11. 创建消息发送函数
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
    FROM unified_groups
    WHERE group_id = group_id_param AND is_deleted = false;
    
    IF group_internal_id IS NULL THEN
        RAISE EXCEPTION '群组不存在或已删除: %', group_id_param;
    END IF;
    
    -- 验证发送者是群组成员
    IF NOT EXISTS (
        SELECT 1 FROM unified_group_members
        WHERE group_id = group_internal_id 
          AND user_id = sender_id_param 
          AND status = 'active' 
          AND can_send_messages = true
          AND is_deleted = false
    ) THEN
        RAISE EXCEPTION '用户无权限在此群组发送消息: %', sender_id_param;
    END IF;
    
    -- 插入消息
    INSERT INTO unified_messages (
        message_id, group_id, sender_id, content, message_type, 
        reply_to_message_id, mentioned_users
    ) VALUES (
        generate_unique_id(), group_internal_id, sender_id_param, 
        content_param, message_type_param, reply_to_message_id_param, 
        mentioned_users_param
    ) RETURNING message_id INTO new_message_id;
    
    -- 更新群组最后活跃时间
    UPDATE unified_groups 
    SET last_active_at = NOW(), updated_at = NOW()
    WHERE id = group_internal_id;
    
    -- 更新其他成员的未读数（除发送者外）
    UPDATE unified_group_members 
    SET unread_count = unread_count + 1, updated_at = NOW()
    WHERE group_id = group_internal_id 
      AND user_id != sender_id_param
      AND status = 'active'
      AND is_deleted = false;
    
    RETURN new_message_id;
END;
$$ LANGUAGE plpgsql;

-- 12. 创建获取会话列表函数
CREATE OR REPLACE FUNCTION get_user_conversations(
    user_id_param BIGINT,
    limit_param INTEGER DEFAULT 20,
    offset_param INTEGER DEFAULT 0
) RETURNS TABLE (
    group_id TEXT,
    group_name TEXT,
    group_type group_type_enum,
    group_avatar TEXT,
    last_message_id TEXT,
    last_message_content TEXT,
    last_message_sender_id BIGINT,
    last_message_at TIMESTAMP WITH TIME ZONE,
    unread_count INTEGER,
    member_count BIGINT,
    is_muted BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        g.group_id,
        g.name as group_name,
        g.group_type,
        g.avatar_url as group_avatar,
        lm.message_id as last_message_id,
        lm.content as last_message_content,
        lm.sender_id as last_message_sender_id,
        lm.created_at as last_message_at,
        gm.unread_count,
        (SELECT COUNT(*) FROM unified_group_members WHERE group_id = g.id AND is_deleted = false AND status = 'active') as member_count,
        (gm.muted_until IS NOT NULL AND gm.muted_until > NOW()) as is_muted
    FROM unified_groups g
    JOIN unified_group_members gm ON g.id = gm.group_id
    LEFT JOIN LATERAL (
        SELECT message_id, content, sender_id, created_at
        FROM unified_messages
        WHERE group_id = g.id AND is_deleted = false
        ORDER BY created_at DESC
        LIMIT 1
    ) lm ON true
    WHERE gm.user_id = user_id_param
      AND gm.status = 'active'
      AND gm.is_deleted = false
      AND g.is_deleted = false
    ORDER BY COALESCE(lm.created_at, g.created_at) DESC
    LIMIT limit_param OFFSET offset_param;
END;
$$ LANGUAGE plpgsql;

-- 13. 创建数据迁移函数（从旧表迁移到新表）
CREATE OR REPLACE FUNCTION migrate_to_unified_architecture()
RETURNS VOID AS $$
DECLARE
    group_record RECORD;
    message_record RECORD;
    new_group_id BIGINT;
BEGIN
    -- 迁移现有群组到统一群组表
    FOR group_record IN 
        SELECT * FROM groups WHERE deleted_at IS NULL
    LOOP
        INSERT INTO unified_groups (
            group_id, name, description, avatar_url, creator_id, owner_id,
            latitude, longitude, location, location_name, group_type,
            max_members, is_active, created_at, updated_at, last_active_at
        ) VALUES (
            group_record.group_id, group_record.name, group_record.description,
            group_record.avatar_url, group_record.creator_id, group_record.owner_id,
            group_record.latitude, group_record.longitude, group_record.location,
            group_record.location_name, 'public_chat', group_record.max_members,
            group_record.is_active, group_record.created_at, group_record.updated_at,
            group_record.last_active_at
        ) RETURNING id INTO new_group_id;
        
        -- 迁移群组成员
        INSERT INTO unified_group_members (
            group_id, user_id, role, status, can_send_messages, 
            notifications_enabled, joined_at, updated_at
        )
        SELECT 
            new_group_id, user_id, 
            CASE role 
                WHEN 'owner' THEN 'owner'::member_role_enum
                WHEN 'admin' THEN 'admin'::member_role_enum
                ELSE 'member'::member_role_enum
            END,
            CASE status
                WHEN 'active' THEN 'active'::member_status_enum
                WHEN 'muted' THEN 'muted'::member_status_enum
                WHEN 'banned' THEN 'banned'::member_status_enum
                ELSE 'left'::member_status_enum
            END,
            can_send_messages, notifications_enabled, joined_at, updated_at
        FROM group_members 
        WHERE group_id = group_record.id AND deleted_at IS NULL;
    END LOOP;
    
    RAISE NOTICE '统一消息架构迁移完成';
END;
$$ LANGUAGE plpgsql;

-- 14. 添加注释
COMMENT ON TABLE unified_groups IS '统一群组表，包含公开群聊和私聊群组';
COMMENT ON TABLE unified_group_members IS '统一群组成员表，包含群聊成员和私聊参与者';
COMMENT ON TABLE unified_messages IS '统一消息表，支持群聊和私聊消息';
COMMENT ON FUNCTION create_private_chat_group(BIGINT, BIGINT) IS '创建私聊群组，如果已存在则返回现有群组ID';
COMMENT ON FUNCTION send_unified_message(TEXT, BIGINT, TEXT, message_type_enum, BIGINT, BIGINT[]) IS '发送统一消息，支持群聊和私聊';
COMMENT ON FUNCTION get_user_conversations(BIGINT, INTEGER, INTEGER) IS '获取用户的会话列表，包含群聊和私聊';
COMMENT ON FUNCTION migrate_to_unified_architecture() IS '将现有数据迁移到统一消息架构';

RAISE NOTICE '统一消息架构已安装完成';
