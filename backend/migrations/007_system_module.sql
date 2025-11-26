-- 007_system_module.sql
-- 系统模块：举报、信息流、排行榜、系统配置

-- ============================================================
-- 举报
-- ============================================================
CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,
    report_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    reporter_id BIGINT NOT NULL,
    target_type report_target_type NOT NULL,
    target_id VARCHAR(255) NOT NULL,
    reason VARCHAR(255) NOT NULL,
    description TEXT,
    status report_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    handled_at TIMESTAMPTZ,
    handled_by BIGINT,
    handler_note TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_reports_cursor_id ON reports (cursor_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_reports_reporter ON reports (reporter_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_reports_status ON reports (status, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_reports_target ON reports (target_type, target_id) WHERE is_deleted = FALSE;

-- ============================================================
-- 信息流
-- ============================================================
CREATE TABLE feed_items (
    id BIGSERIAL PRIMARY KEY,
    feed_item_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
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

CREATE INDEX idx_feed_items_cursor_id ON feed_items (cursor_id) WHERE is_hidden = FALSE;
CREATE INDEX idx_feed_items_user_channel ON feed_items (user_id, channel, created_at DESC) WHERE is_hidden = FALSE;
CREATE INDEX idx_feed_items_expires ON feed_items (expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE feed_feedback (
    id BIGSERIAL PRIMARY KEY,
    feedback_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    feed_item_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_feedback_action CHECK (action IN ('hide', 'not_interested', 'clicked', 'viewed'))
);

CREATE INDEX idx_feed_feedback_feedback_id ON feed_feedback (feedback_id);
CREATE INDEX idx_feed_feedback_cursor_id ON feed_feedback (cursor_id);
CREATE INDEX idx_feed_feedback_feed_item ON feed_feedback (feed_item_id, action);
CREATE INDEX idx_feed_feedback_user ON feed_feedback (user_id, created_at DESC);

-- ============================================================
-- 排行榜
-- ============================================================
CREATE TABLE leaderboard_creator_scores (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    timeframe leaderboard_timeframe NOT NULL,
    scope leaderboard_scope NOT NULL,
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    rank INTEGER,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    location geography(POINT, 4326),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_creator_score UNIQUE (user_id, timeframe, scope)
);

CREATE INDEX idx_creator_scores_timeframe_scope ON leaderboard_creator_scores (timeframe, scope, score DESC);
CREATE INDEX idx_creator_leaderboard ON leaderboard_creator_scores (timeframe, scope, rank) WHERE rank IS NOT NULL;
CREATE INDEX idx_creator_scores_location ON leaderboard_creator_scores USING gist (location) WHERE location IS NOT NULL;

CREATE TABLE leaderboard_moment_scores (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL,
    timeframe leaderboard_timeframe NOT NULL,
    scope leaderboard_scope NOT NULL,
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    rank INTEGER,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    location geography(POINT, 4326),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_moment_score UNIQUE (moment_id, timeframe, scope)
);

CREATE INDEX idx_moment_scores_timeframe_scope ON leaderboard_moment_scores (timeframe, scope, score DESC);
CREATE INDEX idx_moment_leaderboard ON leaderboard_moment_scores (timeframe, scope, rank) WHERE rank IS NOT NULL;
CREATE INDEX idx_moment_scores_location ON leaderboard_moment_scores USING gist (location) WHERE location IS NOT NULL;

-- ============================================================
-- 系统配置
-- ============================================================
CREATE TABLE system_configs (
    id BIGSERIAL PRIMARY KEY,
    config_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
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
CREATE INDEX idx_system_configs_cursor_id ON system_configs (cursor_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_system_configs_category ON system_configs (category, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_system_configs_key ON system_configs (key) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_system_configs_updated_at
    BEFORE UPDATE ON system_configs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE system_config_history (
    id BIGSERIAL PRIMARY KEY,
    history_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    cursor_id UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    config_key VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT NOT NULL,
    changed_by BIGINT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_config_history_history_id ON system_config_history (history_id);
CREATE INDEX idx_config_history_cursor_id ON system_config_history (cursor_id);
CREATE INDEX idx_config_history_key ON system_config_history (config_key, changed_at DESC);
CREATE INDEX idx_config_history_changed_by ON system_config_history (changed_by, changed_at DESC);

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


