-- 快速修复数据库架构
-- 添加缺失的列和表

-- 1. 添加groups表缺失的avatar_url列
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'groups' AND column_name = 'avatar_url'
    ) THEN
        ALTER TABLE groups ADD COLUMN avatar_url TEXT;
    END IF;
END $$;

-- 2. 创建messages表（如果不存在）
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL,
    sender_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_messages_sender_id 
        FOREIGN KEY (sender_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- 3. 创建group_locations表（如果不存在）
CREATE TABLE IF NOT EXISTS group_locations (
    id BIGSERIAL PRIMARY KEY,
    group_id UUID NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_name TEXT NOT NULL DEFAULT '',
    radius DOUBLE PRECISION NOT NULL DEFAULT 1000.0,
    geom GEOGRAPHY(POINT, 4326),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_group_locations_group_id 
        FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE
);

-- 4. 创建events表（如果不存在）
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_version INTEGER NOT NULL DEFAULT 1,
    event_data JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. 创建event_snapshots表（如果不存在）
CREATE TABLE IF NOT EXISTS event_snapshots (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    snapshot_version INTEGER NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. 创建event_processing_progress表（如果不存在）
CREATE TABLE IF NOT EXISTS event_processing_progress (
    id BIGSERIAL PRIMARY KEY,
    handler_name VARCHAR(100) NOT NULL UNIQUE,
    last_processed_event_id BIGINT NOT NULL DEFAULT 0,
    last_processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_errors INTEGER NOT NULL DEFAULT 0
);

-- 7. 创建索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

CREATE INDEX IF NOT EXISTS idx_group_locations_group_id ON group_locations(group_id);
CREATE INDEX IF NOT EXISTS idx_group_locations_geom ON group_locations USING GIST(geom);

CREATE INDEX IF NOT EXISTS idx_events_aggregate_id ON events(aggregate_id);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);

-- 8. 更新地理位置数据（如果geom列为空）
UPDATE group_locations 
SET geom = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography
WHERE geom IS NULL;


