-- 004_group_chat_module.sql
-- 群聊模块：群聊会话、群设置、成员管理、邀请/申请、封禁、群消息、已读状态、消息反应、群公告

-- ============================================================
-- 群聊会话
-- ============================================================
CREATE TABLE group_chats (
    id BIGSERIAL PRIMARY KEY,
    group_chat_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '', -- 优化：默认空字符串
    avatar_url TEXT,
    owner_id BIGINT NOT NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    location_name VARCHAR(255),
    location geography(POINT, 4326),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    is_dismissed BOOLEAN NOT NULL DEFAULT FALSE,
    dismiss_reason TEXT, -- 解散原因
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_group_latitude_range CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT check_group_longitude_range CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE INDEX idx_group_chats_id ON group_chats (group_chat_id) WHERE is_deleted = FALSE AND is_dismissed = FALSE;
CREATE INDEX idx_group_chats_owner ON group_chats (owner_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_chats_location ON group_chats USING gist (location) WHERE location IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_group_chats_name ON group_chats USING gin (name gin_trgm_ops) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_group_chats_updated_at
    BEFORE UPDATE ON group_chats FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_group_chats_location_sync
    BEFORE INSERT OR UPDATE OF latitude, longitude ON group_chats
    FOR EACH ROW WHEN (NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL)
    EXECUTE FUNCTION sync_group_location();

-- ============================================================
-- 群组设置
-- ============================================================
CREATE TABLE group_settings (
    id BIGSERIAL PRIMARY KEY,
    group_chat_id UUID UNIQUE NOT NULL,
    join_method join_method NOT NULL DEFAULT 'open',
    is_searchable BOOLEAN NOT NULL DEFAULT TRUE,
    max_members INTEGER,
    password_hash VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_group_settings_chat ON group_settings (group_chat_id);

CREATE TRIGGER trigger_group_settings_updated_at
    BEFORE UPDATE ON group_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 群组成员
-- ============================================================
CREATE TABLE group_members (
    id BIGSERIAL PRIMARY KEY,
    member_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    role member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    muted_until TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT unique_group_member UNIQUE (group_chat_id, user_id)
);

CREATE INDEX idx_group_members_member_id ON group_members (member_id);
CREATE INDEX idx_group_members_group_chat_id ON group_members (group_chat_id, joined_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_members_user_id ON group_members (user_id, joined_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_members_count ON group_members (group_chat_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_members_role ON group_members (group_chat_id, role) WHERE is_deleted = FALSE;

-- ============================================================
-- 群组成员操作历史
-- ============================================================
CREATE TABLE group_member_history (
    id BIGSERIAL PRIMARY KEY,
    history_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    action member_action NOT NULL,
    operator_id BIGINT,
    old_role member_role,
    new_role member_role,
    reason TEXT,
    muted_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_group_member_history_group ON group_member_history (group_chat_id, created_at DESC);
CREATE INDEX idx_group_member_history_user ON group_member_history (user_id, created_at DESC);
CREATE INDEX idx_group_member_history_action ON group_member_history (action, created_at DESC);

-- ============================================================
-- 群组成员个人设置
-- ============================================================
CREATE TABLE group_member_settings (
    id BIGSERIAL PRIMARY KEY,
    group_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT unique_group_member_settings UNIQUE (group_chat_id, user_id)
);

CREATE INDEX idx_group_member_settings_user ON group_member_settings (user_id);
CREATE INDEX idx_group_member_settings_chat ON group_member_settings (group_chat_id);

CREATE TRIGGER trigger_group_member_settings_updated_at
    BEFORE UPDATE ON group_member_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 群组封禁记录
-- ============================================================
CREATE TABLE group_bans (
    id BIGSERIAL PRIMARY KEY,
    ban_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    banned_by BIGINT NOT NULL,
    reason TEXT,
    banned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unbanned_at TIMESTAMPTZ,
    unbanned_by BIGINT,
    CONSTRAINT unique_group_ban UNIQUE (group_chat_id, user_id)
);

CREATE INDEX idx_group_bans_group ON group_bans (group_chat_id, banned_at DESC) WHERE unbanned_at IS NULL;
CREATE INDEX idx_group_bans_user ON group_bans (user_id, banned_at DESC) WHERE unbanned_at IS NULL;

-- ============================================================
-- 群组邀请
-- ============================================================
CREATE TABLE group_invites (
    id BIGSERIAL PRIMARY KEY,
    invite_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    inviter_id BIGINT NOT NULL,
    invitee_id BIGINT NOT NULL,
    message TEXT,
    status invitation_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ
);

CREATE INDEX idx_group_invites_invite_id ON group_invites (invite_id);
CREATE INDEX idx_group_invites_invitee ON group_invites (invitee_id, status, created_at DESC);
CREATE INDEX idx_group_invites_group ON group_invites (group_chat_id, status, created_at DESC);

-- ============================================================
-- 群组申请
-- ============================================================
CREATE TABLE group_applications (
    id BIGSERIAL PRIMARY KEY,
    application_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    applicant_id BIGINT NOT NULL,
    message TEXT,
    status invitation_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    handled_at TIMESTAMPTZ,
    handled_by BIGINT
);

CREATE INDEX idx_group_applications_application_id ON group_applications (application_id);
CREATE INDEX idx_group_applications_group_chat_id ON group_applications (group_chat_id, status, created_at DESC);
CREATE INDEX idx_group_applications_applicant ON group_applications (applicant_id, created_at DESC);

-- ============================================================
-- 群公告表
-- ============================================================
CREATE TABLE group_announcements (
    id BIGSERIAL PRIMARY KEY,
    announcement_id VARCHAR(36) NOT NULL UNIQUE DEFAULT uuid_generate_v4()::TEXT,
    group_chat_id VARCHAR(36) NOT NULL,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    created_by BIGINT NOT NULL,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 约束
    CONSTRAINT check_title_length CHECK (LENGTH(title) >= 1 AND LENGTH(title) <= 100),
    CONSTRAINT check_content_length CHECK (LENGTH(content) >= 1 AND LENGTH(content) <= 2000)
);

CREATE INDEX idx_group_announcements_group_id ON group_announcements(group_chat_id, is_pinned DESC, created_at DESC);
CREATE INDEX idx_group_announcements_created_by ON group_announcements(created_by);

COMMENT ON TABLE group_announcements IS '群公告表';
COMMENT ON COLUMN group_announcements.announcement_id IS '公告ID（业务标识）';
COMMENT ON COLUMN group_announcements.group_chat_id IS '所属群组ID';
COMMENT ON COLUMN group_announcements.title IS '公告标题';
COMMENT ON COLUMN group_announcements.content IS '公告内容';
COMMENT ON COLUMN group_announcements.created_by IS '发布者用户ID';
COMMENT ON COLUMN group_announcements.is_pinned IS '是否置顶';

-- ============================================================
-- 群聊消息
-- ============================================================
CREATE TABLE group_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
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
    -- 位置消息字段
    location_latitude DOUBLE PRECISION,
    location_longitude DOUBLE PRECISION,
    location_name VARCHAR(200),
    location_address TEXT,
    
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ,
    is_recalled BOOLEAN NOT NULL DEFAULT FALSE,
    recalled_by BIGINT,
    recalled_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    
    -- 约束：当 type='location' 时，位置字段必须存在
    CONSTRAINT check_location_fields_group CHECK (
        (type != 'location') OR 
        (location_latitude IS NOT NULL AND location_longitude IS NOT NULL AND location_name IS NOT NULL)
    )
);

CREATE INDEX idx_group_messages_chat_id ON group_messages (group_chat_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_messages_sender ON group_messages (sender_id, sent_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_messages_mentions ON group_messages USING gin (mentions) WHERE mentions IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_group_messages_content_search ON group_messages USING gin (content gin_trgm_ops) WHERE type = 'text' AND is_deleted = FALSE;

COMMENT ON COLUMN group_messages.location_latitude IS '位置消息的纬度';
COMMENT ON COLUMN group_messages.location_longitude IS '位置消息的经度';
COMMENT ON COLUMN group_messages.location_name IS '位置消息的地点名';
COMMENT ON COLUMN group_messages.location_address IS '位置消息的详细地址（可选）';

-- ============================================================
-- 群聊消息已读状态
-- ============================================================
CREATE TABLE group_message_read_status (
    id BIGSERIAL PRIMARY KEY,
    read_status_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL,
    group_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_message_read_status UNIQUE (message_id, user_id)
);

CREATE INDEX idx_message_read_status_id ON group_message_read_status (read_status_id);
CREATE INDEX idx_message_read_status_message_id ON group_message_read_status (message_id, read_at DESC);
CREATE INDEX idx_message_read_status_user ON group_message_read_status (user_id, read_at DESC);
CREATE INDEX idx_message_read_status_group ON group_message_read_status (group_chat_id, user_id);

-- ============================================================
-- 群聊消息反应
-- ============================================================
CREATE TABLE group_message_reactions (
    id BIGSERIAL PRIMARY KEY,
    reaction_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    type reaction_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_group_message_reaction UNIQUE (message_id, user_id, type)
);

CREATE INDEX idx_group_message_reactions_id ON group_message_reactions (reaction_id);
CREATE INDEX idx_group_message_reactions_message_id ON group_message_reactions (message_id, created_at DESC);
CREATE INDEX idx_group_message_reactions_user ON group_message_reactions (user_id, created_at DESC);

-- ============================================================
-- 聊天未读计数表（统一管理私聊和群聊，高并发优化）
-- ============================================================
CREATE TABLE chat_unread_counts (
    id BIGSERIAL PRIMARY KEY,
    chat_type VARCHAR(20) NOT NULL,
    chat_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    unread_count INTEGER NOT NULL DEFAULT 0,
    last_read_at TIMESTAMPTZ,
    last_read_message_id UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_chat_unread_count UNIQUE (chat_type, chat_id, user_id),
    CONSTRAINT check_chat_type CHECK (chat_type IN ('private_chat', 'group_chat'))
);

CREATE INDEX idx_chat_unread_chat ON chat_unread_counts (chat_type, chat_id);
CREATE INDEX idx_chat_unread_user ON chat_unread_counts (user_id);
CREATE INDEX idx_chat_unread_user_count ON chat_unread_counts (user_id, unread_count) WHERE unread_count > 0;
