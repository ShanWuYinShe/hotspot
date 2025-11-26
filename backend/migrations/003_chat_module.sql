-- 003_chat_module.sql
-- 聊天模块：私聊、群聊、成员管理、未读计数

-- ============================================================
-- 私聊
-- ============================================================
CREATE TABLE private_chats (
    id BIGSERIAL PRIMARY KEY,
    private_chat_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    user_a_id BIGINT NOT NULL,
    user_b_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_private_chat_user_order CHECK (user_a_id < user_b_id),
    CONSTRAINT unique_private_chat_pair UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_private_chats_cursor_id ON private_chats (cursor_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_chats_user_a ON private_chats (user_a_id, updated_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_private_chats_user_b ON private_chats (user_b_id, updated_at DESC) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_private_chats_updated_at
    BEFORE UPDATE ON private_chats FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE private_chat_settings (
    id BIGSERIAL PRIMARY KEY,
    private_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT unique_private_chat_user_settings UNIQUE (private_chat_id, user_id)
);

CREATE INDEX idx_private_chat_settings_user ON private_chat_settings (user_id);
CREATE INDEX idx_private_chat_settings_chat ON private_chat_settings (private_chat_id);

CREATE TRIGGER trigger_private_chat_settings_updated_at
    BEFORE UPDATE ON private_chat_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 群聊
-- ============================================================
CREATE TABLE group_chats (
    id BIGSERIAL PRIMARY KEY,
    group_chat_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
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
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_group_latitude_range CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT check_group_longitude_range CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE INDEX idx_group_chats_id ON group_chats (group_chat_id) WHERE is_deleted = FALSE AND is_dismissed = FALSE;
CREATE INDEX idx_group_chats_cursor_id ON group_chats (cursor_id) WHERE is_deleted = FALSE AND is_dismissed = FALSE;
CREATE INDEX idx_group_chats_owner ON group_chats (owner_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_group_chats_location ON group_chats USING gist (location) WHERE location IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_group_chats_name ON group_chats USING gin (name gin_trgm_ops) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_group_chats_updated_at
    BEFORE UPDATE ON group_chats FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_group_chats_location_sync
    BEFORE INSERT OR UPDATE OF latitude, longitude ON group_chats
    FOR EACH ROW WHEN (NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL)
    EXECUTE FUNCTION sync_group_location();

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

CREATE TABLE group_members (
    id BIGSERIAL PRIMARY KEY,
    member_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    role member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    muted_until TIMESTAMPTZ,
    is_left BOOLEAN NOT NULL DEFAULT FALSE,
    left_at TIMESTAMPTZ,
    CONSTRAINT unique_group_member UNIQUE (group_chat_id, user_id)
);

CREATE INDEX idx_group_members_member_id ON group_members (member_id);
CREATE INDEX idx_group_members_cursor_id ON group_members (cursor_id) WHERE is_left = FALSE;
CREATE INDEX idx_group_members_group_chat_id ON group_members (group_chat_id, joined_at DESC) WHERE is_left = FALSE;
CREATE INDEX idx_group_members_user_id ON group_members (user_id, joined_at DESC) WHERE is_left = FALSE;
CREATE INDEX idx_group_members_count ON group_members (group_chat_id) WHERE is_left = FALSE;
CREATE INDEX idx_group_members_role ON group_members (group_chat_id, role) WHERE is_left = FALSE;

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

CREATE TABLE group_invites (
    id BIGSERIAL PRIMARY KEY,
    invite_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
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
CREATE INDEX idx_group_invites_cursor_id ON group_invites (cursor_id);
CREATE INDEX idx_group_invites_invitee ON group_invites (invitee_id, status, created_at DESC);
CREATE INDEX idx_group_invites_group ON group_invites (group_chat_id, status, created_at DESC);

CREATE TABLE group_applications (
    id BIGSERIAL PRIMARY KEY,
    application_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    group_chat_id UUID NOT NULL,
    applicant_id BIGINT NOT NULL,
    message TEXT,
    status invitation_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    handled_at TIMESTAMPTZ,
    handled_by BIGINT
);

CREATE INDEX idx_group_applications_application_id ON group_applications (application_id);
CREATE INDEX idx_group_applications_cursor_id ON group_applications (cursor_id);
CREATE INDEX idx_group_applications_group_chat_id ON group_applications (group_chat_id, status, created_at DESC);
CREATE INDEX idx_group_applications_applicant ON group_applications (applicant_id, created_at DESC);

-- ============================================================
-- 聊天未读计数表（高并发优化：独立管理未读计数）
-- ============================================================
CREATE TABLE chat_unread_counts (
    id BIGSERIAL PRIMARY KEY,
    chat_type VARCHAR(20) NOT NULL,
    chat_id UUID NOT NULL,
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
