-- ============================================================
-- 002_user_module.sql
-- 用户模块：核心用户、认证、关系、设备
-- ============================================================

-- ─── 用户表 ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    user_id             BIGINT          PRIMARY KEY,  -- 应用层生成（Snowflake）
    username            VARCHAR(50)     NOT NULL UNIQUE,       -- 登录 ID
    email               VARCHAR(255)    UNIQUE,
    phone               VARCHAR(20)     UNIQUE,
    password_hash       VARCHAR(255),
    nickname            VARCHAR(50),
    avatar_url          TEXT,
    bio                 TEXT,
    gender              user_gender     NOT NULL DEFAULT 'male',
    birthday            DATE,
    user_type           user_type       NOT NULL DEFAULT 'normal',
    status              user_status     NOT NULL DEFAULT 'normal',
    -- PostGIS 位置
    location            GEOGRAPHY(POINT, 4326),
    location_name       VARCHAR(255),
    location_updated_at TIMESTAMPTZ,
    region_id           BIGINT,
    -- 注册 & 登录
    register_ip         INET,
    register_location   VARCHAR(255),
    last_login_ip       INET,
    last_login_location VARCHAR(255),
    last_login_at       TIMESTAMPTZ,
    -- 时间戳
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
) WITH (fillfactor = 90);

-- 索引
CREATE INDEX idx_users_username_trgm ON users USING GIN(username gin_trgm_ops);
CREATE INDEX idx_users_nickname_trgm ON users USING GIN(nickname gin_trgm_ops);
CREATE INDEX idx_users_email         ON users(email)    WHERE email IS NOT NULL;
CREATE INDEX idx_users_phone         ON users(phone)    WHERE phone IS NOT NULL;
CREATE INDEX idx_users_location      ON users USING GIST(location);
CREATE INDEX idx_users_created_at    ON users(created_at);

CREATE TRIGGER tr_users_updated_at
    BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 用户设置 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_settings (
    user_id             BIGINT      PRIMARY KEY,
    language            VARCHAR(10) NOT NULL DEFAULT 'zh-CN',
    theme               VARCHAR(20) NOT NULL DEFAULT 'system',
    notifications_enabled BOOLEAN   NOT NULL DEFAULT TRUE,
    privacy_settings    JSONB       NOT NULL DEFAULT '{}'::jsonb,
    preferences         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- 隐私控制（legacy 字段，兼容旧版本）
    privacy_level       privacy_level NOT NULL DEFAULT 'public',
    location_visible    BOOLEAN     NOT NULL DEFAULT TRUE,
    presence_visible    BOOLEAN     NOT NULL DEFAULT TRUE,
    friends_visible     privacy_level NOT NULL DEFAULT 'public',
    following_visible   privacy_level NOT NULL DEFAULT 'public',
    followers_visible   privacy_level NOT NULL DEFAULT 'public',
    groups_visible      privacy_level NOT NULL DEFAULT 'public',
    liked_moments_visible privacy_level NOT NULL DEFAULT 'public',
    stats_visible       privacy_level NOT NULL DEFAULT 'public',
    -- 互动控制
    allow_follow        BOOLEAN     NOT NULL DEFAULT TRUE,
    allow_private_chat  VARCHAR(20) NOT NULL DEFAULT 'everyone'
        CHECK (allow_private_chat IN ('everyone', 'friends', 'nobody')),
    -- 搜索发现
    searchable          BOOLEAN     NOT NULL DEFAULT TRUE,
    phone_searchable    BOOLEAN     NOT NULL DEFAULT FALSE,
    nickname_searchable BOOLEAN     NOT NULL DEFAULT TRUE,
    location_searchable BOOLEAN     NOT NULL DEFAULT TRUE,
    -- 时间戳
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER tr_user_settings_updated_at
    BEFORE UPDATE ON user_settings FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 在线状态（高频更新，低 fillfactor 启用 HOT）────────────
CREATE TABLE IF NOT EXISTS user_online_status (
    user_id         BIGINT      PRIMARY KEY,
    is_online       BOOLEAN     NOT NULL DEFAULT FALSE,
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    connection_count INTEGER    NOT NULL DEFAULT 0,
    client_ip       INET,
    device_info     JSONB
) WITH (fillfactor = 70);

-- ─── 用户统计 Bucket（分片减少行锁争用）────────────────────
CREATE TABLE IF NOT EXISTS user_stats_buckets (
    user_id         BIGINT      NOT NULL,
    bucket_id       SMALLINT    NOT NULL,
    moments_count   INTEGER     NOT NULL DEFAULT 0,
    friends_count   INTEGER     NOT NULL DEFAULT 0,
    following_count INTEGER     NOT NULL DEFAULT 0,
    followers_count INTEGER     NOT NULL DEFAULT 0,
    groups_count    INTEGER     NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, bucket_id)
);
CREATE INDEX idx_user_stats_user ON user_stats_buckets(user_id);

