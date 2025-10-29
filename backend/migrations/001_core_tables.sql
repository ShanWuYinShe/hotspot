-- 启用 PostGIS 扩展
CREATE EXTENSION IF NOT EXISTS postgis;

-- 用户表
CREATE TABLE users (
    user_id BIGINT PRIMARY KEY,
    login_id VARCHAR(255) UNIQUE,
    nickname VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255),
    avatar_url TEXT,
    bio TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    location_name VARCHAR(255),
    location GEOGRAPHY(POINT, 4326),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active_at TIMESTAMP WITH TIME ZONE
);

-- 用户索引
CREATE INDEX idx_users_login_id ON users(login_id);
CREATE INDEX idx_users_nickname ON users(nickname);
CREATE INDEX idx_users_location ON users USING GIST(location);

-- 自动更新 location 字段的触发器
CREATE OR REPLACE FUNCTION update_user_location()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.longitude IS NOT NULL AND NEW.latitude IS NOT NULL THEN
        NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_location
BEFORE INSERT OR UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_user_location();

-- 用户关系表
CREATE TABLE user_relationships (
    relationship_id BIGSERIAL PRIMARY KEY,
    from_user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    to_user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    relationship_type VARCHAR(20) NOT NULL CHECK (relationship_type IN ('friend', 'following', 'blocked')),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    remark VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (from_user_id, to_user_id, relationship_type)
);

-- 用户关系索引
CREATE INDEX idx_user_relationships_from ON user_relationships(from_user_id, relationship_type, status);
CREATE INDEX idx_user_relationships_to ON user_relationships(to_user_id, relationship_type, status);

-- 会话表
CREATE TABLE sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    device_type VARCHAR(50),
    device_name VARCHAR(100),
    ip_address INET,
    location VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- 会话索引
CREATE INDEX idx_sessions_user ON sessions(user_id, last_active_at DESC);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

