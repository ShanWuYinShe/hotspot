-- ============================================================
-- 004_content_module.sql
-- 内容模块：瞬间、评论、标签
-- ============================================================

-- ─── 瞬间表（分区）─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS moments_partitioned (
	    moment_id       UUID            NOT NULL DEFAULT uuidv7(),
	    user_id         BIGINT          NOT NULL,
	    description     TEXT            NOT NULL,
	    mentions        BIGINT[],                       -- @用户 ID 数组
	    -- PostGIS 位置
	    location        GEOGRAPHY(POINT, 4326) NOT NULL,
	    location_name   VARCHAR(255)    NOT NULL,
	    region_id       BIGINT,                         -- 行政区 region_id（最小行政区）
	    -- 隐私
	    privacy_level   privacy_level   NOT NULL DEFAULT 'public',
	    -- 时间戳
	    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
	    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
	    deleted_at      TIMESTAMPTZ,
	    PRIMARY KEY (moment_id, created_at)
	) PARTITION BY RANGE (created_at);

-- 默认分区（兜底）
CREATE TABLE IF NOT EXISTS moments_default PARTITION OF moments_partitioned DEFAULT;

-- 索引
CREATE INDEX idx_moments_user_feed        ON moments_partitioned(user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_moments_location         ON moments_partitioned USING GIST(location) WHERE deleted_at IS NULL;
CREATE INDEX idx_moments_global           ON moments_partitioned(created_at DESC) WHERE deleted_at IS NULL AND privacy_level = 'public';
CREATE INDEX idx_moments_mentions_gin     ON moments_partitioned USING GIN(mentions) WHERE deleted_at IS NULL;
CREATE INDEX idx_moments_partition_region_id ON moments_partitioned(region_id);

-- ─── 瞬间图片 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS moment_images (
    image_id    UUID        PRIMARY KEY DEFAULT uuidv7(),
    moment_id   UUID        NOT NULL,
    file_id     UUID,
    url         TEXT        NOT NULL,
    width       INTEGER,
    height      INTEGER,
    content_type VARCHAR(50),
    sort_order  INTEGER     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_moment_images_order ON moment_images(moment_id, sort_order);

-- ─── 瞬间标签关联 ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS moment_tags (
    moment_id   UUID        NOT NULL,
    tag_id      UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (moment_id, tag_id)
);
CREATE INDEX idx_moment_tags_tag ON moment_tags(tag_id);

-- ─── 瞬间 Emoji 反应 ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS moment_reactions (
    moment_id   UUID            NOT NULL,
    user_id     BIGINT          NOT NULL,
    type        reaction_type   NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    PRIMARY KEY (moment_id, user_id, type)
);
CREATE INDEX idx_moment_reactions_list ON moment_reactions(moment_id, created_at DESC);
CREATE INDEX idx_moment_reactions_user ON moment_reactions(user_id, created_at DESC);

-- ─── 瞬间收藏 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS moment_favorites (
    moment_id   UUID        NOT NULL,
    user_id     BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (moment_id, user_id)
);
CREATE INDEX idx_moment_favorites_user   ON moment_favorites(user_id, created_at DESC);
CREATE INDEX idx_moment_favorites_moment ON moment_favorites(moment_id, created_at DESC);

-- ─── 瞬间统计 Bucket ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS moment_stats_buckets (
    moment_id       UUID        NOT NULL,
    bucket_id       SMALLINT    NOT NULL,
    likes_count     INTEGER     NOT NULL DEFAULT 0,
    comments_count  INTEGER     NOT NULL DEFAULT 0,
    shares_count    INTEGER     NOT NULL DEFAULT 0,
    reactions_count INTEGER     NOT NULL DEFAULT 0,  -- emoji 反应总数（不含 like）
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (moment_id, bucket_id)
);
CREATE INDEX idx_moment_stats_moment ON moment_stats_buckets(moment_id);

-- ─── 评论表（分区）──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS comments_partitioned (
    comment_id          UUID        NOT NULL DEFAULT uuidv7(),
    moment_id           UUID        NOT NULL,
    user_id             BIGINT      NOT NULL,
    content             TEXT        NOT NULL,
    parent_id           UUID,                       -- 顶级评论为 NULL
    reply_to_comment_id UUID,                       -- 回复的具体评论
    reply_to_user_id    BIGINT,                     -- 回复的目标用户
    mentions            BIGINT[],
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    PRIMARY KEY (comment_id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS comments_default PARTITION OF comments_partitioned DEFAULT;

-- 索引
CREATE INDEX idx_comments_moment_root ON comments_partitioned(moment_id, created_at ASC)
    WHERE parent_id IS NULL AND deleted_at IS NULL;
CREATE INDEX idx_comments_replies     ON comments_partitioned(parent_id, created_at ASC)
    WHERE parent_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_comments_user        ON comments_partitioned(user_id, created_at DESC);
CREATE INDEX idx_comments_mentions    ON comments_partitioned USING GIN(mentions) WHERE deleted_at IS NULL;

-- ─── 评论反应 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS comment_reactions (
    comment_id  UUID            NOT NULL,
    user_id     BIGINT          NOT NULL,
    type        reaction_type   NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    PRIMARY KEY (comment_id, user_id, type)
);
CREATE INDEX idx_comment_reactions_list ON comment_reactions(comment_id, created_at DESC);

-- ─── 评论统计 Bucket ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS comment_stats_buckets (
    comment_id      UUID        NOT NULL,
    bucket_id       SMALLINT    NOT NULL,
    likes_count     INTEGER     NOT NULL DEFAULT 0,
    replies_count   INTEGER     NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (comment_id, bucket_id)
);
CREATE INDEX idx_comment_stats_comment ON comment_stats_buckets(comment_id);

-- ─── 标签主表 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tags (
    tag_id      UUID        PRIMARY KEY DEFAULT uuidv7(),
    name        VARCHAR(50) UNIQUE NOT NULL,
    usage_count INTEGER     NOT NULL DEFAULT 0,
    is_blocked  BOOLEAN     NOT NULL DEFAULT FALSE,  -- 管理员屏蔽
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tags_usage     ON tags(usage_count DESC) WHERE is_blocked = FALSE;
CREATE INDEX idx_tags_name_trgm ON tags USING GIN(name gin_trgm_ops);

-- ─── 标签搜索统计（支持趋势计算）──────────────────────────
CREATE TABLE IF NOT EXISTS tag_search_stats (
    tag_id          UUID    NOT NULL,
    search_date     DATE    NOT NULL,
    search_count    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (tag_id, search_date)
);
