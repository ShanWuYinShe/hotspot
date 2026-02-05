-- ============================================================
-- 003_social_module.sql
-- 社交模块：瞬间、评论、图片 (分区表)
-- ============================================================

-- 1. 瞬间表 (分区表)
CREATE TABLE IF NOT EXISTS moments_partitioned (
    moment_id UUID NOT NULL DEFAULT uuidv7(),
    user_id BIGINT NOT NULL,
    description TEXT,
    location GEOGRAPHY(POINT, 4326),
    location_name VARCHAR(100),
    privacy_level VARCHAR(20) DEFAULT 'public', -- public, private, friends
    
    view_count INTEGER DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    comments_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    PRIMARY KEY (moment_id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建默认分区
CREATE TABLE IF NOT EXISTS moments_default PARTITION OF moments_partitioned DEFAULT;

-- 创建初始分区 (2025, 2026)
DO $$
BEGIN
    PERFORM create_partition('moments_2025', 'moments_partitioned', '2025-01-01', '2026-01-01');
    PERFORM create_partition('moments_2026', 'moments_partitioned', '2026-01-01', '2027-01-01');
END $$;

-- 索引
CREATE INDEX IF NOT EXISTS idx_moments_user ON moments_partitioned(user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_moments_location ON moments_partitioned USING GIST(location) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_moments_created_at ON moments_partitioned(created_at DESC);

-- 重命名 (如果尚未重命名)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'moments') THEN
        ALTER TABLE moments_partitioned RENAME TO moments;
    END IF;
END $$;

-- 2. 评论表 (分区表)
CREATE TABLE IF NOT EXISTS comments_partitioned (
    comment_id UUID NOT NULL DEFAULT uuidv7(),
    moment_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    parent_id UUID, -- 回复某条评论
    reply_to_user_id BIGINT, -- 回复某个用户
    
    like_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    PRIMARY KEY (comment_id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建默认分区
CREATE TABLE IF NOT EXISTS comments_default PARTITION OF comments_partitioned DEFAULT;

-- 创建初始分区 (2025, 2026)
DO $$
BEGIN
    PERFORM create_partition('comments_2025', 'comments_partitioned', '2025-01-01', '2026-01-01');
    PERFORM create_partition('comments_2026', 'comments_partitioned', '2026-01-01', '2027-01-01');
END $$;

-- 索引
CREATE INDEX IF NOT EXISTS idx_comments_moment ON comments_partitioned(moment_id, created_at ASC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_comments_user ON comments_partitioned(user_id, created_at DESC);

-- 重命名
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'comments') THEN
        ALTER TABLE comments_partitioned RENAME TO comments;
    END IF;
END $$;

-- 3. 瞬间图片表 (无外键)
CREATE TABLE IF NOT EXISTS moment_images (
    image_id UUID PRIMARY KEY DEFAULT uuidv7(),
    moment_id UUID NOT NULL,
    file_id UUID, -- 引用 media_files.id (逻辑引用)
    
    url TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    sort_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_moment_images_moment ON moment_images(moment_id, sort_order);

-- 4. 瞬间点赞/反应记录 (无外键)
CREATE TABLE IF NOT EXISTS moment_reactions (
    moment_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    type reaction_type DEFAULT 'like',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (moment_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_moment_reactions_moment ON moment_reactions(moment_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_moment_reactions_user ON moment_reactions(user_id, created_at DESC);
