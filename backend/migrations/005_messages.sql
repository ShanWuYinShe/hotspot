-- 消息表（统一私聊和群聊消息）
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES chat_rooms(room_id) ON DELETE CASCADE,
    sender_id BIGINT REFERENCES users(user_id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    message_type message_type NOT NULL DEFAULT 'text'::message_type CHECK (message_type != 'any'::message_type),
    is_edited BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    edited_at TIMESTAMP WITH TIME ZONE,
    reply_to UUID REFERENCES messages(message_id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'sent' NOT NULL CHECK (status IN ('sent', 'delivered', 'read', 'deleted')),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    mentions BIGINT[],
    file_name VARCHAR(255),
    file_size BIGINT,
    thumbnail_url TEXT,
    duration INTEGER,
    system_event VARCHAR(50),
    affected_users BIGINT[],
    CONSTRAINT messages_deleted_check CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL)
    )
);

-- 消息索引
CREATE INDEX idx_messages_id ON messages(id DESC);
CREATE UNIQUE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_room ON messages(room_id, id DESC);
CREATE INDEX idx_messages_sender ON messages(sender_id, id DESC);
CREATE INDEX idx_messages_reply ON messages(reply_to);
CREATE INDEX idx_messages_created ON messages(created_at DESC);
CREATE INDEX idx_messages_not_deleted ON messages(is_deleted) WHERE is_deleted = FALSE;

-- 消息已读记录表（简化版，因为未读数已在room_members中）
CREATE TABLE message_read_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES chat_rooms(room_id) ON DELETE CASCADE,
    last_read_message_id UUID REFERENCES messages(message_id) ON DELETE SET NULL,
    read_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (user_id, room_id)
);

-- 已读记录索引
CREATE INDEX idx_message_read_id ON message_read_records(id DESC);
CREATE INDEX idx_message_read_user_room ON message_read_records(user_id, room_id);

-- 消息反应表
CREATE TABLE message_reactions (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    reaction_type VARCHAR(20) NOT NULL CHECK (reaction_type IN ('like', 'love', 'laugh', 'angry', 'sad', 'wow')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (message_id, user_id, reaction_type)
);

-- 反应索引
CREATE INDEX idx_message_reactions_id ON message_reactions(id DESC);
CREATE INDEX idx_message_reactions_message ON message_reactions(message_id);
CREATE INDEX idx_message_reactions_user ON message_reactions(user_id);

-- 消息已读状态表（记录每条消息对每个用户的已读状态）
CREATE TABLE message_reads (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    is_read BOOLEAN DEFAULT FALSE NOT NULL,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (message_id, user_id)
);

-- 消息已读状态索引
CREATE INDEX idx_message_reads_id ON message_reads(id DESC);
CREATE INDEX idx_message_reads_user ON message_reads(user_id, is_read);
CREATE INDEX idx_message_reads_message ON message_reads(message_id);
CREATE INDEX idx_message_reads_user_created ON message_reads(user_id, created_at DESC);

