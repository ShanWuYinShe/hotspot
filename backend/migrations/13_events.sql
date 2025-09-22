-- ============================================================================
-- 13. 事件存储系统表
-- ============================================================================
-- 
-- 功能职责：
-- - 实现事件溯源(Event Sourcing)模式
-- - 存储领域事件用于审计和重放
-- - 支持事件流处理和状态重建
-- - 提供事件查询和分析功能
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql
--
-- ============================================================================

-- 事件存储表
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,                   -- 事件ID（自增主键，用于排序）
    event_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 事件唯一标识符
    aggregate_type TEXT NOT NULL,              -- 聚合类型：'user'|'group'|'message'|'moment'
    aggregate_id TEXT NOT NULL,                -- 聚合根ID
    event_type TEXT NOT NULL,                  -- 事件类型
    version INTEGER NOT NULL DEFAULT 1,        -- 事件版本号
    event_data JSONB NOT NULL DEFAULT '{}',   -- 事件数据（JSON格式）
    metadata JSONB NOT NULL DEFAULT '{}',     -- 事件元数据
    -- 时间戳
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 事件发生时间
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 事件记录时间
    -- 用户信息
    user_id BIGINT,                            -- 触发事件的用户ID
    session_id TEXT,                           -- 会话ID
    correlation_id TEXT,                       -- 关联ID（用于跟踪相关事件）
    causation_id TEXT,                         -- 因果ID（触发此事件的事件ID）
    -- 状态
    is_processed BOOLEAN NOT NULL DEFAULT false, -- 是否已处理
    processed_at TIMESTAMP WITH TIME ZONE,     -- 处理时间
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- 事件快照表（用于性能优化）
CREATE TABLE event_snapshots (
    id BIGSERIAL PRIMARY KEY,                   -- 快照ID
    snapshot_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 快照唯一标识符
    aggregate_type TEXT NOT NULL,              -- 聚合类型
    aggregate_id TEXT NOT NULL,                -- 聚合根ID
    version INTEGER NOT NULL,                  -- 聚合版本号
    snapshot_data JSONB NOT NULL,             -- 快照数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    
    UNIQUE(aggregate_type, aggregate_id, version)
);

-- 事件处理进度表
CREATE TABLE event_processing_progress (
    id BIGSERIAL PRIMARY KEY,                   -- 状态ID
    handler_name TEXT NOT NULL,               -- 处理器名称
    last_processed_event_id BIGINT,           -- 最后处理的事件ID
    last_processed_at TIMESTAMP WITH TIME ZONE, -- 最后处理时间
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否活跃
    error_count INTEGER NOT NULL DEFAULT 0,    -- 错误次数
    last_error TEXT,                           -- 最后的错误信息
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    
    UNIQUE(handler_name),
    FOREIGN KEY (last_processed_event_id) REFERENCES events(id) ON DELETE SET NULL
);

-- ============================================================================
-- 索引创建
-- ============================================================================

-- 事件存储索引
CREATE INDEX idx_events_aggregate ON events(aggregate_type, aggregate_id);
CREATE INDEX idx_events_aggregate_id ON events(aggregate_id);
CREATE INDEX idx_events_event_type ON events(event_type);
CREATE INDEX idx_events_occurred_at ON events(occurred_at);
CREATE INDEX idx_events_user_id ON events(user_id);
CREATE INDEX idx_events_correlation_id ON events(correlation_id);
CREATE INDEX idx_events_is_processed ON events(is_processed);
CREATE INDEX idx_events_aggregate_occurred ON events(aggregate_type, aggregate_id, occurred_at);

-- 事件快照索引
CREATE INDEX idx_snapshots_aggregate ON event_snapshots(aggregate_type, aggregate_id);
CREATE INDEX idx_snapshots_version ON event_snapshots(version DESC);

-- 事件处理进度索引
CREATE INDEX idx_processing_progress_handler ON event_processing_progress(handler_name);
CREATE INDEX idx_processing_progress_active ON event_processing_progress(is_active);

-- ============================================================================
-- 触发器创建
-- ============================================================================

-- 自动更新事件处理状态的触发器
CREATE OR REPLACE FUNCTION update_event_processing_status_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- 当事件被标记为已处理时，更新处理状态
    IF NEW.is_processed = true AND OLD.is_processed = false THEN
        UPDATE event_processing_progress 
        SET last_processed_event_id = NEW.id,
            last_processed_at = NEW.processed_at,
            updated_at = NOW()
        WHERE handler_name = 'default_processor';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建触发器
CREATE TRIGGER trigger_update_event_processing_status
    AFTER UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_event_processing_status_trigger();

-- ============================================================================
-- 分区表创建（可选，用于大数据量场景）
-- ============================================================================

-- 按月分区事件表（可选）
-- CREATE TABLE events_y2024m01 PARTITION OF events
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- ============================================================================
-- 初始数据插入
-- ============================================================================

-- 插入默认的事件处理器状态
INSERT INTO event_processing_progress (handler_name, is_active) VALUES
    ('default_processor', true),
    ('notification_processor', true),
    ('analytics_processor', true),
    ('audit_processor', true);

-- ============================================================================
-- 表注释
-- ============================================================================

COMMENT ON TABLE events IS '事件存储表，实现事件溯源模式，记录所有领域事件';
COMMENT ON TABLE event_snapshots IS '事件快照表，用于性能优化，存储聚合根的快照状态';
COMMENT ON TABLE event_processing_progress IS '事件处理进度表，跟踪各个事件处理器的处理进度';

-- 列注释
COMMENT ON COLUMN events.event_id IS '事件唯一标识符，用于外部引用和去重';
COMMENT ON COLUMN events.aggregate_type IS '聚合类型，标识事件所属的聚合根类型';
COMMENT ON COLUMN events.aggregate_id IS '聚合根ID，标识具体的聚合根实例';
COMMENT ON COLUMN events.event_type IS '事件类型，描述具体的业务事件';
COMMENT ON COLUMN events.event_data IS '事件数据，包含事件的具体信息';
COMMENT ON COLUMN events.metadata IS '事件元数据，包含版本、时间戳等辅助信息';
COMMENT ON COLUMN events.correlation_id IS '关联ID，用于跟踪相关的事件序列';
COMMENT ON COLUMN events.causation_id IS '因果ID，标识触发此事件的原因事件';

COMMENT ON COLUMN event_snapshots.version IS '聚合版本号，对应快照时的事件序号';
COMMENT ON COLUMN event_snapshots.snapshot_data IS '快照数据，包含聚合根的完整状态';

COMMENT ON COLUMN event_processing_progress.handler_name IS '处理器名称，唯一标识一个事件处理器';
COMMENT ON COLUMN event_processing_progress.last_processed_event_id IS '最后处理的事件ID，用于断点续传';
COMMENT ON COLUMN event_processing_progress.error_count IS '错误次数，用于监控处理器健康状态';