-- ─── 关注关系 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS follows (
    follower_id     BIGINT      NOT NULL,
    following_id    BIGINT      NOT NULL,
    source          follow_source,
    source_moment_id UUID,                          -- source='moment' 时记录来源瞬间
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id)
);
-- 覆盖索引：查我的关注列表
CREATE INDEX idx_follows_follower ON follows(follower_id, created_at DESC) INCLUDE (following_id);
-- 覆盖索引：查我的粉丝列表
CREATE INDEX idx_follows_following ON follows(following_id, created_at DESC) INCLUDE (follower_id);

-- ─── 拉黑关系 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS blocks (
    blocker_id  BIGINT      NOT NULL,
    blocked_id  BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id)
);
CREATE INDEX idx_blocks_blocked ON blocks(blocked_id, created_at DESC);

-- ─── 用户标签关联 ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_tags (
    user_id     BIGINT      NOT NULL,
    tag_id      UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, tag_id)
);
CREATE INDEX idx_user_tags_tag ON user_tags(tag_id);

-- ─── 好友备注 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_friend_remarks (
    user_id     BIGINT      NOT NULL,
    friend_id   BIGINT      NOT NULL,
    remark      VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, friend_id)
);

CREATE TRIGGER tr_user_friend_remarks_updated_at
    BEFORE UPDATE ON user_friend_remarks FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 登录会话（简化版：用 session_id 直接撤销）──────────────
CREATE TABLE IF NOT EXISTS sessions (
    session_id              UUID        PRIMARY KEY DEFAULT uuidv7(),
    user_id                 BIGINT      NOT NULL,
    device_type             VARCHAR(50),
    device_name             VARCHAR(100),
    device_fingerprint      TEXT,
    ip_address              INET,
    ip_location             VARCHAR(255),
    user_agent              TEXT,
    access_token_hash       VARCHAR(255) NOT NULL,
    refresh_token_hash      VARCHAR(255) NOT NULL,
    previous_refresh_token_hash VARCHAR(255),   -- Token 轮换：保留上一代 refresh_token hash
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMPTZ NOT NULL,
    is_revoked              BOOLEAN     NOT NULL DEFAULT FALSE,
    revoked_at              TIMESTAMPTZ
);
CREATE INDEX idx_sessions_user    ON sessions(user_id, created_at DESC);
CREATE INDEX idx_sessions_refresh ON sessions(refresh_token_hash) WHERE is_revoked = FALSE;
CREATE INDEX idx_sessions_prev_refresh ON sessions(previous_refresh_token_hash) WHERE previous_refresh_token_hash IS NOT NULL AND is_revoked = FALSE;

-- ─── 验证码 ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS verification_codes (
    code_id     UUID                    PRIMARY KEY DEFAULT uuidv7(),
    target      VARCHAR(255)            NOT NULL,   -- 手机号/邮箱
    code        VARCHAR(10)             NOT NULL,
    type        verification_code_type  NOT NULL,
    verify_token VARCHAR(255),                      -- 验证成功后生成的一次性 token
    expires_at  TIMESTAMPTZ             NOT NULL,
    used        BOOLEAN                 NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_verify_codes_target ON verification_codes(target, type, used);
CREATE INDEX idx_verify_codes_token  ON verification_codes(verify_token) WHERE verify_token IS NOT NULL AND used = FALSE;

-- ─── 密码重置 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS password_resets (
    reset_id    UUID        PRIMARY KEY DEFAULT uuidv7(),
    user_id     BIGINT      NOT NULL,
    token       VARCHAR(255) NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pwd_resets_token ON password_resets(token);

-- ─── 推送设备令牌 ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS device_tokens (
    token_id        UUID        PRIMARY KEY DEFAULT uuidv7(),
    user_id         BIGINT      NOT NULL,
    platform        VARCHAR(20) NOT NULL,   -- ios, android, harmony, web
    device_token    VARCHAR(512) NOT NULL,
    device_name     VARCHAR(100),
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, platform, device_token)
);
CREATE INDEX idx_device_tokens_user ON device_tokens(user_id);

-- ─── 位置历史 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_location_history (
    history_id  UUID        PRIMARY KEY DEFAULT uuidv7(),
    user_id     BIGINT      NOT NULL,
    location    GEOGRAPHY(POINT, 4326) NOT NULL,
    location_name VARCHAR(255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_user_loc_hist_user ON user_location_history(user_id, created_at DESC);
CREATE INDEX idx_user_loc_hist_geo  ON user_location_history USING GIST(location);

CREATE INDEX idx_users_region_id ON users(region_id);
