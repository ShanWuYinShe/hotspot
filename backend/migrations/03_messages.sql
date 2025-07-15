-- 消息系统相关表
-- 包含: conversations, direct_messages, user_message_read_status

-- 私聊会话表
CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    conversation_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user1_id BIGINT NOT NULL,
    user2_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_preview TEXT,
    
    CONSTRAINT conversations_user_order CHECK (user1_id < user2_id),
    CONSTRAINT conversations_users_unique UNIQUE (user1_id, user2_id),
    CONSTRAINT fk_conversations_user1_id 
        FOREIGN KEY (user1_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_conversations_user2_id 
        FOREIGN KEY (user2_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_conversations_user1_id ON conversations(user1_id);
CREATE INDEX idx_conversations_user2_id ON conversations(user2_id);
CREATE INDEX idx_conversations_last_message_at ON conversations(last_message_at);

-- 私聊消息表
CREATE TABLE IF NOT EXISTS direct_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL,
    sender_id BIGINT NOT NULL,
    recipient_id BIGINT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ,
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.16,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.68,
    geom GEOGRAPHY(POINT, 4326),
    
    CONSTRAINT fk_direct_messages_conversation_id 
        FOREIGN KEY (conversation_id) REFERENCES conversations(conversation_id) ON DELETE CASCADE,
    CONSTRAINT fk_direct_messages_sender_id 
        FOREIGN KEY (sender_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_direct_messages_recipient_id 
        FOREIGN KEY (recipient_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE TRIGGER update_direct_messages_geom
BEFORE INSERT OR UPDATE OF latitude, longitude ON direct_messages
FOR EACH ROW EXECUTE FUNCTION update_geom_column();

CREATE INDEX idx_direct_messages_conversation_id ON direct_messages(conversation_id);
CREATE INDEX idx_direct_messages_created_at ON direct_messages(created_at);
CREATE INDEX idx_direct_messages_sender_id ON direct_messages(sender_id);
CREATE INDEX idx_direct_messages_recipient_id ON direct_messages(recipient_id);
CREATE INDEX idx_direct_messages_read_at ON direct_messages(read_at);
CREATE INDEX idx_direct_messages_geom ON direct_messages USING gist (geom);

-- 用户消息已读状态表
-- 记录每个用户在每个会话/群组中最后已读的消息位置
CREATE TABLE IF NOT EXISTS user_message_read_status (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    
    -- 对于私聊，使用conversation_id
    conversation_id UUID,
    -- 对于群聊，使用group_id  
    group_id UUID,
    
    -- 最后已读消息的ID（消息表的自增ID，不是UUID）
    last_read_message_id BIGINT NOT NULL,
    
    -- 最后更新时间
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_user_message_read_status_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_message_read_status_conversation_id 
        FOREIGN KEY (conversation_id) REFERENCES conversations(conversation_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_message_read_status_group_id 
        FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE,
    
    -- 确保每个用户在每个会话/群组中只有一条记录
    CONSTRAINT unique_user_conversation 
        UNIQUE (user_id, conversation_id),
    CONSTRAINT unique_user_group 
        UNIQUE (user_id, group_id),
    
    -- 确保不能同时设置conversation_id和group_id
    CONSTRAINT check_conversation_or_group 
        CHECK ((conversation_id IS NOT NULL AND group_id IS NULL) OR 
               (conversation_id IS NULL AND group_id IS NOT NULL))
);

-- 创建索引
CREATE INDEX idx_user_message_read_status_user_id ON user_message_read_status(user_id);
CREATE INDEX idx_user_message_read_status_conversation_id ON user_message_read_status(conversation_id);
CREATE INDEX idx_user_message_read_status_group_id ON user_message_read_status(group_id);
CREATE INDEX idx_user_message_read_status_updated_at ON user_message_read_status(updated_at);

-- 创建更新时间的触发器
CREATE OR REPLACE FUNCTION update_user_message_read_status_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_message_read_status_updated_at
    BEFORE UPDATE ON user_message_read_status
    FOR EACH ROW
    EXECUTE FUNCTION update_user_message_read_status_updated_at();
