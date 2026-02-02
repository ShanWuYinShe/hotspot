-- 003_private_chat_module.sql
-- 私聊模块：私聊会话、私聊设置、私聊消息

-- ============================================================
-- 私聊会话
-- 说明：包含陌生人消息限制机制
-- - 互相关注（好友）：无限制
-- - 对方回复过：无限制（can_message_freely = TRUE）
-- - 单向关注 + 对方未回复：3条
-- - 未关注 + 对方未回复：1条
-- ============================================================
CREATE TABLE private_chats (
    id BIGSERIAL PRIMARY KEY,
    -- 移除了 private_chat_id (UUID)，仅使用内部 ID (BIGINT)
    user_a_id BIGINT NOT NULL,
    user_b_id BIGINT NOT NULL,
    -- 陌生人消息限制字段
    user_a_can_message_freely BOOLEAN NOT NULL DEFAULT FALSE,  -- user_a 是否可以无限发送消息
    user_b_can_message_freely BOOLEAN NOT NULL DEFAULT FALSE,  -- user_b 是否可以无限发送消息
    user_a_sent_count INTEGER NOT NULL DEFAULT 0,              -- user_a 已发送消息数（限制期间计数）
    user_b_sent_count INTEGER NOT NULL DEFAULT 0,              -- user_b 已发送消息数（限制期间计数）
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_private_chat_user_order CHECK (user_a_id < user_b_id),
    CONSTRAINT unique_private_chat_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_private_chats_user_a ON private_chats (user_a_id, updated_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_chats_user_b ON private_chats (user_b_id, updated_at DESC) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_private_chats_updated_at
    BEFORE UPDATE ON private_chats FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 私聊设置（每个用户对每个私聊的个人设置）
-- ============================================================
CREATE TABLE private_chat_settings (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL, -- 引用内部 ID
    user_id BIGINT NOT NULL,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT unique_private_chat_user_settings UNIQUE (chat_id, user_id)
);

CREATE INDEX idx_private_chat_settings_user ON private_chat_settings (user_id);
CREATE INDEX idx_private_chat_settings_chat ON private_chat_settings (chat_id);

CREATE TRIGGER trigger_private_chat_settings_updated_at
    BEFORE UPDATE ON private_chat_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 私聊消息
-- ============================================================
CREATE TABLE private_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL, -- 引用内部 ID
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

CREATE INDEX idx_private_messages_chat_id ON private_messages (chat_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_messages_sender ON private_messages (sender_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_messages_content_search ON private_messages USING gin (content gin_trgm_ops) WHERE type = 'text' AND is_deleted = FALSE;

-- ============================================================
-- 私聊消息反应
-- ============================================================
CREATE TABLE private_message_reactions (
    id BIGSERIAL PRIMARY KEY,
    reaction_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    type reaction_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_private_message_reaction UNIQUE (message_id, user_id, type)
);

CREATE INDEX idx_private_message_reactions_id ON private_message_reactions (reaction_id);
CREATE INDEX idx_private_message_reactions_message_id ON private_message_reactions (message_id, created_at DESC);
CREATE INDEX idx_private_message_reactions_user ON private_message_reactions (user_id, created_at DESC);
