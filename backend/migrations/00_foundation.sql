-- ============================================================================
-- HotSpot 基础设施迁移 v2.0
-- ============================================================================
-- 
-- 功能职责：
-- - 启用必要的 PostgreSQL 扩展
-- - 创建通用的数据库函数和触发器
-- - 建立地理位置处理的基础设施
-- - 提供其他模块依赖的公共功能
--
-- 依赖关系：
-- - 无依赖，必须首先执行
-- - 为所有其他迁移文件提供基础功能
--
-- ============================================================================

-- 启用PostGIS扩展（地理位置功能）
CREATE EXTENSION IF NOT EXISTS postgis CASCADE;

-- 启用UUID生成扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" CASCADE;

-- 启用全文搜索扩展
CREATE EXTENSION IF NOT EXISTS pg_trgm CASCADE;

-- 启用加密扩展
CREATE EXTENSION IF NOT EXISTS pgcrypto CASCADE;

-- ============================================================================
-- 通用枚举类型定义
-- ============================================================================

-- 用户类型枚举
CREATE TYPE user_type_enum AS ENUM ('regular', 'temporary', 'admin');

-- 群组类型枚举
CREATE TYPE group_type_enum AS ENUM ('public', 'private', 'temporary');

-- 群组可见性枚举
CREATE TYPE group_visibility_enum AS ENUM ('public', 'members_only', 'hidden');

-- 群组角色枚举
CREATE TYPE group_role_enum AS ENUM ('owner', 'admin', 'member');

-- 消息类型枚举
CREATE TYPE message_type_enum AS ENUM ('text', 'image', 'file', 'location', 'system');

-- 注意：通知类型枚举已移除，通知系统使用 WebSocket 实时推送

-- ============================================================================
-- 通用函数定义
-- ============================================================================

-- 生成唯一ID函数
CREATE OR REPLACE FUNCTION generate_unique_id()
RETURNS TEXT AS $$
BEGIN
    RETURN uuid_generate_v4()::TEXT;
END;
$$ LANGUAGE plpgsql;

-- 更新时间戳函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 地理距离计算函数（米）
CREATE OR REPLACE FUNCTION calculate_distance_meters(
    lat1 DOUBLE PRECISION,
    lon1 DOUBLE PRECISION,
    lat2 DOUBLE PRECISION,
    lon2 DOUBLE PRECISION
)
RETURNS DOUBLE PRECISION AS $$
BEGIN
    RETURN ST_Distance(
        ST_Point(lon1, lat1)::geography,
        ST_Point(lon2, lat2)::geography
    );
END;
$$ LANGUAGE plpgsql;

-- 地理距离计算函数（公里）
CREATE OR REPLACE FUNCTION calculate_distance_km(
    lat1 DOUBLE PRECISION,
    lon1 DOUBLE PRECISION,
    lat2 DOUBLE PRECISION,
    lon2 DOUBLE PRECISION
)
RETURNS DOUBLE PRECISION AS $$
BEGIN
    RETURN calculate_distance_meters(lat1, lon1, lat2, lon2) / 1000.0;
END;
$$ LANGUAGE plpgsql;

-- 验证地理坐标函数
CREATE OR REPLACE FUNCTION validate_coordinates(
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION
)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN latitude >= -90 AND latitude <= 90 
       AND longitude >= -180 AND longitude <= 180;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 审计和日志基础设施
-- ============================================================================

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    table_name VARCHAR(100) NOT NULL,
    operation VARCHAR(10) NOT NULL, -- INSERT, UPDATE, DELETE
    old_data JSONB,
    new_data JSONB,
    user_id BIGINT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT
);

-- 创建审计日志索引
CREATE INDEX IF NOT EXISTS idx_audit_logs_table_name ON audit_logs(table_name);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);

-- 通用审计触发器函数
CREATE OR REPLACE FUNCTION audit_trigger_function()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO audit_logs (table_name, operation, old_data)
        VALUES (TG_TABLE_NAME, TG_OP, row_to_json(OLD));
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_logs (table_name, operation, old_data, new_data)
        VALUES (TG_TABLE_NAME, TG_OP, row_to_json(OLD), row_to_json(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO audit_logs (table_name, operation, new_data)
        VALUES (TG_TABLE_NAME, TG_OP, row_to_json(NEW));
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 性能监控基础设施
-- ============================================================================

-- 查询性能统计表
CREATE TABLE IF NOT EXISTS query_performance_stats (
    id BIGSERIAL PRIMARY KEY,
    query_hash TEXT NOT NULL,
    query_text TEXT,
    execution_count BIGINT DEFAULT 1,
    total_time_ms DOUBLE PRECISION DEFAULT 0,
    avg_time_ms DOUBLE PRECISION DEFAULT 0,
    min_time_ms DOUBLE PRECISION DEFAULT 0,
    max_time_ms DOUBLE PRECISION DEFAULT 0,
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建查询性能统计索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_query_performance_hash ON query_performance_stats(query_hash);
CREATE INDEX IF NOT EXISTS idx_query_performance_last_seen ON query_performance_stats(last_seen);

-- ============================================================================
-- 基础设施完成标记
-- ============================================================================

-- 插入基础设施版本记录
INSERT INTO audit_logs (table_name, operation, new_data) 
VALUES ('foundation', 'INIT', '{"version": "2.0", "description": "基础设施初始化完成"}')
ON CONFLICT DO NOTHING;
