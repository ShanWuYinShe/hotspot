-- 信息流反馈表
CREATE TABLE feed_feedback (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    item_id VARCHAR(255) NOT NULL,
    item_type VARCHAR(20) NOT NULL CHECK (item_type IN ('moment', 'group_card', 'system')),
    action VARCHAR(20) NOT NULL CHECK (action IN ('hide', 'not_interested', 'clicked', 'viewed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 信息流反馈索引
CREATE INDEX idx_feed_feedback_id ON feed_feedback(id DESC);
CREATE INDEX idx_feed_feedback_user ON feed_feedback(user_id, id DESC);
CREATE INDEX idx_feed_feedback_item ON feed_feedback(item_id, item_type);

-- 媒体文件元数据表
CREATE TABLE media_files (
    id BIGSERIAL PRIMARY KEY,
    file_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    category VARCHAR(20) NOT NULL CHECK (category IN ('avatar', 'group_image', 'moment_image', 'file')),
    file_url TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 媒体文件索引
CREATE INDEX idx_media_files_id ON media_files(id DESC);
CREATE UNIQUE INDEX idx_media_files_file_id ON media_files(file_id);
CREATE INDEX idx_media_files_user ON media_files(user_id, id DESC);
CREATE INDEX idx_media_files_category ON media_files(category);

