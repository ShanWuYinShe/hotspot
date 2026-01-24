-- 002_user_module.sql
-- 用户模块：用户表、设置、关系、位置历史、在线状态、会话、密码重置、验证码

-- ============================================================
-- 验证码类型枚举
-- ============================================================
CREATE TYPE verification_type AS ENUM (
    'register',
    'reset_password',
    'change_phone'
);

-- ============================================================
-- 用户核心表
-- ============================================================
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL,
    user_type user_type NOT NULL DEFAULT 'normal',
    gender gender,
    login_id VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255),
    nickname VARCHAR(100) NOT NULL,
    avatar_url TEXT,
    bio TEXT NOT NULL DEFAULT '', -- 优化：默认空字符串而非 NULL
    email VARCHAR(255),
    phone VARCHAR(20),
    current_latitude DOUBLE PRECISION,
    current_longitude DOUBLE PRECISION,
    current_location_name VARCHAR(255),
    location geography(POINT, 4326),
    location_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    is_banned BOOLEAN NOT NULL DEFAULT FALSE,
    ban_reason TEXT, -- 封禁原因
    banned_until TIMESTAMPTZ, -- 封禁截止时间
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_user_latitude_range CHECK (current_latitude IS NULL OR (current_latitude >= -90 AND current_latitude <= 90)),
    CONSTRAINT check_user_longitude_range CHECK (current_longitude IS NULL OR (current_longitude >= -180 AND current_longitude <= 180))
);

