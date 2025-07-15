-- 用户系统相关表
-- 包含: users, user_locations, user_settings

-- 用户主表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    login_id VARCHAR(100) NOT NULL UNIQUE,
    nickname VARCHAR(100) NOT NULL DEFAULT '牛头马面',
    password_hash TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    email VARCHAR(255),
    avatar_url TEXT
);

CREATE INDEX idx_users_last_active_at ON users(last_active_at);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_updated_at ON users(updated_at);
CREATE INDEX idx_users_expires_at ON users(expires_at);
CREATE INDEX idx_users_email ON users(email);

-- 用户位置表
CREATE TABLE IF NOT EXISTS user_locations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.16,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.68,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    location_name TEXT NOT NULL DEFAULT '南天门',
    geom GEOGRAPHY(POINT, 4326),
    CONSTRAINT fk_user_locations_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE TRIGGER update_user_locations_geom
BEFORE INSERT OR UPDATE OF latitude, longitude ON user_locations
FOR EACH ROW EXECUTE FUNCTION update_geom_column();

CREATE INDEX idx_user_locations_geom ON user_locations USING gist (geom);
CREATE INDEX idx_user_locations_updated_at ON user_locations(updated_at);

-- 用户设置表
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_notification_level') THEN
        CREATE TYPE user_notification_level AS ENUM ('all', 'mention', 'important', 'muted');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS user_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    notification_level user_notification_level NOT NULL DEFAULT 'all',
    -- 扩展的通知设置字段
    message_notifications BOOLEAN NOT NULL DEFAULT true,
    group_notifications BOOLEAN NOT NULL DEFAULT true,
    system_notifications BOOLEAN NOT NULL DEFAULT true,
    location_notifications BOOLEAN NOT NULL DEFAULT true,
    quiet_hours_enabled BOOLEAN NOT NULL DEFAULT false,
    quiet_hours_start TIME DEFAULT '22:00:00',
    quiet_hours_end TIME DEFAULT '08:00:00',
    notification_radius_km DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    sound_enabled BOOLEAN NOT NULL DEFAULT true,
    vibration_enabled BOOLEAN NOT NULL DEFAULT true,
    
    CONSTRAINT fk_user_settings_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);

-- 用户好友关系表
CREATE TABLE IF NOT EXISTS friendships (
    id BIGSERIAL PRIMARY KEY,
    friendship_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    requester_id BIGINT NOT NULL,
    addressee_id BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT friendships_different_users CHECK (requester_id != addressee_id),
    CONSTRAINT fk_friendships_requester_id 
        FOREIGN KEY (requester_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_friendships_addressee_id 
        FOREIGN KEY (addressee_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_friendships_unique_pair ON friendships(
    GREATEST(requester_id, addressee_id), 
    LEAST(requester_id, addressee_id)
);
CREATE INDEX idx_friendships_requester_id ON friendships(requester_id);
CREATE INDEX idx_friendships_addressee_id ON friendships(addressee_id);
CREATE INDEX idx_friendships_status ON friendships(status);
