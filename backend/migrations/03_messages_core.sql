-- ============================================================================
-- HotSpot 消息核心模块迁移 v2.0
-- ============================================================================
-- 
-- 功能职责：
-- - 创建消息核心表结构
-- - 群组消息和私聊消息
-- - 消息媒体和附件
-- - 消息状态和已读回执
--
-- 依赖关系：
-- - 依赖 00_foundation.sql 的基础设施
-- - 依赖 01_users_core.sql 的用户系统
-- - 依赖 02_groups_core.sql 的群组系统
--
-- ============================================================================

-- 会话表（群组会话和私聊会话）
CREATE TABLE conversations (
    id BIGSERIAL PRIMARY KEY,
    conversation_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    
    -- 会话类型和关联
    conversation_type VARCHAR(20) NOT NULL, -- 'group', 'private'
    group_id BIGINT REFERENCES groups(id) ON DELETE CASCADE,
    
    -- 会话元数据
    name VARCHAR(200),
    description TEXT,
    avatar_url TEXT,
    
    -- 会话设置
    is_active BOOLEAN DEFAULT true,
    is_archived BOOLEAN DEFAULT false,
    message_retention_days INTEGER DEFAULT 365,
    
    -- 会话统计
    message_count INTEGER DEFAULT 0,
    participant_count INTEGER DEFAULT 0,
    
    -- 最后消息信息
    last_message_id BIGINT,
    last_message_at TIMESTAMP WITH TIME ZONE,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 创建者信息
    created_by BIGINT REFERENCES users(id),
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT conversations_type_check CHECK (conversation_type IN ('group', 'private')),
    CONSTRAINT conversations_group_consistency CHECK (
        (conversation_type = 'group' AND group_id IS NOT NULL) OR
        (conversation_type = 'private' AND group_id IS NULL)
    ),
    CONSTRAINT conversations_message_count_non_negative CHECK (message_count >= 0),
    CONSTRAINT conversations_participant_count_non_negative CHECK (participant_count >= 0)
);

-- 会话参与者表
CREATE TABLE conversation_participants (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 参与者角色和权限
    role VARCHAR(20) DEFAULT 'member', -- 'admin', 'member'
    can_send_messages BOOLEAN DEFAULT true,
    can_add_participants BOOLEAN DEFAULT false,
    
    -- 参与者状态
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'muted', 'left', 'removed'
    muted_until TIMESTAMP WITH TIME ZONE,
    
    -- 消息状态
    last_read_message_id BIGINT,
    last_read_at TIMESTAMP WITH TIME ZONE,
    unread_count INTEGER DEFAULT 0,
    
    -- 通知设置
    notifications_enabled BOOLEAN DEFAULT true,
    mention_notifications_enabled BOOLEAN DEFAULT true,
    
    -- 时间戳
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP WITH TIME ZONE,
    
    -- 唯一约束
    UNIQUE(conversation_id, user_id),
    
    -- 约束
    CONSTRAINT conversation_participants_unread_count_non_negative CHECK (unread_count >= 0)
);

-- 消息主表
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    message_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    
    -- 消息关联
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 消息内容
    message_type message_type_enum NOT NULL DEFAULT 'text',
    content TEXT,
    
    -- 消息元数据
    reply_to_message_id BIGINT REFERENCES messages(id),
    thread_id BIGINT REFERENCES messages(id),
    
    -- 消息状态
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false,
    deleted_for_everyone BOOLEAN DEFAULT false,
    
    -- 消息统计
    reaction_count INTEGER DEFAULT 0,
    reply_count INTEGER DEFAULT 0,
    
    -- 扩展数据
    metadata JSONB DEFAULT '{}',
    mentions BIGINT[] DEFAULT '{}',
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    CONSTRAINT messages_content_not_empty CHECK (
        (message_type = 'text' AND char_length(content) > 0) OR
        (message_type != 'text')
    ),
    CONSTRAINT messages_reaction_count_non_negative CHECK (reaction_count >= 0),
    CONSTRAINT messages_reply_count_non_negative CHECK (reply_count >= 0)
);

-- 消息媒体表
CREATE TABLE message_media (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    
    -- 媒体信息
    media_type VARCHAR(50) NOT NULL, -- 'image', 'video', 'audio', 'file', 'location'
    file_name VARCHAR(255),
    file_size BIGINT,
    mime_type VARCHAR(100),
    
    -- 存储信息
    file_url TEXT NOT NULL,
    thumbnail_url TEXT,
    
    -- 媒体元数据
    width INTEGER,
    height INTEGER,
    duration_seconds INTEGER,
    
    -- 位置信息（用于location类型）
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    address TEXT,
    
    -- 扩展元数据
    metadata JSONB DEFAULT '{}',
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT message_media_file_size_positive CHECK (file_size > 0),
    CONSTRAINT message_media_dimensions_positive CHECK (
        (width IS NULL OR width > 0) AND (height IS NULL OR height > 0)
    ),
    CONSTRAINT message_media_duration_positive CHECK (duration_seconds IS NULL OR duration_seconds > 0),
    CONSTRAINT message_media_location_coordinates CHECK (
        (media_type = 'location' AND latitude IS NOT NULL AND longitude IS NOT NULL AND 
         validate_coordinates(latitude, longitude)) OR
        (media_type != 'location')
    )
);