CREATE INDEX idx_users_user_id ON users (user_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_users_login_id ON users (login_id) WHERE login_id IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_users_nickname ON users USING gin (nickname gin_trgm_ops) WHERE is_deleted = FALSE;
CREATE INDEX idx_users_location ON users USING gist (location) WHERE location IS NOT NULL AND is_deleted = FALSE;

CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_users_location_sync
    BEFORE INSERT OR UPDATE OF current_latitude, current_longitude ON users
    FOR EACH ROW WHEN (NEW.current_latitude IS NOT NULL AND NEW.current_longitude IS NOT NULL)
    EXECUTE FUNCTION sync_user_location();

-- ============================================================
-- 用户设置（与 API 文档 UserSettingsData 对应）
-- ============================================================
CREATE TABLE user_settings (
    user_id BIGINT PRIMARY KEY,
    
    -- ============ 隐私设置 ============
    -- 基础隐私级别（影响大部分信息的默认可见性）
    privacy_level privacy_level NOT NULL DEFAULT 'public',
    -- 位置可见性
    location_visible BOOLEAN NOT NULL DEFAULT TRUE,
    -- 在线状态可见性
    online_visible BOOLEAN NOT NULL DEFAULT TRUE,
    last_online_visible BOOLEAN NOT NULL DEFAULT TRUE,
    -- 列表/统计可见性（细粒度控制）
    friends_visible privacy_level NOT NULL DEFAULT 'public',
    following_visible privacy_level NOT NULL DEFAULT 'public',
    followers_visible privacy_level NOT NULL DEFAULT 'public',
    moments_visible privacy_level NOT NULL DEFAULT 'public',
    liked_moments_visible privacy_level NOT NULL DEFAULT 'public',
    groups_visible privacy_level NOT NULL DEFAULT 'public',
    stats_visible privacy_level NOT NULL DEFAULT 'public',
    
    -- ============ 社交设置 ============
    allow_follow BOOLEAN NOT NULL DEFAULT TRUE,
    allow_private_chat chat_permission NOT NULL DEFAULT 'everyone',
    
    -- ============ 搜索设置 ============
    searchable BOOLEAN NOT NULL DEFAULT TRUE,
    phone_searchable BOOLEAN NOT NULL DEFAULT TRUE,
    nickname_searchable BOOLEAN NOT NULL DEFAULT TRUE,
    location_searchable BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- ============ 时间戳 ============
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

COMMENT ON TABLE user_settings IS '用户设置表，与 API 文档 UserSettingsData 完全对应';
COMMENT ON COLUMN user_settings.privacy_level IS '基础隐私级别：public-公开, friends-仅好友, private-仅自己';
COMMENT ON COLUMN user_settings.location_visible IS '是否显示位置信息';
COMMENT ON COLUMN user_settings.online_visible IS '是否显示在线状态';
COMMENT ON COLUMN user_settings.last_online_visible IS '是否显示最后上线时间';
COMMENT ON COLUMN user_settings.friends_visible IS '好友列表可见性';
COMMENT ON COLUMN user_settings.following_visible IS '关注列表可见性';
COMMENT ON COLUMN user_settings.followers_visible IS '粉丝列表可见性';
COMMENT ON COLUMN user_settings.moments_visible IS '瞬间列表可见性';
COMMENT ON COLUMN user_settings.liked_moments_visible IS '点赞瞬间列表可见性';
COMMENT ON COLUMN user_settings.groups_visible IS '群组列表可见性';
COMMENT ON COLUMN user_settings.stats_visible IS '统计数据可见性';
COMMENT ON COLUMN user_settings.allow_follow IS '是否允许被关注';
COMMENT ON COLUMN user_settings.allow_private_chat IS '私聊权限：everyone-所有人, friends-仅好友, nobody-禁止';
COMMENT ON COLUMN user_settings.searchable IS '总开关：是否可被搜索';
COMMENT ON COLUMN user_settings.phone_searchable IS '是否可通过手机号搜索';
COMMENT ON COLUMN user_settings.nickname_searchable IS '是否可通过昵称搜索';
COMMENT ON COLUMN user_settings.location_searchable IS '是否出现在附近的人';

CREATE TRIGGER trigger_user_settings_updated_at
    BEFORE UPDATE ON user_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 用户在线状态表（高并发优化：从users表分离）
-- ============================================================
CREATE TABLE user_online_status (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    is_online BOOLEAN NOT NULL DEFAULT FALSE,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ,
    connection_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_online_status_user ON user_online_status (user_id);
CREATE INDEX idx_user_online_status_online ON user_online_status (is_online) WHERE is_online = TRUE;

-- ============================================================
-- 用户关系：关注、拉黑、好友备注
-- 说明：好友 = 互相关注，无需独立的好友申请表
-- ============================================================
CREATE TABLE user_follows (
    id BIGSERIAL PRIMARY KEY,
    follow_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    follower_id BIGINT NOT NULL,
    following_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_follow_pair UNIQUE (follower_id, following_id),
    CONSTRAINT check_follow_not_self CHECK (follower_id != following_id)
);

CREATE INDEX idx_user_follows_follow_id ON user_follows (follow_id);
CREATE INDEX idx_user_follows_follower ON user_follows (follower_id, created_at DESC);
CREATE INDEX idx_user_follows_following ON user_follows (following_id, created_at DESC);
CREATE INDEX idx_user_follows_mutual ON user_follows (follower_id, following_id);

-- ============================================================
-- 关注请求（当 allow_follow = false 时）
-- ============================================================
CREATE TYPE follow_request_status AS ENUM (
    'pending',   -- 待处理
    'accepted',  -- 已接受
    'rejected'   -- 已拒绝
);

CREATE TABLE follow_requests (
    id BIGSERIAL PRIMARY KEY,
    request_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    requester_id BIGINT NOT NULL,      -- 请求关注的用户
    target_id BIGINT NOT NULL,         -- 被请求关注的用户
    message TEXT,                      -- 附带消息（可选）
    status follow_request_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    handled_at TIMESTAMPTZ,            -- 处理时间
    CONSTRAINT unique_follow_request UNIQUE (requester_id, target_id),
    CONSTRAINT check_follow_request_not_self CHECK (requester_id != target_id)
);

CREATE INDEX idx_follow_requests_request_id ON follow_requests (request_id);
CREATE INDEX idx_follow_requests_requester ON follow_requests (requester_id, status, created_at DESC);
CREATE INDEX idx_follow_requests_target ON follow_requests (target_id, status, created_at DESC);
CREATE INDEX idx_follow_requests_pending ON follow_requests (target_id, created_at DESC) WHERE status = 'pending';

COMMENT ON TABLE follow_requests IS '关注请求表：当用户设置 allow_follow = false 时，关注需要审批';
COMMENT ON COLUMN follow_requests.request_id IS '关注请求ID，对外暴露的业务标识';
COMMENT ON COLUMN follow_requests.requester_id IS '请求关注的用户ID';
COMMENT ON COLUMN follow_requests.target_id IS '被请求关注的用户ID';
COMMENT ON COLUMN follow_requests.message IS '附带消息';
COMMENT ON COLUMN follow_requests.status IS '请求状态：pending-待处理, accepted-已接受, rejected-已拒绝';
COMMENT ON COLUMN follow_requests.handled_at IS '处理时间';

CREATE TABLE user_blocks (
    id BIGSERIAL PRIMARY KEY,
    block_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    blocker_id BIGINT NOT NULL,
    blocked_id BIGINT NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_block_pair UNIQUE (blocker_id, blocked_id),
    CONSTRAINT check_block_not_self CHECK (blocker_id != blocked_id)
);

CREATE INDEX idx_user_blocks_block_id ON user_blocks (block_id);
CREATE INDEX idx_user_blocks_blocker ON user_blocks (blocker_id, created_at DESC);
CREATE INDEX idx_user_blocks_blocked ON user_blocks (blocked_id, created_at DESC);

CREATE TABLE user_friend_remarks (
    id BIGSERIAL PRIMARY KEY,
    remark_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    friend_id BIGINT NOT NULL,
    remark VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CONSTRAINT unique_friend_remark_pair UNIQUE (user_id, friend_id)
);

CREATE INDEX idx_friend_remarks_remark_id ON user_friend_remarks (remark_id);
CREATE INDEX idx_friend_remarks_user ON user_friend_remarks (user_id, created_at DESC);

CREATE TRIGGER trigger_friend_remarks_updated_at
    BEFORE UPDATE ON user_friend_remarks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 用户位置历史
-- ============================================================
CREATE TABLE user_location_history (
    id BIGSERIAL PRIMARY KEY,
    history_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_name VARCHAR(255),
    location geography(POINT, 4326), -- 由 trigger 自动填充
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_location_latitude_range CHECK (latitude >= -90 AND latitude <= 90),
    CONSTRAINT check_location_longitude_range CHECK (longitude >= -180 AND longitude <= 180)
);

CREATE INDEX idx_location_history_history_id ON user_location_history (history_id);
CREATE INDEX idx_location_history_user_id ON user_location_history (user_id, recorded_at DESC);
CREATE INDEX idx_location_history_location ON user_location_history USING gist (location);

CREATE TRIGGER trigger_location_history_sync
    BEFORE INSERT ON user_location_history
    FOR EACH ROW
    EXECUTE FUNCTION sync_location_history();

-- ============================================================
-- 会话管理
-- ============================================================
CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    device_type VARCHAR(50),
    device_name VARCHAR(100),
    device_fingerprint VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    access_token_hash VARCHAR(255) NOT NULL,
    refresh_token_hash VARCHAR(255) NOT NULL,
    previous_refresh_token_hash VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_sessions_session_id ON sessions (session_id) WHERE is_revoked = FALSE;
CREATE INDEX idx_sessions_user_id ON sessions (user_id, last_active_at DESC) WHERE is_revoked = FALSE;

CREATE TABLE password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    reset_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    code VARCHAR(6) NOT NULL,
    code_hash VARCHAR(255) NOT NULL,
    reset_token_hash VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    used_at TIMESTAMPTZ,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    is_expired BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_password_reset_reset_id ON password_reset_tokens (reset_id) WHERE is_used = FALSE AND is_expired = FALSE;
CREATE INDEX idx_password_reset_user_id ON password_reset_tokens (user_id, created_at DESC);

-- ============================================================
-- 验证码表
-- ============================================================
CREATE TABLE verification_codes (
    id BIGSERIAL PRIMARY KEY,
    code_id VARCHAR(36) NOT NULL UNIQUE DEFAULT uuid_generate_v4()::TEXT,
    phone VARCHAR(20) NOT NULL,
    code VARCHAR(10) NOT NULL,
    type verification_type NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 约束
    CONSTRAINT check_phone_format CHECK (phone ~ '^\+?[1-9]\d{1,14}$'),
    CONSTRAINT check_code_length CHECK (LENGTH(code) >= 4 AND LENGTH(code) <= 10),
    CONSTRAINT check_max_attempts CHECK (max_attempts > 0 AND max_attempts <= 10)
);

CREATE INDEX idx_verification_codes_phone_type ON verification_codes(phone, type, created_at DESC);
CREATE INDEX idx_verification_codes_expires_at ON verification_codes(expires_at) WHERE verified_at IS NULL;
CREATE INDEX idx_verification_codes_code_id ON verification_codes(code_id) WHERE verified_at IS NULL;

COMMENT ON TABLE verification_codes IS '手机验证码表：存储短信验证码，支持注册、重置密码、更换手机号等场景';
COMMENT ON COLUMN verification_codes.code_id IS '验证码会话ID，对外暴露的业务标识';
COMMENT ON COLUMN verification_codes.phone IS '手机号（国际格式，如 +8613812345678）';
COMMENT ON COLUMN verification_codes.code IS '验证码（通常为6位数字）';
COMMENT ON COLUMN verification_codes.type IS '验证码用途类型';
COMMENT ON COLUMN verification_codes.expires_at IS '过期时间（通常为5分钟后）';
COMMENT ON COLUMN verification_codes.attempts IS '已尝试验证次数';
COMMENT ON COLUMN verification_codes.max_attempts IS '最大允许尝试次数';
COMMENT ON COLUMN verification_codes.verified_at IS '验证通过时间（NULL表示未验证）';
