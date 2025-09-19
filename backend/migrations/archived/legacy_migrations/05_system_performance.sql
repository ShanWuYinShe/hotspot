-- ============================================================================
-- HotSpot 系统性能优化模块迁移 v2.0
-- ============================================================================
-- 
-- 功能职责：
-- - 系统性能监控表
-- - 缓存管理表
-- - 任务队列表
-- - 系统监控表
-- - 高级索引优化
--
-- 依赖关系：
-- - 依赖所有核心模块的基础表结构
--
-- ============================================================================

-- 系统配置表
CREATE TABLE system_configs (
    id BIGSERIAL PRIMARY KEY,
    
    -- 配置信息
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT,
    config_type VARCHAR(20) DEFAULT 'string', -- 'string', 'number', 'boolean', 'json'
    
    -- 配置元数据
    description TEXT,
    category VARCHAR(50),
    is_public BOOLEAN DEFAULT false,
    is_editable BOOLEAN DEFAULT true,
    
    -- 版本控制
    version INTEGER DEFAULT 1,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT system_configs_key_not_empty CHECK (char_length(config_key) > 0)
);

-- 缓存管理表
CREATE TABLE cache_entries (
    id BIGSERIAL PRIMARY KEY,
    
    -- 缓存键值
    cache_key VARCHAR(255) UNIQUE NOT NULL,
    cache_value TEXT,
    
    -- 缓存元数据
    cache_type VARCHAR(50),
    namespace VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    
    -- 缓存控制
    expires_at TIMESTAMP WITH TIME ZONE,
    hit_count BIGINT DEFAULT 0,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT cache_entries_hit_count_non_negative CHECK (hit_count >= 0)
);

-- 任务队列表
CREATE TABLE job_queue (
    id BIGSERIAL PRIMARY KEY,
    
    -- 任务信息
    job_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    job_type VARCHAR(100) NOT NULL,
    job_data JSONB NOT NULL DEFAULT '{}',
    
    -- 任务状态
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'cancelled'
    priority INTEGER DEFAULT 0,
    
    -- 执行控制
    max_attempts INTEGER DEFAULT 3,
    attempt_count INTEGER DEFAULT 0,
    
    -- 调度信息
    scheduled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- 错误信息
    error_message TEXT,
    error_details JSONB,
    
    -- 执行统计
    execution_time_ms BIGINT,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT job_queue_max_attempts_positive CHECK (max_attempts > 0),
    CONSTRAINT job_queue_attempt_count_non_negative CHECK (attempt_count >= 0),
    CONSTRAINT job_queue_attempt_count_within_max CHECK (attempt_count <= max_attempts),
    CONSTRAINT job_queue_execution_time_non_negative CHECK (execution_time_ms IS NULL OR execution_time_ms >= 0)
);

-- 注意：通知系统采用纯 WebSocket 实时推送，不使用数据库持久化
-- 此处不创建 notifications 表，符合项目架构设计

-- WebSocket连接表
CREATE TABLE websocket_connections (
    id BIGSERIAL PRIMARY KEY,
    
    -- 连接信息
    connection_id TEXT UNIQUE NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    
    -- 连接状态
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'inactive', 'disconnected'
    
    -- 连接元数据
    ip_address INET,
    user_agent TEXT,
    device_info JSONB DEFAULT '{}',
    
    -- 订阅信息
    subscribed_rooms TEXT[] DEFAULT '{}',
    
    -- 连接统计
    message_count BIGINT DEFAULT 0,
    last_ping_at TIMESTAMP WITH TIME ZONE,
    
    -- 时间戳
    connected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    disconnected_at TIMESTAMP WITH TIME ZONE,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT websocket_connections_message_count_non_negative CHECK (message_count >= 0)
);

-- API请求日志表
CREATE TABLE api_request_logs (
    id BIGSERIAL PRIMARY KEY,
    
    -- 请求信息
    request_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    
    -- 用户信息
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    
    -- 请求元数据
    ip_address INET,
    user_agent TEXT,
    
    -- 响应信息
    status_code INTEGER,
    response_time_ms BIGINT,
    response_size_bytes BIGINT,
    
    -- 错误信息
    error_message TEXT,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT api_request_logs_response_time_non_negative CHECK (response_time_ms IS NULL OR response_time_ms >= 0),
    CONSTRAINT api_request_logs_response_size_non_negative CHECK (response_size_bytes IS NULL OR response_size_bytes >= 0)
);

