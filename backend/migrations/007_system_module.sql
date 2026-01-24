-- 007_system_module.sql
-- 系统模块：举报、信息流、系统配置、公告、文件上传

-- ============================================================
-- 举报
-- ============================================================
CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,
    report_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    reporter_id BIGINT NOT NULL,
    target_type report_target_type NOT NULL,
    target_id VARCHAR(255) NOT NULL,
    reason VARCHAR(255) NOT NULL,
    description TEXT,
    target_snapshot JSONB,
    status report_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    handled_at TIMESTAMPTZ,
    handled_by BIGINT,
    handler_note TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_reports_reporter ON reports (reporter_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_reports_status ON reports (status, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_reports_target ON reports (target_type, target_id) WHERE is_deleted = FALSE;

COMMENT ON COLUMN reports.target_snapshot IS '目标内容快照，包含 content, media_urls, owner_id';

-- ============================================================
-- 信息流
-- ============================================================
CREATE TABLE feed_items (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel feed_channel NOT NULL,
    item_type feed_item_type NOT NULL,
    moment_id UUID,
    group_chat_id UUID,
    reason TEXT,
    score DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    hidden_at TIMESTAMPTZ
);

CREATE INDEX idx_feed_items_user_channel ON feed_items (user_id, channel, created_at DESC) WHERE is_hidden = FALSE;
CREATE INDEX idx_feed_items_expires ON feed_items (expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE feed_feedback (
    id BIGSERIAL PRIMARY KEY,
    feedback_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    feed_item_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_feedback_action CHECK (action IN ('hide', 'not_interested', 'clicked', 'viewed'))
);

CREATE INDEX idx_feed_feedback_feedback_id ON feed_feedback (feedback_id);
CREATE INDEX idx_feed_feedback_feed_item ON feed_feedback (feed_item_id, action);
CREATE INDEX idx_feed_feedback_user ON feed_feedback (user_id, created_at DESC);

-- ============================================================
-- 系统配置
-- ============================================================
CREATE TABLE system_configs (
    id BIGSERIAL PRIMARY KEY,
    config_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    key VARCHAR(100) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    type VARCHAR(20) NOT NULL,
    category VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_config_type CHECK (type IN ('string', 'number', 'boolean'))
);

CREATE INDEX idx_system_configs_config_id ON system_configs (config_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_system_configs_category ON system_configs (category, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_system_configs_key ON system_configs (key) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_system_configs_updated_at
    BEFORE UPDATE ON system_configs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE system_config_history (
    id BIGSERIAL PRIMARY KEY,
    history_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    config_key VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT NOT NULL,
    changed_by BIGINT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_config_history_history_id ON system_config_history (history_id);
CREATE INDEX idx_config_history_key ON system_config_history (config_key, changed_at DESC);
CREATE INDEX idx_config_history_changed_by ON system_config_history (changed_by, changed_at DESC);

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

-- ============================================================
-- 上传文件（用于管理用户上传的文件，支持 file_id -> url 转换）
-- ============================================================
CREATE TABLE uploaded_files (
    id BIGSERIAL PRIMARY KEY,
    file_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    uploader_id BIGINT NOT NULL,
    category VARCHAR(50) NOT NULL,  -- 'avatar', 'group_avatar', 'moment_image', 'chat_image', 'chat_video', 'chat_audio', 'chat_file'
    original_name VARCHAR(255),
    storage_path VARCHAR(500) NOT NULL,  -- 存储路径
    url VARCHAR(500) NOT NULL,  -- 可访问的URL (注意：虽然保留此字段，但新设计中应通过 sign_url 动态生成，此处存 storage_path 或占位)
    thumbnail_url VARCHAR(500),  -- 缩略图URL（仅图片/视频）
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,  -- 文件大小（字节）
    width INTEGER,  -- 宽度（仅图片/视频）
    height INTEGER,  -- 高度（仅图片/视频）
    duration INTEGER,  -- 时长秒数（仅音频/视频）
    checksum VARCHAR(64),  -- SHA256哈希（用于去重）
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,  -- 临时文件过期时间
    is_used BOOLEAN NOT NULL DEFAULT FALSE,  -- 是否已被业务引用
    used_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_file_category CHECK (category IN ('avatar', 'group_avatar', 'moment_image', 'chat_image', 'chat_video', 'chat_audio', 'chat_file'))
);

CREATE INDEX idx_uploaded_files_file_id ON uploaded_files (file_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_uploaded_files_uploader ON uploaded_files (uploader_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_uploaded_files_checksum ON uploaded_files (checksum) WHERE checksum IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_uploaded_files_expires ON uploaded_files (expires_at) WHERE expires_at IS NOT NULL AND is_used = FALSE;

-- 大文件分片上传跟踪
CREATE TABLE upload_chunks (
    id BIGSERIAL PRIMARY KEY,
    upload_id UUID NOT NULL,  -- 上传会话ID
    part_number INTEGER NOT NULL,
    etag VARCHAR(64),
    size BIGINT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (upload_id, part_number)
);

CREATE INDEX idx_upload_chunks_upload_id ON upload_chunks (upload_id);

CREATE TABLE upload_sessions (
    id BIGSERIAL PRIMARY KEY,
    upload_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    uploader_id BIGINT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    chunk_size INTEGER NOT NULL DEFAULT 5242880,  -- 默认5MB
    total_chunks INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',  -- 'in_progress', 'completed', 'failed', 'aborted'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    CONSTRAINT check_upload_status CHECK (status IN ('in_progress', 'completed', 'failed', 'aborted'))
);

CREATE INDEX idx_upload_sessions_upload_id ON upload_sessions (upload_id);
CREATE INDEX idx_upload_sessions_uploader ON upload_sessions (uploader_id, created_at DESC);
CREATE INDEX idx_upload_sessions_expires ON upload_sessions (expires_at) WHERE status = 'in_progress';

CREATE TRIGGER trigger_upload_sessions_updated_at
    BEFORE UPDATE ON upload_sessions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 插入默认配置（修正为南天门坐标）
INSERT INTO system_configs (key, value, type, category, description) VALUES
    ('default_latitude', '35.154922', 'number', 'location', '默认纬度（南天门）'),
    ('default_longitude', '112.682524', 'number', 'location', '默认经度（南天门）'),
    ('default_location_name', '南天门', 'string', 'location', '默认位置名称'),
    ('location_change_threshold', '500', 'number', 'location', '位置变化记录阈值（米）'),
    ('access_token_ttl', '3600', 'number', 'auth', '访问令牌有效期（秒，1小时）'),
    ('refresh_token_ttl', '2592000', 'number', 'auth', '刷新令牌有效期（秒，30天）'),
    ('max_upload_size_avatar', '5242880', 'number', 'media', '头像最大上传大小（字节，5MB）'),
    ('max_upload_size_image', '10485760', 'number', 'media', '图片最大上传大小（字节，10MB）'),
    ('max_upload_size_file', '104857600', 'number', 'media', '文件最大上传大小（字节，100MB）'),
    ('pagination_default_limit', '20', 'number', 'pagination', '分页默认条数'),
    ('pagination_max_limit', '100', 'number', 'pagination', '分页最大条数'),
    ('search_rate_limit', '100', 'number', 'rate_limit', '搜索速率限制（次/小时）'),
    ('message_rate_limit', '1000', 'number', 'rate_limit', '消息速率限制（次/小时）');
