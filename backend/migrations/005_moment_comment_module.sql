-- 005_moment_comment_module.sql
-- 生活瞬间与评论模块

-- ============================================================
-- 生活瞬间
-- ============================================================
CREATE TABLE moments (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    description TEXT NOT NULL,
    images TEXT[],
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_name VARCHAR(255) NOT NULL,
    location geography(POINT, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT check_moment_latitude_range CHECK (latitude >= -90 AND latitude <= 90),
    CONSTRAINT check_moment_longitude_range CHECK (longitude >= -180 AND longitude <= 180)
);

CREATE INDEX idx_moments_user_id ON moments (user_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_moments_location ON moments USING gist (location) WHERE is_deleted = FALSE;
CREATE INDEX idx_moments_created_at ON moments (created_at DESC) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_moments_updated_at
    BEFORE UPDATE ON moments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_moments_location_sync
    BEFORE INSERT OR UPDATE OF latitude, longitude ON moments
    FOR EACH ROW EXECUTE FUNCTION sync_moment_location();

CREATE TABLE moment_likes (
    id BIGSERIAL PRIMARY KEY,
    like_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    moment_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_moment_like UNIQUE (moment_id, user_id)
);

CREATE INDEX idx_moment_likes_id ON moment_likes (like_id);
CREATE INDEX idx_moment_likes_moment_id ON moment_likes (moment_id, created_at DESC);
CREATE INDEX idx_moment_likes_user_id ON moment_likes (user_id, created_at DESC);

CREATE TABLE moment_favorites (
    id BIGSERIAL PRIMARY KEY,
    favorite_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    moment_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_moment_favorite UNIQUE (moment_id, user_id)
);

CREATE INDEX idx_moment_favorites_id ON moment_favorites (favorite_id);
CREATE INDEX idx_moment_favorites_user_id ON moment_favorites (user_id, created_at DESC);
CREATE INDEX idx_moment_favorites_moment_id ON moment_favorites (moment_id);

-- ============================================================
-- 评论
-- ============================================================
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    comment_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    moment_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    parent_id UUID,
    reply_to_user_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_comments_moment_id ON comments (moment_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_comments_parent_id ON comments (parent_id, created_at DESC) WHERE parent_id IS NOT NULL AND is_deleted = FALSE;
CREATE INDEX idx_comments_user_id ON comments (user_id, created_at DESC) WHERE is_deleted = FALSE;

CREATE TRIGGER trigger_comments_updated_at
    BEFORE UPDATE ON comments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE comment_likes (
    id BIGSERIAL PRIMARY KEY,
    like_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    comment_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_comment_like UNIQUE (comment_id, user_id)
);

CREATE INDEX idx_comment_likes_id ON comment_likes (like_id);
CREATE INDEX idx_comment_likes_comment_id ON comment_likes (comment_id, created_at DESC);
CREATE INDEX idx_comment_likes_user_id ON comment_likes (user_id, created_at DESC);


