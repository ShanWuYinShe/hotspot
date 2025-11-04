-- 评论表
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    comment_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    moment_id UUID NOT NULL REFERENCES moments(moment_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    parent_id UUID REFERENCES comments(comment_id) ON DELETE CASCADE,
    reply_to_user_id BIGINT REFERENCES users(user_id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT comments_deleted_check CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL)
    )
);

-- 评论索引
CREATE INDEX idx_comments_id ON comments(id DESC);
CREATE UNIQUE INDEX idx_comments_comment_id ON comments(comment_id);
CREATE INDEX idx_comments_moment ON comments(moment_id, id DESC);
CREATE INDEX idx_comments_user ON comments(user_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);
CREATE INDEX idx_comments_updated ON comments(updated_at DESC);
CREATE INDEX idx_comments_not_deleted ON comments(is_deleted) WHERE is_deleted = FALSE;

-- 评论点赞表
CREATE TABLE comment_likes (
    id BIGSERIAL PRIMARY KEY,
    comment_id UUID NOT NULL REFERENCES comments(comment_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE (comment_id, user_id)
);

-- 评论点赞索引
CREATE INDEX idx_comment_likes_id ON comment_likes(id DESC);
CREATE INDEX idx_comment_likes_comment ON comment_likes(comment_id);
CREATE INDEX idx_comment_likes_user ON comment_likes(user_id);

