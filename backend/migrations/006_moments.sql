-- 生活瞬间表
CREATE TABLE moments (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_name VARCHAR(255) NOT NULL,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    tags TEXT[],
    images TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT moments_deleted_check CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL)
    )
);

-- 瞬间索引
CREATE INDEX idx_moments_id ON moments(id DESC);
CREATE UNIQUE INDEX idx_moments_moment_id ON moments(moment_id);
CREATE INDEX idx_moments_user ON moments(user_id, id DESC);
CREATE INDEX idx_moments_location ON moments USING GIST(location);
CREATE INDEX idx_moments_tags ON moments USING GIN(tags);
CREATE INDEX idx_moments_created ON moments(created_at DESC);
CREATE INDEX idx_moments_not_deleted ON moments(is_deleted) WHERE is_deleted = FALSE;

-- 自动更新 moments location 字段的触发器
CREATE OR REPLACE FUNCTION update_moment_location()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_moment_location
BEFORE INSERT OR UPDATE ON moments
FOR EACH ROW
EXECUTE FUNCTION update_moment_location();

-- 瞬间点赞表
CREATE TABLE moment_likes (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (moment_id, user_id)
);

-- 点赞索引
CREATE INDEX idx_moment_likes_id ON moment_likes(id DESC);
CREATE INDEX idx_moment_likes_moment ON moment_likes(moment_id);
CREATE INDEX idx_moment_likes_user ON moment_likes(user_id);

-- 瞬间收藏表
CREATE TABLE moment_collections (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (moment_id, user_id)
);

-- 收藏索引
CREATE INDEX idx_moment_collections_id ON moment_collections(id DESC);
CREATE INDEX idx_moment_collections_user ON moment_collections(user_id, id DESC);
CREATE INDEX idx_moment_collections_moment ON moment_collections(moment_id);

