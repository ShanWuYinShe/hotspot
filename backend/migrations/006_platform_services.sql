-- ============================================================
-- 006_platform_services.sql
-- 平台服务：通知、举报、分享、媒体上传
-- ============================================================

-- ─── 通知表（分区）──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS notifications_partitioned (
    notification_id UUID                NOT NULL DEFAULT uuidv7(),
    user_id         BIGINT              NOT NULL,   -- 接收者
    notification_type VARCHAR(50)       NOT NULL,   -- moment_like, new_follower, group_invite, ...
    category        notification_category NOT NULL,
    -- 聚合支持
    aggregation_key VARCHAR(255),                   -- 聚合键（如 moment:{id}:like）
    aggregated_count INTEGER            NOT NULL DEFAULT 1,
    -- 内容
    title           VARCHAR(255)        NOT NULL,
    body            TEXT,
    actor_user_id   BIGINT,                         -- 触发者
    target_type     VARCHAR(20),                    -- moment, comment, user, group, url
    target_id       VARCHAR(100),
    target_url      TEXT,
    target_preview_text TEXT,
    target_preview_image_url TEXT,
    payload         JSONB,                          -- 额外数据
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    PRIMARY KEY (notification_id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS notifications_default PARTITION OF notifications_partitioned DEFAULT;

-- 未读通知（部分索引）
CREATE INDEX idx_notif_user_time   ON notifications_partitioned(user_id, created_at DESC);
CREATE INDEX idx_notif_user_type   ON notifications_partitioned(user_id, notification_type, created_at DESC);

-- ─── 通知偏好（简化版：仅分类级开关）──────────────────────
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id             BIGINT      PRIMARY KEY,
    -- 分类总开关
    social_enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    group_enabled       BOOLEAN     NOT NULL DEFAULT TRUE,
    system_enabled      BOOLEAN     NOT NULL DEFAULT TRUE,   -- 不可关闭，始终 TRUE
    -- 免打扰
    quiet_hours_enabled BOOLEAN     NOT NULL DEFAULT FALSE,
    quiet_hours_start   VARCHAR(5),                          -- HH:mm
    quiet_hours_end     VARCHAR(5),
    quiet_hours_timezone VARCHAR(50),                        -- IANA 时区
    -- 推送设置
    push_enabled        BOOLEAN     NOT NULL DEFAULT TRUE,
    push_sound          BOOLEAN     NOT NULL DEFAULT TRUE,
    push_vibration      BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER tr_notif_prefs_updated_at
    BEFORE UPDATE ON notification_preferences FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 通知订阅总开关 ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS notification_subscriptions (
    user_id                     BIGINT      PRIMARY KEY,
    new_follower_enabled        BOOLEAN     NOT NULL DEFAULT TRUE,
    mutual_follow_enabled       BOOLEAN     NOT NULL DEFAULT TRUE,
    follow_request_enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    follow_request_handled_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    moment_like_enabled         BOOLEAN     NOT NULL DEFAULT TRUE,
    moment_comment_enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    comment_reply_enabled       BOOLEAN     NOT NULL DEFAULT TRUE,
    group_invite_enabled        BOOLEAN     NOT NULL DEFAULT TRUE,
    group_invite_response_enabled BOOLEAN   NOT NULL DEFAULT TRUE,
    group_apply_enabled         BOOLEAN     NOT NULL DEFAULT TRUE,
    group_apply_result_enabled  BOOLEAN     NOT NULL DEFAULT TRUE,
    group_announcement_enabled  BOOLEAN     NOT NULL DEFAULT TRUE,
    message_mention_enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    system_announcement_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER tr_notification_subscriptions_updated_at
    BEFORE UPDATE ON notification_subscriptions FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 通知已读状态（全局水位线）────────────────────────────
CREATE TABLE IF NOT EXISTS notification_read_state (
    user_id         BIGINT      PRIMARY KEY,
    last_read_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── 通知逐条已读 ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS notification_read_items (
    user_id         BIGINT      NOT NULL,
    notification_id UUID        NOT NULL,
    read_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, notification_id)
);
CREATE INDEX idx_notif_read_items_user ON notification_read_items(user_id, read_at DESC);

-- ─── 举报 ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS reports (
    report_id       UUID                PRIMARY KEY DEFAULT uuidv7(),
    reporter_id     BIGINT              NOT NULL,
    target_type     report_target_type  NOT NULL,
    target_id       VARCHAR(100)        NOT NULL,
    reason          VARCHAR(100)        NOT NULL,
    description     TEXT,
    status          report_status       NOT NULL DEFAULT 'pending',
    -- 管理员处理
    handled_by      BIGINT,
    handled_action  report_handle_action,
    handled_note    TEXT,
    handled_at      TIMESTAMPTZ,
    -- 目标快照（处理时存证）
    target_snapshot JSONB,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reports_status   ON reports(status, created_at);
CREATE INDEX idx_reports_reporter ON reports(reporter_id, created_at DESC);
CREATE INDEX idx_reports_target   ON reports(target_type, target_id);

CREATE TRIGGER tr_reports_updated_at
    BEFORE UPDATE ON reports FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 分享 ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS shares (
    share_id        UUID        PRIMARY KEY DEFAULT uuidv7(),
    sharer_id       BIGINT      NOT NULL,
    share_type      share_type  NOT NULL,
    share_code      VARCHAR(16) NOT NULL UNIQUE,
    -- 分享内容（按 share_type 对应不同字段有值）
    content_id      VARCHAR(100),                   -- user_id / group_id / moment_id
    payload         JSONB,                          -- reference_point 坐标等额外数据
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_shares_code    ON shares(share_code);
CREATE INDEX idx_shares_creator ON shares(sharer_id, created_at DESC);
CREATE INDEX idx_shares_expires ON shares(expires_at) WHERE expires_at IS NOT NULL;

-- 唯一分享码生成（依赖 shares 表和 001 中的 generate_share_code）
CREATE OR REPLACE FUNCTION generate_share_code_unique(p_length INTEGER DEFAULT 8)
RETURNS TEXT AS $$
DECLARE
    new_code TEXT;
BEGIN
    LOOP
        new_code := generate_share_code(p_length);
        EXIT WHEN NOT EXISTS (SELECT 1 FROM shares WHERE share_code = new_code);
    END LOOP;
    RETURN new_code;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- ─── 媒体文件 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS media_files (
    file_id         UUID        PRIMARY KEY DEFAULT uuidv7(),
    uploader_id     BIGINT      NOT NULL,
    category        media_category,
    url             TEXT        NOT NULL,
    thumbnail_url   TEXT,
    storage_path    TEXT        NOT NULL,
    meta            JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- 文件元数据
    file_name       VARCHAR(255),
    file_size       BIGINT,
    mime_type       VARCHAR(100),
    width           INTEGER,
    height          INTEGER,
    duration        INTEGER,                        -- 音视频时长（秒）
    hash_md5        VARCHAR(32),                    -- 秒传/去重
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ                     -- 临时文件
);
CREATE INDEX idx_media_uploader ON media_files(uploader_id, created_at DESC);
CREATE INDEX idx_media_hash     ON media_files(hash_md5) WHERE hash_md5 IS NOT NULL;

-- ─── 上传会话（Presigned URL）───────────────────────────────
CREATE TABLE IF NOT EXISTS upload_sessions (
    upload_id       UUID            PRIMARY KEY DEFAULT uuidv7(),
    user_id         BIGINT          NOT NULL,
    file_name       VARCHAR(255)    NOT NULL,
    file_size       BIGINT          NOT NULL,
    mime_type       VARCHAR(100)    NOT NULL,
    category        media_category  NOT NULL,
    upload_url      TEXT            NOT NULL,
    storage_path    TEXT            NOT NULL,
    status          upload_status   NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ     NOT NULL
);
