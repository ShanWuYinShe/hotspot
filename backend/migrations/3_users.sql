-- ============================================================================
-- 04. 用户系统表
-- ============================================================================
-- 
-- 功能职责：
-- - 定义用户相关的所有表结构
-- - 实现用户ID编码规范
-- - 支持用户设置、活动记录、关系管理
-- - 提供地理位置支持
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql
--
-- ============================================================================

-- 用户表
CREATE TABLE users (
    id BIGINT PRIMARY KEY DEFAULT generate_user_id(1), -- 编码用户类型的主键（默认临时用户）
    login_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 登录标识符（临时用户为UUID，正式用户为邮箱/手机号）
    nickname TEXT NOT NULL,                      -- 用户昵称
    avatar_url TEXT,                             -- 头像URL
    bio TEXT,                                    -- 个人简介
    password_hash TEXT,                          -- 密码哈希（临时用户为NULL）
    is_verified BOOLEAN NOT NULL DEFAULT false, -- 是否已验证
    verified_at TIMESTAMP WITH TIME ZONE,       -- 验证时间
    is_active BOOLEAN NOT NULL DEFAULT true,    -- 账户是否激活
    is_deleted BOOLEAN NOT NULL DEFAULT false,  -- 软删除标记
    -- 位置信息
    latitude DOUBLE PRECISION,                   -- 当前纬度
    longitude DOUBLE PRECISION,                  -- 当前经度
    location GEOGRAPHY(POINT, 4326),            -- PostGIS地理位置点
    location_name TEXT,                          -- 当前位置名称/地址
    location_accuracy DOUBLE PRECISION,         -- 位置精度（米）
    location_updated_at TIMESTAMP WITH TIME ZONE, -- 位置更新时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE         -- 删除时间
);

-- 用户设置表
CREATE TABLE user_settings (
    id BIGSERIAL PRIMARY KEY,                    -- 设置ID（自增主键，用于高性能分页）
    setting_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 设置唯一标识符（对外API使用，保护数据隐私）
    user_id BIGINT NOT NULL,                    -- 用户ID
    privacy_level TEXT DEFAULT 'public',        -- 隐私级别：'public'|'friends'|'private'
    location_sharing BOOLEAN DEFAULT true,      -- 位置分享开关
    notification_enabled BOOLEAN DEFAULT true,  -- 通知开关
    mention_notifications BOOLEAN DEFAULT true, -- 提及通知开关
    group_invite_notifications BOOLEAN DEFAULT true, -- 群组邀请通知开关
    moment_notifications BOOLEAN DEFAULT true,  -- 生活瞬间通知开关
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    
    UNIQUE(user_id)                             -- 唯一约束
);

-- 用户活动记录表（分区表）
CREATE TABLE user_activity (
    id BIGSERIAL,                               -- 活动记录ID（自增主键，用于高性能分页）
    activity_id TEXT NOT NULL DEFAULT generate_unique_id(), -- 活动唯一标识符（对外API使用，保护数据隐私）
    user_id BIGINT NOT NULL,                    -- 用户ID
    activity_type activity_type_enum NOT NULL,  -- 活动类型
    target_type TEXT,                          -- 目标类型：'group'|'moment'|'message'|'user'
    target_id TEXT,                            -- 目标ID（群组ID、瞬间ID、消息ID等）
    description TEXT,                          -- 活动描述
    metadata JSONB NOT NULL DEFAULT '{}',     -- 完整元数据（包含IP地址、设备信息、位置信息、用户代理等）
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 活动时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    PRIMARY KEY (id, created_at)               -- 复合主键，支持分区
) PARTITION BY RANGE (created_at);

-- 用户关系表（好友、关注、拉黑）
CREATE TABLE user_relationships (
    id BIGSERIAL PRIMARY KEY,                   -- 关系ID（自增主键，用于高性能分页）
    relationship_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 关系唯一标识符（对外API使用，保护数据隐私）
    from_user_id BIGINT NOT NULL,              -- 发起用户ID
    to_user_id BIGINT NOT NULL,                -- 目标用户ID
    relationship_type relationship_type_enum NOT NULL, -- 关系类型
    relationship_status relationship_status_enum NOT NULL DEFAULT 'pending', -- 关系状态
    message TEXT,                              -- 申请消息（好友申请时）
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(from_user_id, to_user_id, relationship_type) -- 防止重复关系
);

