-- 动态系统相关表
-- 包含: tags, moments, moment_tags, moment_likes

-- 标签表
CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    tag_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    counter INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_tags_name ON tags(name);
CREATE INDEX idx_tags_counter ON tags(counter DESC);

-- 动态表
CREATE TABLE IF NOT EXISTS moments (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    content TEXT,
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.16,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.68,
    location_name TEXT NOT NULL DEFAULT '南天门',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    geom GEOGRAPHY(POINT, 4326),
    
    CONSTRAINT fk_moments_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE TRIGGER update_moments_geom
BEFORE INSERT OR UPDATE OF latitude, longitude ON moments
FOR EACH ROW EXECUTE FUNCTION update_geom_column();

CREATE INDEX idx_moments_geom ON moments USING gist (geom);
CREATE INDEX idx_moments_user_id ON moments(user_id);
CREATE INDEX idx_moments_created_at ON moments(created_at);

-- 动态标签关联表
CREATE TABLE IF NOT EXISTS moment_tags (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    
    CONSTRAINT moment_tags_moment_tag_unique UNIQUE (moment_id, tag_id),
    CONSTRAINT fk_moment_tags_moment_id 
        FOREIGN KEY (moment_id) REFERENCES moments(moment_id) ON DELETE CASCADE,
    CONSTRAINT fk_moment_tags_tag_id 
        FOREIGN KEY (tag_id) REFERENCES tags(tag_id) ON DELETE CASCADE
);

CREATE INDEX idx_moment_tags_moment_id ON moment_tags(moment_id);
CREATE INDEX idx_moment_tags_tag_id ON moment_tags(tag_id);

-- 动态点赞表
CREATE TABLE IF NOT EXISTS moment_likes (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT moment_likes_moment_user_unique UNIQUE (moment_id, user_id),
    CONSTRAINT fk_moment_likes_moment_id 
        FOREIGN KEY (moment_id) REFERENCES moments(moment_id) ON DELETE CASCADE,
    CONSTRAINT fk_moment_likes_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_moment_likes_moment_id ON moment_likes(moment_id);
CREATE INDEX idx_moment_likes_user_id ON moment_likes(user_id);
CREATE INDEX idx_moment_likes_created_at ON moment_likes(created_at);
