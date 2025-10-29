-- 私聊会话表
CREATE TABLE conversations (
    conversation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    user2_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_message_at TIMESTAMP WITH TIME ZONE,
    UNIQUE (user1_id, user2_id),
    CHECK (user1_id < user2_id)
);

-- 会话索引
CREATE INDEX idx_conversations_user1 ON conversations(user1_id, last_message_at DESC);
CREATE INDEX idx_conversations_user2 ON conversations(user2_id, last_message_at DESC);

-- 消息表
CREATE TABLE messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id VARCHAR(255) NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('conversation', 'group')),
    sender_id BIGINT REFERENCES users(user_id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    message_type VARCHAR(20) DEFAULT 'text' CHECK (message_type IN ('text', 'image', 'video', 'audio', 'file', 'location', 'system')),
    is_edited BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    edited_at TIMESTAMP WITH TIME ZONE,
    reply_to_id UUID REFERENCES messages(message_id) ON DELETE SET NULL,
    mentions BIGINT[],
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    location_name VARCHAR(255),
    file_name VARCHAR(255),
    file_size BIGINT,
    thumbnail_url TEXT,
    duration INTEGER,
    system_event VARCHAR(50),
    affected_users BIGINT[]
);

-- 消息索引
CREATE INDEX idx_messages_target ON messages(target_id, target_type, created_at DESC);
CREATE INDEX idx_messages_sender ON messages(sender_id, created_at DESC);
CREATE INDEX idx_messages_reply ON messages(reply_to_id);

-- 消息已读记录表
CREATE TABLE message_read_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    target_id VARCHAR(255) NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('conversation', 'group')),
    last_read_message_id UUID REFERENCES messages(message_id) ON DELETE SET NULL,
    read_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (user_id, target_id, target_type)
);

-- 已读记录索引
CREATE INDEX idx_message_read_user_target ON message_read_records(user_id, target_id, target_type);

-- 消息反应表
CREATE TABLE message_reactions (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    reaction_type VARCHAR(20) NOT NULL CHECK (reaction_type IN ('like', 'love', 'laugh', 'angry', 'sad', 'wow')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (message_id, user_id, reaction_type)
);

-- 反应索引
CREATE INDEX idx_message_reactions_message ON message_reactions(message_id);
CREATE INDEX idx_message_reactions_user ON message_reactions(user_id);

-- 生活瞬间表
CREATE TABLE moments (
    moment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_name VARCHAR(255) NOT NULL,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    tags TEXT[],
    images TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE
);

-- 瞬间索引
CREATE INDEX idx_moments_user ON moments(user_id, created_at DESC);
CREATE INDEX idx_moments_location ON moments USING GIST(location);
CREATE INDEX idx_moments_tags ON moments USING GIN(tags);
CREATE INDEX idx_moments_created ON moments(created_at DESC);

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
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (moment_id, user_id)
);

-- 点赞索引
CREATE INDEX idx_moment_likes_moment ON moment_likes(moment_id);
CREATE INDEX idx_moment_likes_user ON moment_likes(user_id);

-- 瞬间收藏表
CREATE TABLE moment_collections (
    id BIGSERIAL PRIMARY KEY,
    moment_id UUID NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (moment_id, user_id)
);

-- 收藏索引
CREATE INDEX idx_moment_collections_user ON moment_collections(user_id, created_at DESC);
CREATE INDEX idx_moment_collections_moment ON moment_collections(moment_id);

-- 评论表
CREATE TABLE comments (
    comment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    moment_id UUID NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    parent_id UUID REFERENCES comments(comment_id) ON DELETE CASCADE,
    reply_to_user_id BIGINT REFERENCES users(user_id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 评论索引
CREATE INDEX idx_comments_moment ON comments(moment_id, created_at DESC);
CREATE INDEX idx_comments_user ON comments(user_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);

-- 评论点赞表
CREATE TABLE comment_likes (
    id BIGSERIAL PRIMARY KEY,
    comment_id UUID NOT NULL REFERENCES comments(comment_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (comment_id, user_id)
);

-- 评论点赞索引
CREATE INDEX idx_comment_likes_comment ON comment_likes(comment_id);
CREATE INDEX idx_comment_likes_user ON comment_likes(user_id);

-- 举报表
CREATE TABLE reports (
    report_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('user', 'moment', 'comment', 'message', 'group')),
    target_id VARCHAR(255) NOT NULL,
    reason VARCHAR(100) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'reviewing', 'resolved', 'dismissed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    handled_at TIMESTAMP WITH TIME ZONE,
    handler_id BIGINT REFERENCES users(user_id),
    handler_note TEXT
);

-- 举报索引
CREATE INDEX idx_reports_reporter ON reports(reporter_id);
CREATE INDEX idx_reports_target ON reports(target_type, target_id);
CREATE INDEX idx_reports_status ON reports(status, created_at DESC);

-- 会话通知设置表
CREATE TABLE conversation_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    target_id VARCHAR(255) NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('conversation', 'group')),
    muted BOOLEAN DEFAULT FALSE,
    mention_only BOOLEAN DEFAULT FALSE,
    pinned BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (user_id, target_id, target_type)
);

-- 会话设置索引
CREATE INDEX idx_conversation_settings_user ON conversation_settings(user_id);

