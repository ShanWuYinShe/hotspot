-- ============================================================
-- 003_group_module.sql
-- 群组模块：群组、成员、申请、邀请、封禁、公告
-- ============================================================

-- ─── 群组表（合并原 group_settings）─────────────────────────
CREATE TABLE IF NOT EXISTS groups (
    group_id        BIGSERIAL       PRIMARY KEY,
    name            VARCHAR(100)    NOT NULL,
    description     TEXT,
    avatar_url      TEXT,
    owner_id        BIGINT          NOT NULL,
    -- 群组设置（原 group_settings 字段合入）
    join_method     group_join_method NOT NULL DEFAULT 'open',
    is_searchable   BOOLEAN         NOT NULL DEFAULT TRUE,
    max_members     INTEGER         NOT NULL DEFAULT 200,
    password_hash   TEXT,                           -- open + 密码场景
    -- PostGIS 位置
    location        GEOGRAPHY(POINT, 4326) NOT NULL,
    location_name   VARCHAR(255)    NOT NULL,
    -- 活跃度
    last_activity_at TIMESTAMPTZ,
    -- 时间戳
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    dismissed_at    TIMESTAMPTZ                     -- 软删除（解散）
);

CREATE INDEX idx_groups_owner      ON groups(owner_id);
CREATE INDEX idx_groups_name_trgm  ON groups USING GIN(name gin_trgm_ops);
CREATE INDEX idx_groups_location   ON groups USING GIST(location) WHERE dismissed_at IS NULL;
CREATE INDEX idx_groups_active     ON groups(last_activity_at DESC) WHERE dismissed_at IS NULL;
CREATE INDEX idx_groups_searchable ON groups(created_at DESC) WHERE dismissed_at IS NULL AND is_searchable = TRUE;

CREATE TRIGGER tr_groups_updated_at
    BEFORE UPDATE ON groups FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 群组成员 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_members (
    group_id    BIGINT      NOT NULL,
    user_id     BIGINT      NOT NULL,
    role        group_role  NOT NULL DEFAULT 'member',
    is_muted    BOOLEAN     NOT NULL DEFAULT FALSE,
    muted_until TIMESTAMPTZ,
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    inviter_id  BIGINT,
    PRIMARY KEY (group_id, user_id)
);
-- 查某群的成员列表
CREATE INDEX idx_group_members_joined ON group_members(group_id, joined_at DESC);
-- 查某用户加入了哪些群
CREATE INDEX idx_group_members_user   ON group_members(user_id, joined_at DESC);

-- ─── 用户群组偏好（免打扰/置顶）──────────────────────────
CREATE TABLE IF NOT EXISTS group_user_preferences (
    group_id    BIGINT      NOT NULL,
    user_id     BIGINT      NOT NULL,
    is_muted    BOOLEAN     NOT NULL DEFAULT FALSE,
    is_pinned   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);
CREATE INDEX idx_group_user_prefs_user ON group_user_preferences(user_id, updated_at DESC);

CREATE TRIGGER tr_group_user_prefs_updated_at
    BEFORE UPDATE ON group_user_preferences FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 群组统计 Bucket ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_stats_buckets (
    group_id        BIGINT      NOT NULL,
    bucket_id       SMALLINT    NOT NULL,
    member_count    INTEGER     NOT NULL DEFAULT 0,
    message_count   INTEGER     NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, bucket_id)
);
CREATE INDEX idx_group_stats_group ON group_stats_buckets(group_id);

-- ─── 群组加入申请（用户 → 群组）──────────────────────────
CREATE TABLE IF NOT EXISTS group_applications (
    application_id  UUID               PRIMARY KEY DEFAULT uuidv7(),
    group_id        BIGINT             NOT NULL,
    user_id         BIGINT             NOT NULL,    -- 申请人
    message         TEXT,
    status          application_status NOT NULL DEFAULT 'pending',
    handled_by      BIGINT,
    handled_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);
-- 管理员看某群的申请列表
CREATE INDEX idx_group_apps_group  ON group_applications(group_id, status, created_at DESC);
-- 用户看自己的申请列表
CREATE INDEX idx_group_apps_user   ON group_applications(user_id, status, created_at DESC);
-- 防重：同一用户对同一群组只能有一个 pending 申请
CREATE UNIQUE INDEX idx_group_apps_pending_unique
    ON group_applications(group_id, user_id) WHERE status = 'pending';

-- ─── 群组邀请（成员 → 用户）──────────────────────────────
CREATE TABLE IF NOT EXISTS group_invites (
    invite_id   UUID               PRIMARY KEY DEFAULT uuidv7(),
    group_id    BIGINT             NOT NULL,
    inviter_id  BIGINT             NOT NULL,        -- 邀请人
    invitee_id  BIGINT             NOT NULL,        -- 被邀请人
    message     TEXT,
    status      application_status NOT NULL DEFAULT 'pending',
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);
-- 被邀请人看自己的邀请列表
CREATE INDEX idx_group_invites_invitee ON group_invites(invitee_id, status, created_at DESC);
-- 某群的邀请列表
CREATE INDEX idx_group_invites_group   ON group_invites(group_id, created_at DESC);
-- 防重：同一群组对同一用户只能有一个 pending 邀请
CREATE UNIQUE INDEX idx_group_invites_pending_unique
    ON group_invites(group_id, invitee_id) WHERE status = 'pending';

-- ─── 群组封禁 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_bans (
    ban_id      UUID        PRIMARY KEY DEFAULT uuidv7(),
    group_id    BIGINT      NOT NULL,
    user_id     BIGINT      NOT NULL,
    banned_by   BIGINT      NOT NULL,
    reason      TEXT,
    banned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unbanned_at TIMESTAMPTZ,
    unbanned_by BIGINT,
    UNIQUE (group_id, user_id)
);
CREATE INDEX idx_group_bans_group ON group_bans(group_id, banned_at DESC);
CREATE INDEX idx_group_bans_user  ON group_bans(user_id);

-- ─── 群公告 ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_announcements (
    announcement_id UUID        PRIMARY KEY DEFAULT uuidv7(),
    group_id        BIGINT      NOT NULL,
    publisher_id    BIGINT      NOT NULL,
    title           VARCHAR(200) NOT NULL,
    content         TEXT        NOT NULL,
    is_pinned       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_group_announcements_list
    ON group_announcements(group_id, is_pinned DESC, created_at DESC) WHERE deleted_at IS NULL;

CREATE TRIGGER tr_group_announcements_updated_at
    BEFORE UPDATE ON group_announcements FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ─── 群组标签关联 ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_tags (
    group_id    BIGINT      NOT NULL,
    tag_id      UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, tag_id)
);
CREATE INDEX idx_group_tags_tag ON group_tags(tag_id);

-- ─── 群组成员变更历史 ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_member_history (
    history_id  UUID            PRIMARY KEY DEFAULT uuidv7(),
    group_id    BIGINT          NOT NULL,
    user_id     BIGINT          NOT NULL,
    action      member_action   NOT NULL,
    operator_id BIGINT,
    old_role    group_role,
    new_role    group_role,
    reason      TEXT,
    muted_until TIMESTAMPTZ,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_group_member_history ON group_member_history(group_id, created_at DESC);