-- 系统指标表
CREATE TABLE system_metrics (
    id BIGSERIAL PRIMARY KEY,
    
    -- 指标信息
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    metric_type VARCHAR(20) DEFAULT 'gauge', -- 'gauge', 'counter', 'histogram'
    
    -- 指标标签
    labels JSONB DEFAULT '{}',
    
    -- 时间戳
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT system_metrics_name_not_empty CHECK (char_length(metric_name) > 0)
);

-- ============================================================================
-- 高级索引创建
-- ============================================================================

-- 系统配置表索引
CREATE INDEX idx_system_configs_key ON system_configs(config_key);
CREATE INDEX idx_system_configs_category ON system_configs(category);
CREATE INDEX idx_system_configs_is_public ON system_configs(is_public);

-- 缓存管理表索引
CREATE INDEX idx_cache_entries_key ON cache_entries(cache_key);
CREATE INDEX idx_cache_entries_namespace ON cache_entries(namespace);
CREATE INDEX idx_cache_entries_expires_at ON cache_entries(expires_at);
CREATE INDEX idx_cache_entries_tags ON cache_entries USING GIN(tags);
CREATE INDEX idx_cache_entries_last_accessed ON cache_entries(last_accessed_at);

-- 任务队列表索引
CREATE INDEX idx_job_queue_job_id ON job_queue(job_id);
CREATE INDEX idx_job_queue_status ON job_queue(status);
CREATE INDEX idx_job_queue_job_type ON job_queue(job_type);
CREATE INDEX idx_job_queue_priority ON job_queue(priority DESC);
CREATE INDEX idx_job_queue_scheduled_at ON job_queue(scheduled_at);
CREATE INDEX idx_job_queue_status_priority ON job_queue(status, priority DESC) WHERE status = 'pending';

-- 通知相关索引已移除（使用 WebSocket 实时通知）

-- WebSocket连接表索引
CREATE INDEX idx_websocket_connections_connection_id ON websocket_connections(connection_id);
CREATE INDEX idx_websocket_connections_user_id ON websocket_connections(user_id);
CREATE INDEX idx_websocket_connections_status ON websocket_connections(status);
CREATE INDEX idx_websocket_connections_last_activity ON websocket_connections(last_activity_at);
CREATE INDEX idx_websocket_connections_rooms ON websocket_connections USING GIN(subscribed_rooms);

-- API请求日志表索引
CREATE INDEX idx_api_request_logs_request_id ON api_request_logs(request_id);
CREATE INDEX idx_api_request_logs_user_id ON api_request_logs(user_id);
CREATE INDEX idx_api_request_logs_path ON api_request_logs(path);
CREATE INDEX idx_api_request_logs_status_code ON api_request_logs(status_code);
CREATE INDEX idx_api_request_logs_created_at ON api_request_logs(created_at);
CREATE INDEX idx_api_request_logs_response_time ON api_request_logs(response_time_ms);

-- 系统指标表索引
CREATE INDEX idx_system_metrics_name ON system_metrics(metric_name);
CREATE INDEX idx_system_metrics_timestamp ON system_metrics(timestamp);
CREATE INDEX idx_system_metrics_name_timestamp ON system_metrics(metric_name, timestamp);
CREATE INDEX idx_system_metrics_labels ON system_metrics USING GIN(labels);

-- ============================================================================
-- 复合索引优化（基于查询模式）
-- ============================================================================

-- 用户活跃度复合索引
CREATE INDEX idx_users_active_recent ON users(is_active, last_login_at) WHERE is_active = true;

-- 群组活跃度复合索引
CREATE INDEX idx_groups_active_recent ON groups(is_active, updated_at) WHERE is_active = true;

-- 消息时间线复合索引
CREATE INDEX idx_messages_conversation_time ON messages(conversation_id, created_at DESC) WHERE is_deleted = false;

-- 瞬间时间线复合索引
CREATE INDEX idx_moments_user_timeline ON moments(user_id, created_at DESC) WHERE is_deleted = false;
CREATE INDEX idx_moments_public_timeline ON moments(visibility, created_at DESC) WHERE visibility = 'public' AND is_deleted = false;

