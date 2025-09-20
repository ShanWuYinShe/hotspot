-- ============================================================================
-- 05. 群组系统表（统一消息架构）
-- ============================================================================
-- 
-- 功能职责：
-- - 实现统一消息架构（私聊和群聊统一管理）
-- - 定义群组相关的所有表结构
-- - 支持群组成员管理和权限控制
-- - 提供地理位置群组功能
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql, 04_users.sql
--
-- ============================================================================

-- 群组表（统一消息架构核心表）
CREATE TABLE groups (
    id BIGSERIAL PRIMARY KEY,                   -- 群组ID（自增主键，用于高性能分页）
    group_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 群组唯一标识符（对外API使用，保护数据隐私）
    name TEXT NOT NULL,                        -- 群组名称
    description TEXT,                          -- 群组描述
    avatar_url TEXT,                           -- 群组头像URL
    owner_id BIGINT NOT NULL,                  -- 群主ID
    group_type group_type_enum NOT NULL DEFAULT 'public_chat', -- 群组类型
    is_public BOOLEAN NOT NULL DEFAULT true,   -- 是否公开（可搜索）
    password_hash TEXT,                        -- 群组密码哈希（私有群组）
    max_members INTEGER NOT NULL DEFAULT 500,  -- 最大成员数
    member_count INTEGER NOT NULL DEFAULT 1,   -- 当前成员数
    -- 地理位置信息（用于位置群组）
    latitude DOUBLE PRECISION,                 -- 群组纬度
    longitude DOUBLE PRECISION,                -- 群组经度
    location GEOGRAPHY(POINT, 4326),          -- PostGIS地理位置点
    location_name TEXT,                        -- 位置名称
    location_radius DOUBLE PRECISION DEFAULT 1000, -- 位置半径（米）
    -- 状态和时间
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 群组是否活跃
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 最后活动时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 群组成员表
CREATE TABLE group_members (
    id BIGSERIAL PRIMARY KEY,                   -- 成员记录ID（自增主键，用于高性能分页）
    member_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 成员唯一标识符（对外API使用，保护数据隐私）
    group_id TEXT NOT NULL,                    -- 群组ID（外键关联groups.group_id）
    user_id BIGINT NOT NULL,                   -- 用户ID
    role member_role_enum NOT NULL DEFAULT 'member', -- 成员角色
    status member_status_enum NOT NULL DEFAULT 'active', -- 成员状态
    nickname TEXT,                             -- 群内昵称（可选）
    muted_until TIMESTAMP WITH TIME ZONE,      -- 禁言到期时间
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 加入时间
    left_at TIMESTAMP WITH TIME ZONE,          -- 离开时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(group_id, user_id)                  -- 防止重复加入
);

-- 群组邀请表
CREATE TABLE group_invitations (
    id BIGSERIAL PRIMARY KEY,                   -- 邀请ID（自增主键）
    invitation_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 邀请唯一标识符
    group_id TEXT NOT NULL,                    -- 群组ID
    inviter_id BIGINT NOT NULL,                -- 邀请人ID
    invitee_id BIGINT NOT NULL,                -- 被邀请人ID
    message TEXT,                              -- 邀请消息
    status TEXT NOT NULL DEFAULT 'pending',    -- 邀请状态：'pending'|'accepted'|'rejected'|'expired'
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL, -- 邀请过期时间
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(group_id, invitee_id, status)       -- 防止重复邀请（同状态）
);

-- 群组设置表
CREATE TABLE group_settings (
    id BIGSERIAL PRIMARY KEY,                   -- 设置ID
    group_id TEXT UNIQUE NOT NULL,             -- 群组ID
    allow_member_invite BOOLEAN NOT NULL DEFAULT true, -- 允许成员邀请他人
    allow_member_modify_info BOOLEAN NOT NULL DEFAULT false, -- 允许成员修改群信息
    message_approval_required BOOLEAN NOT NULL DEFAULT false, -- 消息需要审核
    auto_delete_messages_days INTEGER,         -- 消息自动删除天数（NULL表示不删除）
    welcome_message TEXT,                      -- 欢迎消息
    rules TEXT,                                -- 群规
    tags TEXT[],                               -- 群组标签
    metadata JSONB NOT NULL DEFAULT '{}',     -- 其他设置元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 群组统计表（用于性能优化）
CREATE TABLE group_statistics (
    id BIGSERIAL PRIMARY KEY,                   -- 统计ID
    group_id TEXT UNIQUE NOT NULL,             -- 群组ID
    total_messages BIGINT NOT NULL DEFAULT 0,  -- 总消息数
    total_members INTEGER NOT NULL DEFAULT 0,  -- 总成员数
    active_members_today INTEGER NOT NULL DEFAULT 0, -- 今日活跃成员数
    active_members_week INTEGER NOT NULL DEFAULT 0,  -- 本周活跃成员数
    active_members_month INTEGER NOT NULL DEFAULT 0, -- 本月活跃成员数
    last_message_at TIMESTAMP WITH TIME ZONE,  -- 最后消息时间
    last_calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 最后计算时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 添加外键约束
ALTER TABLE groups ADD CONSTRAINT fk_groups_owner_id 
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE group_members ADD CONSTRAINT fk_group_members_group_id 
    FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE;

ALTER TABLE group_members ADD CONSTRAINT fk_group_members_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE group_invitations ADD CONSTRAINT fk_group_invitations_group_id 
    FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE;

ALTER TABLE group_invitations ADD CONSTRAINT fk_group_invitations_inviter_id 
    FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE group_invitations ADD CONSTRAINT fk_group_invitations_invitee_id 
    FOREIGN KEY (invitee_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE group_settings ADD CONSTRAINT fk_group_settings_group_id 
    FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE;

ALTER TABLE group_statistics ADD CONSTRAINT fk_group_statistics_group_id 
    FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE;

-- 添加检查约束
ALTER TABLE groups ADD CONSTRAINT check_groups_name_length 
    CHECK (length(name) >= 1 AND length(name) <= 100);

ALTER TABLE groups ADD CONSTRAINT check_groups_description_length 
    CHECK (description IS NULL OR length(description) <= 500);

ALTER TABLE groups ADD CONSTRAINT check_groups_max_members 
    CHECK (max_members > 0 AND max_members <= 10000);

ALTER TABLE groups ADD CONSTRAINT check_groups_member_count 
    CHECK (member_count >= 0 AND member_count <= max_members);

ALTER TABLE groups ADD CONSTRAINT check_groups_location_coordinates 
    CHECK ((latitude IS NULL AND longitude IS NULL) OR 
           (latitude IS NOT NULL AND longitude IS NOT NULL AND 
            latitude >= -90 AND latitude <= 90 AND 
            longitude >= -180 AND longitude <= 180));

ALTER TABLE groups ADD CONSTRAINT check_groups_location_radius 
    CHECK (location_radius IS NULL OR (location_radius > 0 AND location_radius <= 100000));

ALTER TABLE group_members ADD CONSTRAINT check_group_members_nickname_length 
    CHECK (nickname IS NULL OR (length(nickname) >= 1 AND length(nickname) <= 50));

ALTER TABLE group_invitations ADD CONSTRAINT check_group_invitations_expires_at 
    CHECK (expires_at > created_at);

-- 添加表注释
COMMENT ON TABLE groups IS '群组表，统一消息架构的核心表，支持公开群聊和私聊';
COMMENT ON TABLE group_members IS '群组成员表，管理群组成员关系和权限';
COMMENT ON TABLE group_invitations IS '群组邀请表，管理群组邀请流程';
COMMENT ON TABLE group_settings IS '群组设置表，存储群组配置和规则';
COMMENT ON TABLE group_statistics IS '群组统计表，用于性能优化和数据分析';

-- 添加列注释
COMMENT ON COLUMN groups.group_id IS '群组唯一标识符，对外API使用';
COMMENT ON COLUMN groups.group_type IS '群组类型：public_chat=公开群聊, private_chat=私聊';
COMMENT ON COLUMN groups.location IS 'PostGIS地理位置点，用于位置群组功能';
COMMENT ON COLUMN groups.location_radius IS '位置群组的有效半径（米）';
COMMENT ON COLUMN group_members.role IS '成员角色：owner=群主, admin=管理员, member=普通成员';
COMMENT ON COLUMN group_members.status IS '成员状态：active=活跃, muted=禁言, banned=封禁, left=已退出';
COMMENT ON COLUMN group_settings.metadata IS '群组其他设置的JSON元数据';
COMMENT ON COLUMN group_statistics.total_messages IS '群组总消息数，用于统计和排序';