-- 用户会话表（登录会话管理）
CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,                   -- 会话ID（自增主键）
    session_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 会话唯一标识符
    user_id BIGINT NOT NULL,                   -- 用户ID
    device_id TEXT,                            -- 设备标识
    device_info JSONB,                         -- 设备信息
    ip_address INET,                           -- IP地址
    user_agent TEXT,                           -- 用户代理
    refresh_token_hash TEXT,                   -- 刷新令牌哈希
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 会话是否活跃
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 最后活动时间
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL, -- 过期时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 用户位置历史表（可选，用于位置轨迹分析）
CREATE TABLE user_location_history (
    id BIGSERIAL PRIMARY KEY,                   -- 位置记录ID
    user_id BIGINT NOT NULL,                   -- 用户ID
    latitude DOUBLE PRECISION NOT NULL,        -- 纬度
    longitude DOUBLE PRECISION NOT NULL,       -- 经度
    location GEOGRAPHY(POINT, 4326) NOT NULL,  -- PostGIS地理位置点
    location_name TEXT,                        -- 位置名称
    accuracy DOUBLE PRECISION,                 -- 位置精度（米）
    altitude DOUBLE PRECISION,                 -- 海拔高度（米）
    speed DOUBLE PRECISION,                    -- 移动速度（米/秒）
    heading DOUBLE PRECISION,                  -- 移动方向（度）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() -- 记录时间
);

-- 添加外键约束
ALTER TABLE user_settings ADD CONSTRAINT fk_user_settings_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_relationships ADD CONSTRAINT fk_user_relationships_from_user_id 
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_relationships ADD CONSTRAINT fk_user_relationships_to_user_id 
    FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_sessions ADD CONSTRAINT fk_user_sessions_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_location_history ADD CONSTRAINT fk_user_location_history_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 添加检查约束
ALTER TABLE users ADD CONSTRAINT check_users_nickname_length 
    CHECK (length(nickname) >= 1 AND length(nickname) <= 50);

ALTER TABLE users ADD CONSTRAINT check_users_bio_length 
    CHECK (bio IS NULL OR length(bio) <= 500);

ALTER TABLE users ADD CONSTRAINT check_users_location_coordinates 
    CHECK ((latitude IS NULL AND longitude IS NULL) OR 
           (latitude IS NOT NULL AND longitude IS NOT NULL AND 
            latitude >= -90 AND latitude <= 90 AND 
            longitude >= -180 AND longitude <= 180));

ALTER TABLE user_relationships ADD CONSTRAINT check_user_relationships_not_self 
    CHECK (from_user_id != to_user_id);

-- 添加表注释
COMMENT ON TABLE users IS '用户基础信息表，支持用户ID编码规范和地理位置功能';
COMMENT ON TABLE user_settings IS '用户设置表，存储个人偏好和隐私设置';
COMMENT ON TABLE user_activity IS '用户活动记录表，分区存储用户操作日志';
COMMENT ON TABLE user_relationships IS '用户关系表，管理好友、关注、拉黑等关系';
COMMENT ON TABLE user_sessions IS '用户会话表，管理登录会话和设备信息';
COMMENT ON TABLE user_location_history IS '用户位置历史表，记录位置变化轨迹';

-- 添加列注释
COMMENT ON COLUMN users.id IS '用户ID，使用编码规范：高8位表示用户类型，低56位为序列号';
COMMENT ON COLUMN users.login_id IS '登录标识符，临时用户为UUID，正式用户为邮箱/手机号';
COMMENT ON COLUMN users.location IS 'PostGIS地理位置点，用于地理查询和距离计算';
COMMENT ON COLUMN user_activity.metadata IS '活动元数据，包含IP、设备信息、位置等详细信息';
COMMENT ON COLUMN user_relationships.relationship_type IS '关系类型：friend=好友, follow=关注, block=拉黑';
COMMENT ON COLUMN user_relationships.relationship_status IS '关系状态：pending=待处理, accepted=已接受, rejected=已拒绝, active=活跃';