-- 地理位置查询优化索引
CREATE INDEX idx_user_locations_current_spatial ON user_locations(location) WHERE is_current = true;
CREATE INDEX idx_group_locations_primary_spatial ON group_locations(location) WHERE is_primary = true;
CREATE INDEX idx_moments_location_spatial ON moments(location) WHERE location IS NOT NULL AND is_deleted = false;

-- 好友关系查询优化索引
CREATE INDEX idx_friendships_active ON friendships(user_id, status) WHERE status = 'accepted';
CREATE INDEX idx_friendships_mutual ON friendships(user_id, friend_id, status) WHERE status = 'accepted';

-- 通知查询优化索引已移除（使用 WebSocket 实时通知）

-- ============================================================================
-- 触发器创建
-- ============================================================================

-- 系统配置表更新时间戳触发器
CREATE TRIGGER system_configs_updated_at_trigger
    BEFORE UPDATE ON system_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 任务队列表更新时间戳触发器
CREATE TRIGGER job_queue_updated_at_trigger
    BEFORE UPDATE ON job_queue
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 系统维护函数
-- ============================================================================

-- 清理过期缓存函数
CREATE OR REPLACE FUNCTION cleanup_expired_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM cache_entries 
    WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- 注意：通知清理函数已移除，通知系统使用 WebSocket 实时推送，无需数据库清理

-- 清理旧的API日志函数
CREATE OR REPLACE FUNCTION cleanup_old_api_logs(retention_days INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM api_request_logs 
    WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '1 day' * retention_days;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- 清理旧的系统指标函数
CREATE OR REPLACE FUNCTION cleanup_old_metrics(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM system_metrics 
    WHERE timestamp < CURRENT_TIMESTAMP - INTERVAL '1 day' * retention_days;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 初始系统配置
-- ============================================================================

-- 插入默认系统配置
INSERT INTO system_configs (config_key, config_value, config_type, description, category, is_public) VALUES
('app.name', 'HotSpot', 'string', '应用程序名称', 'app', true),
('app.version', '2.0.0', 'string', '应用程序版本', 'app', true),
('app.environment', 'development', 'string', '运行环境', 'app', false),
('user.max_friends', '1000', 'number', '用户最大好友数量', 'user', true),
('group.max_members', '500', 'number', '群组最大成员数量', 'group', true),
('message.retention_days', '365', 'number', '消息保留天数', 'message', false),
('cache.default_ttl', '3600', 'number', '默认缓存TTL（秒）', 'cache', false)
-- 注意：通知配置已移除（使用 WebSocket 实时通知）
ON CONFLICT (config_key) DO NOTHING;

-- 事件存储表（用于事件溯源）
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT UNIQUE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    aggregate_id TEXT NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    correlation_id TEXT,
    causation_id TEXT,
    user_id TEXT,
    session_id TEXT,
    occurred_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 事件存储表索引
CREATE INDEX idx_events_aggregate ON events(aggregate_id, aggregate_type);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_occurred_at ON events(occurred_at);
CREATE INDEX idx_events_correlation_id ON events(correlation_id);
CREATE INDEX idx_events_user_id ON events(user_id);

-- 事件处理进度表（用于事件处理器状态跟踪）
CREATE TABLE event_processing_progress (
    id BIGSERIAL PRIMARY KEY,
    handler_name VARCHAR(100) UNIQUE NOT NULL,
    last_processed_event_id BIGINT NOT NULL,
    last_processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 事件处理进度表索引
CREATE INDEX idx_event_processing_progress_handler ON event_processing_progress(handler_name);
CREATE INDEX idx_event_processing_progress_updated_at ON event_processing_progress(updated_at);

-- 事件快照表（用于事件溯源性能优化）
CREATE TABLE event_snapshots (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id TEXT NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    version INTEGER NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 确保每个聚合只有一个快照
    UNIQUE(aggregate_id, aggregate_type)
);

-- 事件快照表索引
CREATE INDEX idx_event_snapshots_aggregate ON event_snapshots(aggregate_id, aggregate_type);
CREATE INDEX idx_event_snapshots_created_at ON event_snapshots(created_at);
