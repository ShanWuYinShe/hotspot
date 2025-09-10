-- ============================================================================
-- HotSpot 群组核心模块迁移 v2.0
-- ============================================================================
-- 
-- 功能职责：
-- - 创建群组核心表结构
-- - 群组成员管理
-- - 群组权限和角色
-- - 群组位置信息
--
-- 依赖关系：
-- - 依赖 00_foundation.sql 的基础设施
-- - 依赖 01_users_core.sql 的用户系统
--
-- ============================================================================

-- 群组主表
CREATE TABLE groups (
    -- 主键和基础信息
    id BIGSERIAL PRIMARY KEY,
    group_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    
    -- 群组基础信息
    name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar_url TEXT,
    
    -- 群组类型和可见性
    group_type group_type_enum NOT NULL DEFAULT 'public',
    visibility group_visibility_enum NOT NULL DEFAULT 'public',
    
    -- 群组设置
    max_members INTEGER DEFAULT 100,
    is_active BOOLEAN NOT NULL DEFAULT true,
    requires_approval BOOLEAN DEFAULT false,
    allow_member_invite BOOLEAN DEFAULT true,
    password_hash VARCHAR(255), -- 群组密码哈希，可选
    
    -- 群组元数据
    member_count INTEGER DEFAULT 0,
    tags TEXT[] DEFAULT '{}',
    settings JSONB DEFAULT '{}',
    
    -- 创建者和管理
    created_by BIGINT NOT NULL REFERENCES users(id),
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    CONSTRAINT groups_name_length CHECK (char_length(name) >= 2),
    CONSTRAINT groups_max_members_positive CHECK (max_members > 0),
    CONSTRAINT groups_member_count_non_negative CHECK (member_count >= 0)
);

-- 群组位置表
CREATE TABLE group_locations (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    
    -- 地理位置信息
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    radius_meters DOUBLE PRECISION DEFAULT 1000,
    
    -- 地址信息
    address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    postal_code VARCHAR(20),
    
    -- PostGIS地理点和区域
    location GEOGRAPHY(POINT, 4326) GENERATED ALWAYS AS (ST_Point(longitude, latitude)) STORED,
    coverage_area GEOGRAPHY(POLYGON, 4326),
    
    -- 位置元数据
    location_name VARCHAR(200),
    location_type VARCHAR(50) DEFAULT 'meeting_point', -- meeting_point, activity_area, service_zone
    is_primary BOOLEAN DEFAULT true,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT group_locations_valid_coordinates CHECK (validate_coordinates(latitude, longitude)),
    CONSTRAINT group_locations_radius_positive CHECK (radius_meters > 0),
    CONSTRAINT group_locations_unique_primary UNIQUE (group_id, is_primary) DEFERRABLE INITIALLY DEFERRED
);

-- 群组成员表
CREATE TABLE group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 成员角色和状态
    role group_role_enum NOT NULL DEFAULT 'member',
    status VARCHAR(20) DEFAULT 'active', -- active, pending, banned, left
    
    -- 成员权限
    can_invite BOOLEAN DEFAULT false,
    can_manage_members BOOLEAN DEFAULT false,
    can_edit_group BOOLEAN DEFAULT false,
    
    -- 成员统计
    message_count INTEGER DEFAULT 0,
    last_activity_at TIMESTAMP WITH TIME ZONE,
    
    -- 时间戳
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP WITH TIME ZONE,
    
    -- 唯一约束
    UNIQUE(group_id, user_id),
    
    -- 约束
    CONSTRAINT group_members_message_count_non_negative CHECK (message_count >= 0)
);

-- 群组邀请表
CREATE TABLE group_invitations (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    inviter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invitee_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    
    -- 邀请信息
    invitation_code TEXT UNIQUE,
    invitation_message TEXT,
    invitee_email VARCHAR(255),
    invitee_phone VARCHAR(20),
    
    -- 邀请状态
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, declined, expired, cancelled
    
    -- 邀请设置
    expires_at TIMESTAMP WITH TIME ZONE,
    max_uses INTEGER DEFAULT 1,
    used_count INTEGER DEFAULT 0,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    responded_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    CONSTRAINT group_invitations_max_uses_positive CHECK (max_uses > 0),
    CONSTRAINT group_invitations_used_count_non_negative CHECK (used_count >= 0),
    CONSTRAINT group_invitations_used_count_within_max CHECK (used_count <= max_uses),
    CONSTRAINT group_invitations_invitee_info CHECK (
        invitee_id IS NOT NULL OR invitee_email IS NOT NULL OR invitee_phone IS NOT NULL
    )
);

