-- 群组系统相关表
-- 包含: groups, group_members, group_messages, group_events

-- 群组角色枚举类型
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'group_role') THEN
        CREATE TYPE group_role AS ENUM ('admin', 'owner', 'member');
    END IF;
END $$;

-- 群组主表
CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    group_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT,
    owner_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.16,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.68,
    location_name TEXT NOT NULL DEFAULT '南天门',
    password_hash TEXT,
    geom GEOGRAPHY(POINT, 4326),
    
    CONSTRAINT fk_groups_owner_id 
        FOREIGN KEY (owner_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE TRIGGER update_groups_geom
BEFORE INSERT OR UPDATE OF latitude, longitude ON groups
FOR EACH ROW EXECUTE FUNCTION update_geom_column();

CREATE INDEX idx_groups_geom ON groups USING gist (geom);
CREATE INDEX idx_groups_owner_id ON groups(owner_id);
CREATE INDEX idx_groups_last_active_at ON groups(last_active_at);

-- 群组成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    role group_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notification_level VARCHAR(20) DEFAULT 'all' CHECK (notification_level IN ('all', 'mentions', 'none')),
    last_active TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT group_members_group_user_unique UNIQUE (group_id, user_id),
    CONSTRAINT fk_group_members_group_id 
        FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE,
    CONSTRAINT fk_group_members_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_group_members_group_id ON group_members(group_id);
CREATE INDEX idx_group_members_user_id ON group_members(user_id);
CREATE INDEX idx_group_members_notification_level ON group_members(notification_level);
CREATE INDEX idx_group_members_last_active ON group_members(last_active);

-- 群组消息表
CREATE TABLE IF NOT EXISTS group_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    group_id UUID NOT NULL,
    sender_id BIGINT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    latitude DOUBLE PRECISION NOT NULL DEFAULT 35.16,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 112.68,
    geom GEOGRAPHY(POINT, 4326),
    
    CONSTRAINT fk_group_messages_group_id 
        FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE,
    CONSTRAINT fk_group_messages_sender_id 
        FOREIGN KEY (sender_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE TRIGGER update_group_messages_geom
BEFORE INSERT OR UPDATE OF latitude, longitude ON group_messages
FOR EACH ROW EXECUTE FUNCTION update_geom_column();

CREATE INDEX idx_group_messages_group_id ON group_messages(group_id);
CREATE INDEX idx_group_messages_created_at ON group_messages(created_at);
CREATE INDEX idx_group_messages_sender_id ON group_messages(sender_id);
CREATE INDEX idx_group_messages_geom ON group_messages USING gist (geom);

-- 群组事件表
CREATE TABLE IF NOT EXISTS group_events (
    id BIGSERIAL PRIMARY KEY,
    group_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    target_user_id BIGINT,
    event_type VARCHAR(10) NOT NULL CHECK (event_type IN ('create', 'join', 'leave', 'update', 'kick', 'transfer')),
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_group_events_group_id 
        FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE,
    CONSTRAINT fk_group_events_user_id 
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_group_events_target_user_id 
        FOREIGN KEY (target_user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_group_events_group_id ON group_events(group_id);
CREATE INDEX idx_group_events_user_id ON group_events(user_id);
CREATE INDEX idx_group_events_target_user_id ON group_events(target_user_id);
CREATE INDEX idx_group_events_event_type ON group_events(event_type);
CREATE INDEX idx_group_events_created_at ON group_events(created_at);
