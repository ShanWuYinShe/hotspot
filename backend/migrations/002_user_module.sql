-- ============================================================
-- 002_user_module.sql
-- 用户模块：用户、统计、设置、关系
-- ============================================================

-- 1. 用户表
CREATE TABLE IF NOT EXISTS users (
    user_id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(20) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    
    nickname VARCHAR(50),
    avatar_url TEXT,
    bio TEXT,
    gender user_gender DEFAULT 'unknown',
    birthday DATE,
    
    user_type VARCHAR(20) DEFAULT 'normal', -- normal, official, admin
    status user_status DEFAULT 'active',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- 2. 用户统计表 (无外键)
CREATE TABLE IF NOT EXISTS user_stats (
    user_id BIGINT PRIMARY KEY, -- 对应 users.user_id
    moments_count INTEGER DEFAULT 0,
    following_count INTEGER DEFAULT 0,
    followers_count INTEGER DEFAULT 0,
    likes_received INTEGER DEFAULT 0,
    groups_count INTEGER DEFAULT 0,
    reactions_given INTEGER DEFAULT 0,
    
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. 用户设置表 (无外键)
CREATE TABLE IF NOT EXISTS user_settings (
    user_id BIGINT PRIMARY KEY, -- 对应 users.user_id
    language VARCHAR(10) DEFAULT 'zh-CN',
    theme VARCHAR(10) DEFAULT 'system',
    notifications_enabled BOOLEAN DEFAULT TRUE,
    privacy_settings JSONB DEFAULT '{}'::jsonb,
    preferences JSONB DEFAULT '{}'::jsonb,
    
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT check_preferences_is_object CHECK (jsonb_typeof(preferences) = 'object'),
    CONSTRAINT check_privacy_is_object CHECK (jsonb_typeof(privacy_settings) = 'object')
);

-- 4. 关注关系表 (无外键)
CREATE TABLE IF NOT EXISTS follows (
    follower_id BIGINT NOT NULL, -- 关注者
    following_id BIGINT NOT NULL, -- 被关注者
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (follower_id, following_id)
);

CREATE INDEX IF NOT EXISTS idx_follows_follower ON follows(follower_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_follows_following ON follows(following_id, created_at DESC);

-- 5. 关注请求表 (无外键)
CREATE TABLE IF NOT EXISTS follow_requests (
    request_id UUID PRIMARY KEY DEFAULT uuidv7(),
    requester_id BIGINT NOT NULL,
    target_id BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, rejected
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_follow_requests_target ON follow_requests(target_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_follow_requests_requester ON follow_requests(requester_id, status, created_at DESC);

-- 6. 黑名单表 (无外键)
CREATE TABLE IF NOT EXISTS blocks (
    blocker_id BIGINT NOT NULL,
    blocked_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (blocker_id, blocked_id)
);

-- 7. 密码重置记录 (无外键)
CREATE TABLE IF NOT EXISTS password_resets (
    reset_id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id BIGINT NOT NULL,
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_resets_token ON password_resets(token);
CREATE INDEX IF NOT EXISTS idx_password_resets_user ON password_resets(user_id);

-- 8. 验证码记录 (无外键)
CREATE TABLE IF NOT EXISTS verification_codes (
    code_id UUID PRIMARY KEY DEFAULT uuidv7(),
    target VARCHAR(255) NOT NULL, -- email or phone
    code VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL, -- register, login, reset_password
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_codes_target ON verification_codes(target, type, used);

-- 9. 设备Token表 (无外键)
CREATE TABLE IF NOT EXISTS device_tokens (
    token_id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id BIGINT NOT NULL,
    device_token VARCHAR(255) NOT NULL,
    platform VARCHAR(20) NOT NULL, -- ios, android, harmony
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, device_token)
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user ON device_tokens(user_id);

-- 10. 用户位置历史 (无外键)
CREATE TABLE IF NOT EXISTS user_location_history (
    history_id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id BIGINT NOT NULL,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_location_history_user ON user_location_history(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_location_history_geo ON user_location_history USING GIST(location);
