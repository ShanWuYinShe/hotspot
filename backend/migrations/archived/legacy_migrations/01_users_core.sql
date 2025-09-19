-- ============================================================================
-- HotSpot 用户核心模块迁移 v2.0
-- ============================================================================
-- 
-- 功能职责：
-- - 创建用户核心表结构
-- - 用户认证和授权
-- - 用户基础信息管理
-- - 用户位置信息
--
-- 依赖关系：
-- - 依赖 00_foundation.sql 的基础设施
--
-- ============================================================================

-- 用户主表
CREATE TABLE users (
    -- 主键和基础信息
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    
    -- 认证信息
    username VARCHAR(50) UNIQUE,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(20) UNIQUE,
    password_hash VARCHAR(255),
    
    -- 用户类型和状态
    user_type user_type_enum NOT NULL DEFAULT 'regular',
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    
    -- 个人信息
    nickname VARCHAR(100),
    avatar_url TEXT,
    bio TEXT,
    birth_date DATE,
    gender VARCHAR(10),
    
    -- 隐私设置
    privacy_settings JSONB DEFAULT '{}',
    notification_settings JSONB DEFAULT '{}',
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE, -- 临时用户过期时间
    
    -- 约束
    CONSTRAINT users_username_length CHECK (char_length(username) >= 3),
    CONSTRAINT users_nickname_length CHECK (char_length(nickname) >= 1),
    CONSTRAINT users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_valid_user_type CHECK (user_type IN ('regular', 'temporary', 'admin'))
);

-- 用户位置表
CREATE TABLE user_locations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 地理位置信息
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    altitude DOUBLE PRECISION,
    accuracy DOUBLE PRECISION,
    
    -- 地址信息
    address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    postal_code VARCHAR(20),
    
    -- PostGIS地理点
    location GEOGRAPHY(POINT, 4326) GENERATED ALWAYS AS (ST_Point(longitude, latitude)) STORED,
    
    -- 位置元数据
    location_source VARCHAR(50) DEFAULT 'gps', -- gps, network, manual
    is_current BOOLEAN DEFAULT true,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT user_locations_valid_coordinates CHECK (validate_coordinates(latitude, longitude)),
    CONSTRAINT user_locations_unique_current UNIQUE (user_id, is_current) DEFERRABLE INITIALLY DEFERRED
);

-- 用户会话表
CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 会话信息
    session_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    device_id TEXT,
    device_info JSONB DEFAULT '{}',
    
    -- 网络信息
    ip_address INET,
    user_agent TEXT,
    
    -- 会话状态
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 用户偏好设置表
CREATE TABLE user_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 语言和地区设置
    language VARCHAR(10) DEFAULT 'zh-CN',
    timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
    
    -- 隐私设置
    profile_visibility VARCHAR(20) DEFAULT 'public', -- public, friends, private
    location_sharing BOOLEAN DEFAULT true,
    online_status_visible BOOLEAN DEFAULT true,
    
    -- 通知设置
    push_notifications BOOLEAN DEFAULT true,
    email_notifications BOOLEAN DEFAULT true,
    sms_notifications BOOLEAN DEFAULT false,
    
    -- 应用设置
    theme VARCHAR(20) DEFAULT 'auto', -- light, dark, auto
    search_radius_km INTEGER DEFAULT 10,
    
    -- 扩展设置
    custom_settings JSONB DEFAULT '{}',
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束
    UNIQUE(user_id)
);

-- ============================================================================
-- 索引创建
-- ============================================================================

-- 用户表索引
CREATE INDEX idx_users_user_id ON users(user_id);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_users_user_type ON users(user_type);
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_updated_at ON users(updated_at);
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

-- 用户位置表索引
CREATE INDEX idx_user_locations_user_id ON user_locations(user_id);
CREATE INDEX idx_user_locations_is_current ON user_locations(is_current) WHERE is_current = true;
CREATE INDEX idx_user_locations_location ON user_locations USING GIST(location);
CREATE INDEX idx_user_locations_created_at ON user_locations(created_at);

-- 用户会话表索引
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_session_token ON user_sessions(session_token);
CREATE INDEX idx_user_sessions_is_active ON user_sessions(is_active);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

-- 用户偏好设置表索引
CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

-- ============================================================================
-- 触发器创建
-- ============================================================================

-- 用户表更新时间戳触发器
CREATE TRIGGER users_updated_at_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 用户位置表更新时间戳触发器
CREATE TRIGGER user_locations_updated_at_trigger
    BEFORE UPDATE ON user_locations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 用户偏好设置表更新时间戳触发器
CREATE TRIGGER user_preferences_updated_at_trigger
    BEFORE UPDATE ON user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 用户审计触发器
CREATE TRIGGER users_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();

-- ============================================================================
-- 初始数据
-- ============================================================================

-- 创建系统管理员用户
INSERT INTO users (username, email, user_type, nickname, is_verified)
VALUES ('system', 'system@hotspot.local', 'admin', '系统管理员', true)
ON CONFLICT (username) DO NOTHING;

-- 为系统用户创建默认偏好设置
INSERT INTO user_preferences (user_id, profile_visibility, location_sharing)
SELECT id, 'private', false FROM users WHERE username = 'system'
ON CONFLICT (user_id) DO NOTHING;
