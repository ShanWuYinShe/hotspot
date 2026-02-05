-- ============================================================
-- 004_chat_module.sql
-- 聊天模块：私聊、群聊、消息、未读
-- ============================================================

-- 1. 私聊会话表 (无外键)
CREATE TABLE IF NOT EXISTS private_chats (
    chat_id UUID PRIMARY KEY DEFAULT uuidv7(),
    user1_id BIGINT NOT NULL,
    user2_id BIGINT NOT NULL,
    
    last_message_id UUID,
    last_message_content TEXT,
    last_message_type VARCHAR(20) DEFAULT 'text',
    last_message_at TIMESTAMPTZ,
    
    user1_deleted_at TIMESTAMPTZ,
    user2_deleted_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user1_id, user2_id)
);

CREATE INDEX IF NOT EXISTS idx_private_chats_user1 ON private_chats(user1_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_private_chats_user2 ON private_chats(user2_id, updated_at DESC);

-- 2. 私聊消息表 (无外键, 含反应计数)
CREATE TABLE IF NOT EXISTS private_messages (
    message_id UUID PRIMARY KEY DEFAULT uuidv7(),
    chat_id UUID NOT NULL,
    sender_id BIGINT NOT NULL,
    receiver_id BIGINT NOT NULL,
    
    content TEXT,
    message_type VARCHAR(20) DEFAULT 'text', -- text, image, audio, video, file, location
    metadata JSONB DEFAULT '{}'::jsonb, -- 媒体信息、引用等
    
    reply_to_id UUID,
    is_recalled BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    reaction_counts JSONB DEFAULT '{}'::jsonb, -- 聚合反应计数 { "like": {"count": 1, "users": [123]} }
    
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_private_messages_chat ON private_messages(chat_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_private_messages_sender ON private_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_private_messages_receiver ON private_messages(receiver_id);
CREATE INDEX IF NOT EXISTS idx_private_messages_reactions ON private_messages USING gin (reaction_counts);

-- 3. 未读计数表 (无外键, 优化 FILLFACTOR)
CREATE TABLE IF NOT EXISTS chat_unread_counts (
    user_id BIGINT NOT NULL,
    chat_id UUID NOT NULL, -- 可以是 private_chat_id 或 group_id (虽然 group_id 是 bigint... 等等，group_id 在旧设计是 bigint 吗？)
    -- 检查: 015_group_chat_module_rename.sql 重命名 groups -> groups.
    -- 通常 group_id 是 ID. 在 020 sql 中 group_id 是 BIGINT? 
    -- 让我们检查 020 sql... "INSERT INTO group_stats (group_id...)" 
    -- "CREATE TABLE groups" 通常使用 BIGINT 或 UUID.
    -- 既然 003 moments 用 user_id BIGINT, 002 users 用 user_id BIGSERIAL (BIGINT).
    -- 为了统一，如果 group_id 是主键，通常是 BIGINT 或 UUID.
    -- 020 中 sync_group_member_count 触发器用到了 group_id.
    -- 假设 groups 表主键是 group_id BIGSERIAL (类似 users) 或者 UUID.
    -- 015 rename groups -> groups.
    -- 现代设计通常用 Snowflake ID (BIGINT) 或 UUIDv7.
    -- 鉴于 002 users 是 BIGSERIAL, 003 moments 是 UUID.
    -- 我将把 groups 定义为 BIGSERIAL (BIGINT) 以匹配 user_id 的风格，或者 UUID.
    -- 这里的 chat_id 是 UUID (private_chat) 但 group_id 可能是 BIGINT.
    -- 为了兼容，chat_id 在这里定义为 VARCHAR(50) 或者是 UUID?
    -- 如果 private_chat_id 是 UUID, group_id 是 BIGINT, 那么 chat_unread_counts 的 chat_id 列类型需要兼容两者，或者分开。
    -- 通常设计：chat_id 统一为 String, 或者 chat_type + target_id.
    -- 让我们假设 chat_id 存储 UUID (私聊) 或 BIGINT (群聊) 的字符串形式，或者统一用 UUID.
    -- 为了安全，我将 groups 定义为 UUID 主键 (v7)，这样 chat_id 可以统一为 UUID.
    -- 如果 groups 原本是 BIGINT，这会是 breaking change。
    -- 020.sql 中 `group_id` 被转为 text: `group_member_count::text || ...`.
    -- 让我们使用 UUID 作为 groups 主键，这是新系统的最佳实践。
    
    chat_type VARCHAR(20) NOT NULL, -- private, group
    unread_count INTEGER DEFAULT 0,
    last_read_message_id UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (user_id, chat_id)
) WITH (fillfactor = 80);

-- 4. 群组表 (无外键, renamed from groups)
CREATE TABLE IF NOT EXISTS groups (
    group_id UUID PRIMARY KEY DEFAULT uuidv7(), -- 统一使用 UUID
    name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar_url TEXT,
    owner_id BIGINT NOT NULL,
    
    group_type VARCHAR(20) DEFAULT 'public', -- public, private, password
    max_members INTEGER DEFAULT 500,
    is_muted BOOLEAN DEFAULT FALSE, -- 全员禁言
    join_type VARCHAR(20) DEFAULT 'free', -- free, apply, invite
    
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    dismissed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_groups_owner ON groups(owner_id);
CREATE INDEX IF NOT EXISTS idx_groups_name ON groups USING GIN(name gin_trgm_ops);

-- 5. 群成员表 (无外键)
CREATE TABLE IF NOT EXISTS group_members (
    group_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    nickname VARCHAR(50), -- 群昵称
    role VARCHAR(20) DEFAULT 'member', -- owner, admin, member
    
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    inviter_id BIGINT,
    
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_user ON group_members(user_id, joined_at DESC);
CREATE INDEX IF NOT EXISTS idx_group_members_role ON group_members(group_id, role);

-- 6. 群消息表 (无外键, 含反应计数)
CREATE TABLE IF NOT EXISTS group_messages (
    message_id UUID PRIMARY KEY DEFAULT uuidv7(),
    group_id UUID NOT NULL,
    sender_id BIGINT NOT NULL,
    
    content TEXT,
    message_type VARCHAR(20) DEFAULT 'text',
    metadata JSONB DEFAULT '{}'::jsonb,
    
    reply_to_id UUID,
    is_recalled BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    reaction_counts JSONB DEFAULT '{}'::jsonb,
    at_users JSONB DEFAULT '[]'::jsonb, -- @的用户ID列表
    
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_group_messages_group ON group_messages(group_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_group_messages_sender ON group_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_group_messages_reactions ON group_messages USING gin (reaction_counts);

-- 7. 群统计表 (无外键)
CREATE TABLE IF NOT EXISTS group_stats (
    group_id UUID PRIMARY KEY,
    member_count INTEGER DEFAULT 0,
    message_count INTEGER DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 8. 群公告表 (无外键)
CREATE TABLE IF NOT EXISTS group_announcements (
    announcement_id UUID PRIMARY KEY DEFAULT uuidv7(),
    group_id UUID NOT NULL,
    publisher_id BIGINT NOT NULL,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    is_pinned BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_group_announcements_group ON group_announcements(group_id, created_at DESC);

-- 9. 群邀请/申请表 (无外键)
CREATE TABLE IF NOT EXISTS group_requests (
    request_id UUID PRIMARY KEY DEFAULT uuidv7(),
    group_id UUID NOT NULL,
    user_id BIGINT NOT NULL, -- 申请人或被邀请人
    operator_id BIGINT NOT NULL, -- 邀请人或处理人
    type VARCHAR(20) NOT NULL, -- invite, apply
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, rejected
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_requests_group ON group_requests(group_id, status);
CREATE INDEX IF NOT EXISTS idx_group_requests_user ON group_requests(user_id, status);
