-- ============================================================
-- 007_admin_module.sql
-- 管理模块：RBAC、审计、系统配置、公告
-- ============================================================

-- 1. 管理员用户表 (无外键)
CREATE TABLE IF NOT EXISTS admin_users (
    user_id BIGINT PRIMARY KEY, -- 对应 users.user_id
    role_id UUID, -- 默认角色ID (可选)
    is_super_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. 角色表 (无外键)
CREATE TABLE IF NOT EXISTS admin_roles (
    role_id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE, -- 系统预置角色
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. 权限表 (无外键)
CREATE TABLE IF NOT EXISTS admin_permissions (
    permission_key VARCHAR(100) PRIMARY KEY, -- 如: user:create
    name VARCHAR(50) NOT NULL,
    description TEXT,
    module VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. 角色-权限关联表 (无外键)
CREATE TABLE IF NOT EXISTS admin_role_permissions (
    role_id UUID NOT NULL,
    permission_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_key)
);

-- 5. 用户-角色关联表 (无外键)
CREATE TABLE IF NOT EXISTS admin_user_roles (
    user_id BIGINT NOT NULL,
    role_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- 6. 审计日志表 (分区表)
CREATE TABLE IF NOT EXISTS audit_logs_partitioned (
    log_id UUID NOT NULL DEFAULT uuidv7(),
    action VARCHAR(100) NOT NULL,
    operator_id BIGINT NOT NULL,
    target_type VARCHAR(50),
    target_id VARCHAR(100),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (log_id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建默认分区
CREATE TABLE IF NOT EXISTS audit_logs_default PARTITION OF audit_logs_partitioned DEFAULT;

-- 创建初始分区 (2025, 2026)
DO $$
BEGIN
    PERFORM create_partition('audit_logs_2025', 'audit_logs_partitioned', '2025-01-01', '2026-01-01');
    PERFORM create_partition('audit_logs_2026', 'audit_logs_partitioned', '2026-01-01', '2027-01-01');
END $$;

-- 索引
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator ON audit_logs_partitioned(operator_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs_partitioned(action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target ON audit_logs_partitioned(target_type, target_id);

-- 重命名
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'audit_logs') THEN
        ALTER TABLE audit_logs_partitioned RENAME TO audit_logs;
    END IF;
END $$;

-- 7. 系统配置表 (无外键)
CREATE TABLE IF NOT EXISTS system_configs (
    config_key VARCHAR(100) PRIMARY KEY,
    config_value JSONB NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE, -- 是否公开给前端
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by BIGINT,
    
    CONSTRAINT check_config_value_not_null CHECK (jsonb_typeof(config_value) != 'null')
);

-- 8. 系统公告表 (无外键)
CREATE TABLE IF NOT EXISTS announcements (
    announcement_id UUID PRIMARY KEY DEFAULT uuidv7(),
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(20) DEFAULT 'system', -- system, maintenance, activity
    target_user_type VARCHAR(20) DEFAULT 'all', -- all, user, admin
    
    start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_at TIMESTAMPTZ,
    
    is_published BOOLEAN DEFAULT FALSE,
    publisher_id BIGINT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_announcements_active ON announcements(start_at, end_at, is_published);
