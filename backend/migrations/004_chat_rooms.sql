-- 聊天室表（统一私聊和群聊）
CREATE TABLE chat_rooms (
    id BIGSERIAL PRIMARY KEY,
    room_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    room_type room_type NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT,
    avatar_url TEXT,
    owner_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    password_hash VARCHAR(255),
    max_members INTEGER NOT NULL DEFAULT 500,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chat_rooms_deleted_check CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL)
    )
);

-- 聊天室索引
CREATE INDEX idx_chat_rooms_id ON chat_rooms(id DESC);
CREATE UNIQUE INDEX idx_chat_rooms_room_id ON chat_rooms(room_id);
CREATE INDEX idx_chat_rooms_name ON chat_rooms(name) WHERE name IS NOT NULL;
CREATE INDEX idx_chat_rooms_owner ON chat_rooms(owner_id) WHERE owner_id IS NOT NULL;
CREATE INDEX idx_chat_rooms_type ON chat_rooms(room_type);
CREATE INDEX idx_chat_rooms_last_message ON chat_rooms(last_message_at DESC NULLS LAST);
CREATE INDEX idx_chat_rooms_not_deleted ON chat_rooms(is_deleted) WHERE is_deleted = FALSE;

-- 聊天室成员表（统一私聊和群聊成员）
CREATE TABLE room_members (
    id BIGSERIAL PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES chat_rooms(room_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role member_role NOT NULL DEFAULT 'member'::member_role,
    nickname VARCHAR(100),
    is_banned BOOLEAN NOT NULL DEFAULT FALSE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    muted_until TIMESTAMP WITH TIME ZONE,
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    unread_count INTEGER NOT NULL DEFAULT 0,
    last_read_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT room_members_deleted_check CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL)
    ),
    UNIQUE (room_id, user_id)
);

-- 聊天室成员索引
CREATE INDEX idx_room_members_id ON room_members(id DESC);
CREATE INDEX idx_room_members_room ON room_members(room_id, is_deleted) WHERE is_deleted = FALSE;
CREATE INDEX idx_room_members_user ON room_members(user_id, is_deleted) WHERE is_deleted = FALSE;
CREATE INDEX idx_room_members_role ON room_members(room_id, role) WHERE is_deleted = FALSE;
CREATE INDEX idx_room_members_unread ON room_members(user_id, unread_count) WHERE unread_count > 0 AND is_deleted = FALSE;
CREATE INDEX idx_room_members_muted ON room_members(user_id, is_muted) WHERE is_muted = TRUE AND is_deleted = FALSE;
CREATE INDEX idx_room_members_banned ON room_members(user_id, is_banned) WHERE is_banned = TRUE AND is_deleted = FALSE;

-- 聊天室邀请表（仅用于群聊邀请）
CREATE TABLE room_invites (
    id BIGSERIAL PRIMARY KEY,
    invite_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES chat_rooms(room_id) ON DELETE CASCADE,
    inviter_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    invitee_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    message TEXT,
    status invite_status NOT NULL DEFAULT 'pending'::invite_status CHECK (status != 'any'::invite_status),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    responded_at TIMESTAMP WITH TIME ZONE
);

-- 聊天室邀请索引
CREATE INDEX idx_room_invites_id ON room_invites(id DESC);
CREATE UNIQUE INDEX idx_room_invites_invite_id ON room_invites(invite_id);
CREATE INDEX idx_room_invites_invitee ON room_invites(invitee_id, status);
CREATE INDEX idx_room_invites_room ON room_invites(room_id);

-- 聊天室申请表（仅用于私密群申请）
CREATE TABLE room_applications (
    id BIGSERIAL PRIMARY KEY,
    application_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES chat_rooms(room_id) ON DELETE CASCADE,
    applicant_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    message TEXT,
    status invite_status NOT NULL DEFAULT 'pending'::invite_status CHECK (status != 'any'::invite_status),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewer_id BIGINT REFERENCES users(user_id),
    review_reason TEXT
);

-- 聊天室申请索引
CREATE INDEX idx_room_applications_id ON room_applications(id DESC);
CREATE UNIQUE INDEX idx_room_applications_application_id ON room_applications(application_id);
CREATE INDEX idx_room_applications_room ON room_applications(room_id, status);
CREATE INDEX idx_room_applications_applicant ON room_applications(applicant_id);

-- 聊天室设置表（统一私聊和群聊的个人设置）
CREATE TABLE room_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES chat_rooms(room_id) ON DELETE CASCADE,
    muted BOOLEAN NOT NULL DEFAULT FALSE,
    mention_only BOOLEAN NOT NULL DEFAULT FALSE,
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, room_id)
);

-- 聊天室设置索引
CREATE INDEX idx_room_settings_id ON room_settings(id DESC);
CREATE INDEX idx_room_settings_user ON room_settings(user_id);
CREATE INDEX idx_room_settings_pinned ON room_settings(user_id, pinned) WHERE pinned = TRUE;

