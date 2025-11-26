-- 004_message_module.sql
-- 消息模块：私聊消息、群聊消息、已读状态、消息反应

-- ============================================================
-- 私聊消息
-- ============================================================
CREATE TABLE private_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    private_chat_id UUID NOT NULL,
    sender_id BIGINT NOT NULL,
    type message_type NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    reply_to_message_id UUID,
    file_name VARCHAR(255),
    file_size BIGINT,
    thumbnail_url TEXT,
    duration INTEGER,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ,
    is_recalled BOOLEAN NOT NULL DEFAULT FALSE,
    recalled_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_private_messages_cursor_id ON private_messages (cursor_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_messages_chat_id ON private_messages (private_chat_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_messages_sender ON private_messages (sender_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_messages_content_search ON private_messages USING gin (content gin_trgm_ops) WHERE type = 'text' AND is_deleted = FALSE;

-- ============================================================
-- 群聊消息
-- ============================================================
CREATE TABLE group_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    sender_id BIGINT NOT NULL,
    type message_type NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    reply_to_message_id UUID,
    mentions BIGINT[],
    file_name VARCHAR(255),
    file_size BIGINT,
    thumbnail_url TEXT,
    duration INTEGER,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ,
    is_recalled BOOLEAN NOT NULL DEFAULT FALSE,
    recalled_by BIGINT,
    recalled_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_group_messages_cursor_id ON group_messages (cursor_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_messages_chat_id ON group_messages (group_chat_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_messages_sender ON group_messages (sender_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_messages_mentions ON group_messages USING gin (mentions) WHERE mentions IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_group_messages_content_search ON group_messages USING gin (content gin_trgm_ops) WHERE type = 'text' AND is_deleted = FALSE;

-- ============================================================
-- 群聊消息已读状态
-- ============================================================
CREATE TABLE group_message_read_status (
    id BIGSERIAL PRIMARY KEY,
    read_status_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL,
    group_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_message_read_status UNIQUE (message_id, user_id)
);

CREATE INDEX idx_message_read_status_id ON group_message_read_status (read_status_id);
CREATE INDEX idx_message_read_status_cursor_id ON group_message_read_status (cursor_id);
CREATE INDEX idx_message_read_status_message_id ON group_message_read_status (message_id, read_at DESC);
CREATE INDEX idx_message_read_status_user ON group_message_read_status (user_id, read_at DESC);
CREATE INDEX idx_message_read_status_group ON group_message_read_status (group_chat_id, user_id);

-- ============================================================
-- 消息反应
-- ============================================================
CREATE TABLE message_reactions (
    id BIGSERIAL PRIMARY KEY,
    reaction_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    type reaction_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_message_reaction UNIQUE (message_id, user_id, type)
);

CREATE INDEX idx_message_reactions_id ON message_reactions (reaction_id);
CREATE INDEX idx_message_reactions_cursor_id ON message_reactions (cursor_id);
CREATE INDEX idx_message_reactions_message_id ON message_reactions (message_id, created_at DESC);
CREATE INDEX idx_message_reactions_user ON message_reactions (user_id, created_at DESC);


