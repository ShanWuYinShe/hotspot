-- ============================================================
-- 005_chat_module.sql
-- 聊天模块：私聊会话、群聊消息、反应、删除
-- ============================================================

-- ─── 私聊会话 ───────────────────────────────────────────────
-- user1_id < user2_id 保证唯一性
CREATE TABLE IF NOT EXISTS private_chats (
    chat_id             UUID        PRIMARY KEY DEFAULT uuidv7(),
    user1_id            BIGINT      NOT NULL,
    user2_id            BIGINT      NOT NULL,
    -- 最新消息快照（反规范化，支撑会话列表展示）
    last_message_id     UUID,
    last_message_content TEXT,                       -- 前 50 字符预览
    last_message_type   VARCHAR(20),
    last_message_sender_id BIGINT,
    last_message_at     TIMESTAMPTZ,
    -- 单方删除时间线
    user1_cleared_at    TIMESTAMPTZ,                -- user1 删除会话的时间点
    user2_cleared_at    TIMESTAMPTZ,                -- user2 删除会话的时间点
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (user1_id < user2_id)
);
-- 保证双方唯一会话
CREATE UNIQUE INDEX idx_private_chats_pair ON private_chats(user1_id, user2_id);
-- 查某用户的会话列表
CREATE INDEX idx_private_chats_u1 ON private_chats(user1_id, updated_at DESC);
CREATE INDEX idx_private_chats_u2 ON private_chats(user2_id, updated_at DESC);

-- ─── 统一会话状态（私聊 + 群聊）──────────────────────────
-- chat_type: 'private' / 'group'
-- chat_id: 对应 private_chats.chat_id (UUID as text) 或 groups.group_id (BIGINT as text)
CREATE TABLE IF NOT EXISTS chat_conversations_state (
    user_id             BIGINT      NOT NULL,
    chat_type           VARCHAR(20) NOT NULL,
    chat_id             VARCHAR(50) NOT NULL,
    unread_count        INTEGER     NOT NULL DEFAULT 0,
    last_read_message_id UUID,
    is_pinned           BOOLEAN     NOT NULL DEFAULT FALSE,
    is_muted            BOOLEAN     NOT NULL DEFAULT FALSE,
    is_hidden           BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, chat_type, chat_id)
) WITH (fillfactor = 70);

-- 查某用户的全部会话列表（覆盖索引，避免回表）
CREATE INDEX idx_chat_state_user ON chat_conversations_state(user_id, updated_at DESC)
    INCLUDE (chat_type, chat_id, unread_count, is_pinned, is_muted, is_hidden);

-- ─── 私聊消息（分区）────────────────────────────────────────
CREATE TABLE IF NOT EXISTS private_messages_partitioned (
    message_id      UUID        NOT NULL DEFAULT uuidv7(),
    chat_id         UUID        NOT NULL,
    sender_id       BIGINT      NOT NULL,
    receiver_id     BIGINT      NOT NULL,
    message_type    VARCHAR(20) NOT NULL,
    content         TEXT,
    metadata        JSONB,                          -- 媒体信息 { url, thumbnail_url, file_name, file_size, ... }
    share_code      VARCHAR(16),
    reply_to_id     UUID,
    is_recalled     BOOLEAN     NOT NULL DEFAULT FALSE,
    recalled_at     TIMESTAMPTZ,
    edited_at       TIMESTAMPTZ,
    reaction_counts JSONB       NOT NULL DEFAULT '{}',  -- { "love": 2, "laugh": 1 }
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, sent_at)
) PARTITION BY RANGE (sent_at);

CREATE TABLE IF NOT EXISTS private_messages_default PARTITION OF private_messages_partitioned DEFAULT;

CREATE INDEX idx_pm_chat_time ON private_messages_partitioned(chat_id, sent_at DESC);

-- ─── 群聊消息（分区）────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_messages_partitioned (
    message_id      UUID        NOT NULL DEFAULT uuidv7(),
    group_id        BIGINT      NOT NULL,
    sender_id       BIGINT      NOT NULL,
    message_type    VARCHAR(20) NOT NULL,
    content         TEXT,
    metadata        JSONB,
    share_code      VARCHAR(16),
    reply_to_id     UUID,
    at_users        JSONB       NOT NULL DEFAULT '[]'::jsonb,  -- [ user_id, ... ]
    is_recalled     BOOLEAN     NOT NULL DEFAULT FALSE,
    recalled_at     TIMESTAMPTZ,
    recalled_by     BIGINT,
    edited_at       TIMESTAMPTZ,
    reaction_counts JSONB       NOT NULL DEFAULT '{}',
    read_count      INTEGER     NOT NULL DEFAULT 0,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, sent_at)
) PARTITION BY RANGE (sent_at);

CREATE TABLE IF NOT EXISTS group_messages_default PARTITION OF group_messages_partitioned DEFAULT;

CREATE INDEX idx_gm_group_time ON group_messages_partitioned(group_id, sent_at DESC);

-- ─── 私聊消息反应 ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS private_message_reactions (
    message_id      UUID            NOT NULL,
    user_id         BIGINT          NOT NULL,
    reaction_type   reaction_type   NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, reaction_type)
);
CREATE INDEX idx_pm_reactions_list ON private_message_reactions(message_id, created_at DESC);

-- ─── 群聊消息反应 ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_message_reactions (
    message_id      UUID            NOT NULL,
    user_id         BIGINT          NOT NULL,
    reaction_type   reaction_type   NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, reaction_type)
);
CREATE INDEX idx_gm_reactions_list ON group_message_reactions(message_id, created_at DESC);

-- ─── 私聊消息删除（单方视角）──────────────────────────────
CREATE TABLE IF NOT EXISTS private_message_deletions (
    message_id  UUID        NOT NULL,
    user_id     BIGINT      NOT NULL,
    deleted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);
CREATE INDEX idx_pm_deletions_user ON private_message_deletions(user_id, deleted_at DESC);

-- ─── 群聊消息删除（单方视角）──────────────────────────────
CREATE TABLE IF NOT EXISTS group_message_deletions (
    message_id  UUID        NOT NULL,
    user_id     BIGINT      NOT NULL,
    deleted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);
CREATE INDEX idx_gm_deletions_user ON group_message_deletions(user_id, deleted_at DESC);

-- ─── 消息反应计数聚合 ──────────────────────────────────────
-- scope: 'private' / 'group'，与消息表对应
CREATE TABLE IF NOT EXISTS message_reaction_counts (
    scope           VARCHAR(10)     NOT NULL,
    message_id      UUID            NOT NULL,
    reaction_type   reaction_type   NOT NULL,
    count           INTEGER         NOT NULL DEFAULT 0,
    PRIMARY KEY (scope, message_id, reaction_type)
);
CREATE INDEX idx_msg_reaction_counts ON message_reaction_counts(scope, message_id);
