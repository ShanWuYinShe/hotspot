CREATE TABLE IF NOT EXISTS media_upload_sessions (
    upload_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    file_name TEXT NOT NULL,
    declared_file_size INTEGER NOT NULL,
    declared_mime_type TEXT NOT NULL,
    category TEXT NOT NULL,
    md5 TEXT NULL,
    status TEXT NOT NULL,
    file_id TEXT NULL,
    final_file_size INTEGER NULL,
    final_mime_type TEXT NULL,
    thumbnail_url TEXT NULL,
    duration INTEGER NULL,
    width INTEGER NULL,
    height INTEGER NULL,
    completed_idempotency_key TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NULL
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_upload_sessions_user_created_at
    ON media_upload_sessions (user_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_upload_sessions_storage_key
    ON media_upload_sessions (storage_key);
