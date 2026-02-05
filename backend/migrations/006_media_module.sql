-- ============================================================
-- 006_media_module.sql
-- 媒体模块：文件、上传、URL刷新
-- ============================================================

-- 1. 上传会话表 (分区表)
CREATE TABLE IF NOT EXISTS upload_sessions_partitioned (
    upload_id UUID NOT NULL DEFAULT uuidv7(),
    user_id BIGINT NOT NULL,
    
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    category media_category NOT NULL,
    
    upload_url TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    
    status upload_status NOT NULL DEFAULT 'pending',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    
    PRIMARY KEY (upload_id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建默认分区
CREATE TABLE IF NOT EXISTS upload_sessions_default PARTITION OF upload_sessions_partitioned DEFAULT;

-- 创建初始分区 (2025, 2026)
DO $$
BEGIN
    PERFORM create_partition('upload_sessions_2025', 'upload_sessions_partitioned', '2025-01-01', '2026-01-01');
    PERFORM create_partition('upload_sessions_2026', 'upload_sessions_partitioned', '2026-01-01', '2027-01-01');
END $$;

-- 重命名
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'upload_sessions') THEN
        ALTER TABLE upload_sessions_partitioned RENAME TO upload_sessions;
    END IF;
END $$;

-- 2. 媒体文件表 (无外键)
CREATE TABLE IF NOT EXISTS media_files (
    file_id UUID PRIMARY KEY DEFAULT uuidv7(),
    uploader_id BIGINT NOT NULL,
    
    url TEXT NOT NULL,
    thumbnail_url TEXT,
    storage_path TEXT NOT NULL,
    
    meta JSONB, -- { width, height, duration, size, mime }
    hash_md5 VARCHAR(32),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ -- 用于预签名URL过期时间
);

CREATE INDEX IF NOT EXISTS idx_media_files_uploader ON media_files(uploader_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_files_hash ON media_files(hash_md5);

-- 3. 媒体URL刷新队列 (无外键)
CREATE TABLE IF NOT EXISTS media_url_refresh_queue (
    queue_id UUID PRIMARY KEY DEFAULT uuidv7(),
    moment_id UUID,
    message_id UUID,
    file_id UUID NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_refreshed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_media_refresh_queue_created ON media_url_refresh_queue(created_at) WHERE processed = FALSE;
CREATE INDEX IF NOT EXISTS idx_media_refresh_queue_moment ON media_url_refresh_queue(moment_id) WHERE moment_id IS NOT NULL;

-- 4. URL刷新函数
CREATE OR REPLACE FUNCTION refresh_presigned_url(
    p_media_url TEXT,
    p_file_id UUID,
    p_expires_at TIMESTAMPTZ
) RETURNS TEXT AS $$
DECLARE
    v_url TEXT;
    v_expires_at TIMESTAMPTZ;
BEGIN
    -- 获取当前文件信息
    SELECT url, expires_at INTO v_url, v_expires_at
    FROM media_files
    WHERE file_id = p_file_id;
    
    -- 如果 URL 未过期且已生成，则不需要刷新
    IF v_expires_at > NOW() AND v_url IS NOT NULL THEN
        -- 返回现有 URL
        RETURN v_url;
    END IF;
    
    -- 否则生成新的预签名 URL (需要后端服务处理)
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
