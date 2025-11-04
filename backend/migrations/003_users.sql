-- 用户表
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    login_id VARCHAR(255) UNIQUE,
    nickname VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255),
    avatar_url TEXT,
    bio TEXT,
    privacy_level VARCHAR(20) DEFAULT 'public' NOT NULL CHECK (privacy_level IN ('public', 'friends', 'private')),
    location_sharing BOOLEAN DEFAULT TRUE NOT NULL,
    allow_follow BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_active_at TIMESTAMP WITH TIME ZONE,
    last_seen_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT users_deleted_check CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL)
    )
);

-- 用户索引
CREATE INDEX idx_users_id ON users(id DESC);
CREATE UNIQUE INDEX idx_users_user_id ON users(user_id);
CREATE INDEX idx_users_login_id ON users(login_id);
CREATE INDEX idx_users_nickname ON users(nickname);
CREATE INDEX idx_users_last_seen ON users(last_seen_at DESC NULLS LAST);
CREATE INDEX idx_users_not_deleted ON users(is_deleted) WHERE is_deleted = FALSE;

-- 用户关系表
CREATE TABLE user_relationships (
    id BIGSERIAL PRIMARY KEY,
    relationship_id BIGINT NOT NULL UNIQUE,
    from_user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    to_user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    relationship_type relationship_type NOT NULL CHECK (relationship_type != 'any'::relationship_type),
    status relationship_status NOT NULL DEFAULT 'pending'::relationship_status CHECK (status != 'any'::relationship_status),
    remark VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (from_user_id, to_user_id, relationship_type)
);

-- 用户关系索引
CREATE INDEX idx_user_relationships_id ON user_relationships(id DESC);
CREATE UNIQUE INDEX idx_user_relationships_relationship_id ON user_relationships(relationship_id);
CREATE INDEX idx_user_relationships_from ON user_relationships(from_user_id, relationship_type, status);
CREATE INDEX idx_user_relationships_to ON user_relationships(to_user_id, relationship_type, status);

-- 会话表
CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    device_type VARCHAR(50),
    device_name VARCHAR(100),
    ip_address INET,
    location VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- 会话索引
CREATE INDEX idx_sessions_id ON sessions(id DESC);
CREATE UNIQUE INDEX idx_sessions_session_id ON sessions(session_id);
CREATE INDEX idx_sessions_user ON sessions(user_id, last_active_at DESC);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

