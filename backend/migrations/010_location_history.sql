-- 用户位置历史表
CREATE TABLE user_location_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.154922,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.682524,
    location_name VARCHAR(255) NOT NULL DEFAULT '南天门',
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_location_history_user ON user_location_history(user_id, created_at DESC);
CREATE INDEX idx_user_location_history_location ON user_location_history USING GIST(location);

-- 聊天室位置历史表（包括公开群组和私密群组）
CREATE TABLE room_location_history (
    id BIGSERIAL PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES chat_rooms(room_id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.154922,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.682524,
    location_name VARCHAR(255) NOT NULL DEFAULT '南天门',
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_room_location_history_room ON room_location_history(room_id, created_at DESC);
CREATE INDEX idx_room_location_history_location ON room_location_history USING GIST(location);

-- 触发器：自动更新用户位置历史的location字段
CREATE OR REPLACE FUNCTION update_user_location_history_location()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_location_history_location
BEFORE INSERT ON user_location_history
FOR EACH ROW
EXECUTE FUNCTION update_user_location_history_location();

-- 触发器：自动更新聊天室位置历史的location字段
CREATE OR REPLACE FUNCTION update_room_location_history_location()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_room_location_history_location
BEFORE INSERT ON room_location_history
FOR EACH ROW
EXECUTE FUNCTION update_room_location_history_location();



