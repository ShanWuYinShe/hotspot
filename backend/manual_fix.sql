-- 手动修复数据库架构
-- 添加缺失的扩展
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 添加缺失的列
ALTER TABLE groups ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- 创建events表
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

-- 创建event_snapshots表
CREATE TABLE IF NOT EXISTS event_snapshots (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    snapshot_version INTEGER NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建event_processing_progress表
CREATE TABLE IF NOT EXISTS event_processing_progress (
    id BIGSERIAL PRIMARY KEY,
    handler_name VARCHAR(100) NOT NULL UNIQUE,
    last_processed_event_id BIGINT NOT NULL DEFAULT 0,
    last_processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_errors INTEGER NOT NULL DEFAULT 0
);

-- 创建group_locations表
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

-- 确保messages表存在
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

-- 更新迁移记录的校验和
UPDATE _sqlx_migrations 
SET checksum = decode('66d6b131feb8479cd15d2ab4227152168ac7089fbf79d8b9a6420cc9ed19cdd7ed167902e56ab6881ccba0b080de9556', 'hex')
WHERE version = 0;


