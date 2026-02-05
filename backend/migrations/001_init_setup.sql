-- ============================================================
-- 001_init_setup.sql
-- 基础设置：扩展、通用函数、类型定义
-- ============================================================

-- 1. 启用必要的扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- 用于模糊搜索优化

-- 2. 创建自动更新 updated_at 的函数
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 3. 通用类型定义
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_gender') THEN
        CREATE TYPE user_gender AS ENUM ('male', 'female', 'other', 'unknown');
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
        CREATE TYPE user_status AS ENUM ('active', 'inactive', 'banned', 'deleted');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'reaction_type') THEN
        CREATE TYPE reaction_type AS ENUM ('like', 'love', 'haha', 'wow', 'sad', 'angry');
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'upload_status') THEN
        CREATE TYPE upload_status AS ENUM ('pending', 'processing', 'completed', 'failed');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'media_category') THEN
        CREATE TYPE media_category AS ENUM ('image', 'video', 'audio', 'document', 'other');
    END IF;
END $$;

-- 4. 分区管理表
CREATE TABLE IF NOT EXISTS partition_templates (
    table_name TEXT NOT NULL,
    partition_prefix TEXT NOT NULL,
    retention_days INTEGER NOT NULL DEFAULT 90,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. 分区创建辅助函数
CREATE OR REPLACE FUNCTION create_partition(
    table_name TEXT,
    partition_name TEXT,
    start_date DATE,
    end_date DATE
) RETURNS VOID AS $$
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF %s FOR VALUES FROM (%L) TO (%L)',
        table_name, partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- 6. 迁移记录表初始化
-- 注意：SQLx 会自动创建和管理 _sqlx_migrations 表，无需手动创建 schema_migrations 表
