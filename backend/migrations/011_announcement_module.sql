-- 011_announcement_module.sql
-- 系统公告模块

-- ============================================================
-- 系统公告
-- ============================================================
CREATE TABLE system_announcements (
    id BIGSERIAL PRIMARY KEY,
    announcement_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL, -- 'info', 'warning', 'error'
    action_url TEXT,
    created_by BIGINT NOT NULL, -- admin user_id
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    send_push BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_announcement_severity CHECK (severity IN ('info', 'warning', 'error'))
);

CREATE INDEX idx_announcements_created ON system_announcements (created_at DESC) WHERE is_deleted = FALSE;
