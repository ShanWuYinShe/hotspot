-- ============================================================
-- 007_admin_module.sql
-- 管理模块：RBAC、审计日志、系统公告、系统配置
-- ============================================================

-- ─── RBAC：角色 ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS admin_roles (
    role_id     UUID        PRIMARY KEY DEFAULT uuidv7(),
    name        VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    is_system   BOOLEAN     NOT NULL DEFAULT FALSE, -- 系统内置角色不可删除
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER tr_admin_roles_updated_at
    BEFORE UPDATE ON admin_roles FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── RBAC：权限字典 ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS admin_permissions (
    permission_id   UUID        PRIMARY KEY DEFAULT uuidv7(),
    key             VARCHAR(100) UNIQUE NOT NULL,   -- 如 'admin.users.ban'
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    module          VARCHAR(50) NOT NULL,            -- 所属模块
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_admin_perms_module ON admin_permissions(module);

-- ─── RBAC：角色 ↔ 权限 ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS admin_role_permissions (
    role_id         UUID NOT NULL,
    permission_id   UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);
CREATE INDEX idx_role_perms_perm ON admin_role_permissions(permission_id);

-- ─── RBAC：管理员用户 ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS admin_users (
    user_id     BIGINT      PRIMARY KEY,            -- 关联 users.user_id
    role_id     UUID        NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    assigned_by BIGINT,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER tr_admin_users_updated_at
    BEFORE UPDATE ON admin_users FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── RBAC：用户直接权限（角色之外的补充权限）──────────────
CREATE TABLE IF NOT EXISTS admin_user_permissions (
    user_id         BIGINT NOT NULL,
    permission_id   UUID   NOT NULL,
    granted_by      BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, permission_id)
);
CREATE INDEX idx_admin_user_perms_perm ON admin_user_permissions(permission_id);

-- ─── 审计日志（分区，无 before_state / after_state）──────────
CREATE TABLE IF NOT EXISTS audit_logs_partitioned (
    log_id          UUID        NOT NULL DEFAULT uuidv7(),
    operator_id     BIGINT      NOT NULL,           -- 操作者（管理员）
    action          VARCHAR(50) NOT NULL,            -- ban_user, unban_user, delete_moment, ...
    target_type     VARCHAR(50) NOT NULL,            -- user, group, moment, comment, system, ...
    target_id       VARCHAR(100),
    details         JSONB,                           -- 操作细节（替代 before_state/after_state）
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (log_id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS audit_logs_default PARTITION OF audit_logs_partitioned DEFAULT;

CREATE INDEX idx_audit_operator ON audit_logs_partitioned(operator_id, created_at DESC);
CREATE INDEX idx_audit_action   ON audit_logs_partitioned(action, created_at DESC);
CREATE INDEX idx_audit_target   ON audit_logs_partitioned(target_type, target_id, created_at DESC);

-- ─── 系统公告 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS system_announcements (
    announcement_id UUID                    PRIMARY KEY DEFAULT uuidv7(),
    title           VARCHAR(200)            NOT NULL,
    content         TEXT                    NOT NULL,
    severity        announcement_severity   NOT NULL DEFAULT 'info',
    publisher_id    BIGINT                  NOT NULL,
    is_active       BOOLEAN                 NOT NULL DEFAULT TRUE,
    start_time      TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    end_time        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sys_announcements_active
    ON system_announcements(start_time, end_time) WHERE is_active = TRUE;

CREATE TRIGGER tr_sys_announcements_updated_at
    BEFORE UPDATE ON system_announcements FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 系统配置（键值对，仅管理员访问）────────────────────────
CREATE TABLE IF NOT EXISTS system_configs (
    config_key  VARCHAR(100) PRIMARY KEY,
    value       JSONB        NOT NULL DEFAULT '{}',
    description TEXT,
    updated_by  BIGINT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER tr_system_configs_updated_at
    BEFORE UPDATE ON system_configs FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 初始种子数据 ───────────────────────────────────────────

-- 系统内置角色
INSERT INTO admin_roles (name, description, is_system) VALUES
    ('super_admin', '超级管理员 - 拥有所有权限', TRUE),
    ('content_moderator', '内容审核员 - 管理用户内容和举报', TRUE),
    ('user_manager', '用户管理员 - 管理用户账号', TRUE)
ON CONFLICT (name) DO NOTHING;

-- 权限字典（按模块组织）
INSERT INTO admin_permissions (key, name, module, description) VALUES
    -- 用户管理
    ('admin.users.view',      '查看用户列表',  'users', '查看和搜索用户'),
    ('admin.users.ban',       '封禁用户',      'users', '封禁/解封用户账号'),
    ('admin.users.edit',      '编辑用户',      'users', '编辑用户信息'),
    ('admin.users.reset_password', '重置密码',  'users', '重置用户密码'),
    -- 内容审核
    ('admin.content.view',    '查看内容',      'content', '查看用户发布的内容'),
    ('admin.content.delete',  '删除内容',      'content', '删除违规内容'),
    ('admin.content.review',  '审核内容',      'content', '审核举报的内容'),
    -- 群组管理
    ('admin.groups.view',     '查看群组',      'groups', '查看和搜索群组'),
    ('admin.groups.dismiss',  '解散群组',      'groups', '解散群组'),
    ('admin.groups.edit',     '编辑群组',      'groups', '编辑群组信息'),
    -- 举报处理
    ('admin.reports.view',    '查看举报',      'reports', '查看举报列表'),
    ('admin.reports.handle',  '处理举报',      'reports', '审核和处理举报'),
    -- 审计日志
    ('admin.audit.view',      '查看审计日志',  'audit', '查看操作审计日志'),
    -- RBAC 管理
    ('admin.rbac.view',       '查看角色权限',  'rbac', '查看角色和权限'),
    ('admin.rbac.manage',     '管理角色权限',  'rbac', '创建/编辑/删除角色和权限'),
    -- 系统配置
    ('admin.config.view',     '查看系统配置',  'config', '查看系统配置项'),
    ('admin.config.edit',     '编辑系统配置',  'config', '修改系统配置项'),
    -- 系统公告
    ('admin.announcements.view',   '查看系统公告', 'announcements', '查看系统公告'),
    ('admin.announcements.manage', '管理系统公告', 'announcements', '创建/编辑/删除系统公告')
ON CONFLICT (key) DO NOTHING;

-- 为超级管理员赋予全部权限
INSERT INTO admin_role_permissions (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM admin_roles r, admin_permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- 为内容审核员赋予内容+举报权限
INSERT INTO admin_role_permissions (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM admin_roles r, admin_permissions p
WHERE r.name = 'content_moderator'
  AND p.module IN ('content', 'reports')
ON CONFLICT DO NOTHING;

-- 为用户管理员赋予用户管理权限
INSERT INTO admin_role_permissions (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM admin_roles r, admin_permissions p
WHERE r.name = 'user_manager'
  AND p.module = 'users'
ON CONFLICT DO NOTHING;
