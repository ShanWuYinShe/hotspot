-- 签到系统相关表
-- 包含: tags, checkins, checkin_tags, checkin_likes

-- 标签表
CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    tag_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    counter INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_tags_name ON tags(name);
CREATE INDEX idx_tags_counter ON tags(counter DESC);

-- 签到表
CREATE TABLE IF NOT EXISTS checkins (
    id BIGSERIAL PRIMARY KEY,
    checkin_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    description TEXT,
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.16,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.68,
    location_name TEXT NOT NULL DEFAULT '南天门',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    geom GEOGRAPHY(POINT, 4326),
    
    CONSTRAINT fk_checkins_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE TRIGGER update_checkins_geom
BEFORE INSERT OR UPDATE OF latitude, longitude ON checkins
FOR EACH ROW EXECUTE FUNCTION update_geom_column();

CREATE INDEX idx_checkins_geom ON checkins USING gist (geom);
CREATE INDEX idx_checkins_user_id ON checkins(user_id);
CREATE INDEX idx_checkins_created_at ON checkins(created_at);

-- 签到标签关联表
CREATE TABLE IF NOT EXISTS checkin_tags (
    id BIGSERIAL PRIMARY KEY,
    checkin_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    
    CONSTRAINT checkin_tags_checkin_tag_unique UNIQUE (checkin_id, tag_id),
    CONSTRAINT fk_checkin_tags_checkin_id 
        FOREIGN KEY (checkin_id) REFERENCES checkins(checkin_id) ON DELETE CASCADE,
    CONSTRAINT fk_checkin_tags_tag_id 
        FOREIGN KEY (tag_id) REFERENCES tags(tag_id) ON DELETE CASCADE
);

CREATE INDEX idx_checkin_tags_checkin_id ON checkin_tags(checkin_id);
CREATE INDEX idx_checkin_tags_tag_id ON checkin_tags(tag_id);

-- 签到点赞表
CREATE TABLE IF NOT EXISTS checkin_likes (
    id BIGSERIAL PRIMARY KEY,
    checkin_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT checkin_likes_checkin_user_unique UNIQUE (checkin_id, user_id),
    CONSTRAINT fk_checkin_likes_checkin_id 
        FOREIGN KEY (checkin_id) REFERENCES checkins(checkin_id) ON DELETE CASCADE,
    CONSTRAINT fk_checkin_likes_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_checkin_likes_checkin_id ON checkin_likes(checkin_id);
CREATE INDEX idx_checkin_likes_user_id ON checkin_likes(user_id);
CREATE INDEX idx_checkin_likes_created_at ON checkin_likes(created_at);