-- 消息反应表
CREATE TABLE message_reactions (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 反应信息
    reaction_type VARCHAR(50) NOT NULL, -- 'like', 'love', 'laugh', 'angry', 'sad', 'wow'
    emoji VARCHAR(10),
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束（每个用户对每条消息只能有一个反应）
    UNIQUE(message_id, user_id)
);

-- 消息已读状态表
CREATE TABLE message_read_status (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 已读状态
    read_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束
    UNIQUE(message_id, user_id)
);

-- ============================================================================
-- 索引创建
-- ============================================================================

-- 会话表索引
CREATE INDEX idx_conversations_conversation_id ON conversations(conversation_id);
CREATE INDEX idx_conversations_type ON conversations(conversation_type);
CREATE INDEX idx_conversations_group_id ON conversations(group_id);
CREATE INDEX idx_conversations_is_active ON conversations(is_active);
CREATE INDEX idx_conversations_last_activity ON conversations(last_activity_at);
CREATE INDEX idx_conversations_created_by ON conversations(created_by);

-- 会话参与者表索引
CREATE INDEX idx_conversation_participants_conversation_id ON conversation_participants(conversation_id);
CREATE INDEX idx_conversation_participants_user_id ON conversation_participants(user_id);
CREATE INDEX idx_conversation_participants_status ON conversation_participants(status);
CREATE INDEX idx_conversation_participants_unread_count ON conversation_participants(unread_count) WHERE unread_count > 0;

-- 消息表索引
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_message_type ON messages(message_type);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_reply_to ON messages(reply_to_message_id);
CREATE INDEX idx_messages_thread_id ON messages(thread_id);
CREATE INDEX idx_messages_is_deleted ON messages(is_deleted);
CREATE INDEX idx_messages_mentions ON messages USING GIN(mentions);

-- 消息媒体表索引
CREATE INDEX idx_message_media_message_id ON message_media(message_id);
CREATE INDEX idx_message_media_type ON message_media(media_type);
CREATE INDEX idx_message_media_file_size ON message_media(file_size);

-- 消息反应表索引
CREATE INDEX idx_message_reactions_message_id ON message_reactions(message_id);
CREATE INDEX idx_message_reactions_user_id ON message_reactions(user_id);
CREATE INDEX idx_message_reactions_type ON message_reactions(reaction_type);

-- 消息已读状态表索引
CREATE INDEX idx_message_read_status_message_id ON message_read_status(message_id);
CREATE INDEX idx_message_read_status_user_id ON message_read_status(user_id);
CREATE INDEX idx_message_read_status_read_at ON message_read_status(read_at);

-- ============================================================================
-- 触发器创建
-- ============================================================================

-- 会话表更新时间戳触发器
CREATE TRIGGER conversations_updated_at_trigger
    BEFORE UPDATE ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 会话参与者表更新时间戳触发器
CREATE TRIGGER conversation_participants_updated_at_trigger
    BEFORE UPDATE ON conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 消息表更新时间戳触发器
CREATE TRIGGER messages_updated_at_trigger
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 消息审计触发器
CREATE TRIGGER messages_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();

-- ============================================================================
-- 消息统计维护函数和触发器
-- ============================================================================

-- 更新会话消息数量函数
CREATE OR REPLACE FUNCTION update_conversation_message_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE conversations 
        SET message_count = message_count + 1,
            last_message_id = NEW.id,
            last_message_at = NEW.created_at,
            last_activity_at = NEW.created_at
        WHERE id = NEW.conversation_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE conversations 
        SET message_count = message_count - 1 
        WHERE id = OLD.conversation_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 会话消息数量维护触发器
CREATE TRIGGER conversation_message_count_trigger
    AFTER INSERT OR DELETE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_conversation_message_count();

-- 更新消息反应数量函数
CREATE OR REPLACE FUNCTION update_message_reaction_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE messages 
        SET reaction_count = reaction_count + 1 
        WHERE id = NEW.message_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE messages 
        SET reaction_count = reaction_count - 1 
        WHERE id = OLD.message_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 消息反应数量维护触发器
CREATE TRIGGER message_reaction_count_trigger
    AFTER INSERT OR DELETE ON message_reactions
    FOR EACH ROW
    EXECUTE FUNCTION update_message_reaction_count();

-- 更新未读消息数量函数
CREATE OR REPLACE FUNCTION update_unread_message_count()
RETURNS TRIGGER AS $$
DECLARE
    participant_record RECORD;
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- 为所有会话参与者增加未读数量（除了发送者）
        FOR participant_record IN 
            SELECT user_id FROM conversation_participants 
            WHERE conversation_id = NEW.conversation_id 
            AND user_id != NEW.sender_id 
            AND status = 'active'
        LOOP
            UPDATE conversation_participants 
            SET unread_count = unread_count + 1 
            WHERE conversation_id = NEW.conversation_id 
            AND user_id = participant_record.user_id;
        END LOOP;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 未读消息数量维护触发器
CREATE TRIGGER unread_message_count_trigger
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_unread_message_count();
