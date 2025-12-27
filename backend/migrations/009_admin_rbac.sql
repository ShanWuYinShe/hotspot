-- 009_admin_rbac.sql
-- 管理员 RBAC：超级管理员 + 角色/权限表

-- ============================================================
-- 管理员用户表（只记录“是否为管理员/是否超级管理员”）
-- ============================================================
CREATE TABLE admin_users (
    user_id BIGINT PRIMARY KEY,
    is_super BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_admin_users_is_super ON admin_users (is_super) WHERE is_super = TRUE;

CREATE TRIGGER trigger_admin_users_updated_at
    BEFORE UPDATE ON admin_users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 权限字典（固定 key，可扩展）
-- ============================================================
CREATE TABLE admin_permissions (
    key VARCHAR(100) PRIMARY KEY,
    description TEXT NOT NULL
);

-- ============================================================
-- 角色表（用于给管理员分组授权）
-- ============================================================
CREATE TABLE admin_roles (
    id BIGSERIAL PRIMARY KEY,
    role_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_admin_roles_role_id ON admin_roles (role_id);

CREATE TRIGGER trigger_admin_roles_updated_at
    BEFORE UPDATE ON admin_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 角色-权限映射
-- ============================================================
CREATE TABLE admin_role_permissions (
    role_id UUID NOT NULL,
    permission_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_key)
);

CREATE INDEX idx_admin_role_permissions_role ON admin_role_permissions (role_id);
CREATE INDEX idx_admin_role_permissions_perm ON admin_role_permissions (permission_key);

-- ============================================================
-- 管理员-角色映射
-- ============================================================
CREATE TABLE admin_user_roles (
    user_id BIGINT NOT NULL,
    role_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_admin_user_roles_user ON admin_user_roles (user_id);
CREATE INDEX idx_admin_user_roles_role ON admin_user_roles (role_id);

-- ============================================================
-- 预置权限字典与角色（可按需扩展）
-- ============================================================
INSERT INTO admin_permissions (key, description) VALUES
    ('admin.rbac.manage', '管理员权限管理（授权/角色/权限）'),
    ('admin.reports.read', '查看举报列表'),
    ('admin.reports.write', '处理举报（resolve/dismiss/escalate）'),
    ('admin.config.read', '查看系统配置（list/detail/history/changes）'),
    ('admin.config.write', '修改系统配置（create/update/delete）')
ON CONFLICT (key) DO NOTHING;

-- 预置角色
INSERT INTO admin_roles (name, description) VALUES
    ('moderator', '内容审核员（举报处理）'),
    ('config_manager', '系统配置管理员')
ON CONFLICT (name) DO NOTHING;

-- 预置角色权限
INSERT INTO admin_role_permissions (role_id, permission_key)
SELECT r.role_id, p.key
FROM admin_roles r
JOIN admin_permissions p ON (
    (r.name = 'moderator' AND p.key IN ('admin.reports.read', 'admin.reports.write'))
 OR (r.name = 'config_manager' AND p.key IN ('admin.config.read', 'admin.config.write'))
)
ON CONFLICT (role_id, permission_key) DO NOTHING;