-- 群组设置表
CREATE TABLE group_settings (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    
    -- 消息设置
    message_retention_days INTEGER DEFAULT 365,
    allow_file_sharing BOOLEAN DEFAULT true,
    max_file_size_mb INTEGER DEFAULT 10,
    
    -- 成员设置
    auto_approve_join BOOLEAN DEFAULT false,
    member_can_add_members BOOLEAN DEFAULT false,
    member_can_create_events BOOLEAN DEFAULT true,
    
    -- 隐私设置
    searchable BOOLEAN DEFAULT true,
    show_member_list BOOLEAN DEFAULT true,
    allow_external_invites BOOLEAN DEFAULT true,
    
    -- 通知设置
    notify_on_join BOOLEAN DEFAULT true,
    notify_on_leave BOOLEAN DEFAULT false,
    notify_on_message BOOLEAN DEFAULT true,
    
    -- 扩展设置
    custom_settings JSONB DEFAULT '{}',
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束
    UNIQUE(group_id)
);

-- ============================================================================
-- 索引创建
-- ============================================================================

-- 群组表索引
CREATE INDEX idx_groups_group_id ON groups(group_id);
CREATE INDEX idx_groups_name ON groups(name);
CREATE INDEX idx_groups_group_type ON groups(group_type);
CREATE INDEX idx_groups_visibility ON groups(visibility);
CREATE INDEX idx_groups_is_active ON groups(is_active);
CREATE INDEX idx_groups_created_by ON groups(created_by);
CREATE INDEX idx_groups_created_at ON groups(created_at);
CREATE INDEX idx_groups_member_count ON groups(member_count);
CREATE INDEX idx_groups_tags ON groups USING GIN(tags);

-- 群组位置表索引
CREATE INDEX idx_group_locations_group_id ON group_locations(group_id);
CREATE INDEX idx_group_locations_location ON group_locations USING GIST(location);
CREATE INDEX idx_group_locations_is_primary ON group_locations(is_primary) WHERE is_primary = true;
CREATE INDEX idx_group_locations_location_type ON group_locations(location_type);

-- 群组成员表索引
CREATE INDEX idx_group_members_group_id ON group_members(group_id);
CREATE INDEX idx_group_members_user_id ON group_members(user_id);
CREATE INDEX idx_group_members_role ON group_members(role);
CREATE INDEX idx_group_members_status ON group_members(status);
CREATE INDEX idx_group_members_joined_at ON group_members(joined_at);
CREATE INDEX idx_group_members_last_activity ON group_members(last_activity_at);

-- 群组邀请表索引
CREATE INDEX idx_group_invitations_group_id ON group_invitations(group_id);
CREATE INDEX idx_group_invitations_inviter_id ON group_invitations(inviter_id);
CREATE INDEX idx_group_invitations_invitee_id ON group_invitations(invitee_id);
CREATE INDEX idx_group_invitations_status ON group_invitations(status);
CREATE INDEX idx_group_invitations_expires_at ON group_invitations(expires_at);
CREATE INDEX idx_group_invitations_code ON group_invitations(invitation_code);

-- 群组设置表索引
CREATE INDEX idx_group_settings_group_id ON group_settings(group_id);

-- ============================================================================
-- 触发器创建
-- ============================================================================

-- 群组表更新时间戳触发器
CREATE TRIGGER groups_updated_at_trigger
    BEFORE UPDATE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 群组位置表更新时间戳触发器
CREATE TRIGGER group_locations_updated_at_trigger
    BEFORE UPDATE ON group_locations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 群组成员表更新时间戳触发器
CREATE TRIGGER group_members_updated_at_trigger
    BEFORE UPDATE ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 群组设置表更新时间戳触发器
CREATE TRIGGER group_settings_updated_at_trigger
    BEFORE UPDATE ON group_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 群组审计触发器
CREATE TRIGGER groups_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();

-- 群组成员审计触发器
CREATE TRIGGER group_members_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();

-- ============================================================================
-- 群组成员数量维护函数和触发器
-- ============================================================================

-- 更新群组成员数量函数
CREATE OR REPLACE FUNCTION update_group_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE groups 
        SET member_count = member_count + 1 
        WHERE id = NEW.group_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE groups 
        SET member_count = member_count - 1 
        WHERE id = OLD.group_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 群组成员数量维护触发器
CREATE TRIGGER group_member_count_trigger
    AFTER INSERT OR DELETE ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION update_group_member_count();

-- ============================================================================
-- 初始化群组设置函数
-- ============================================================================

-- 为新群组创建默认设置函数
CREATE OR REPLACE FUNCTION create_default_group_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO group_settings (group_id)
    VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 群组创建时自动创建设置触发器
CREATE TRIGGER create_group_settings_trigger
    AFTER INSERT ON groups
    FOR EACH ROW
    EXECUTE FUNCTION create_default_group_settings();
