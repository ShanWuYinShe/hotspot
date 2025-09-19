-- ============================================================================
-- HotSpot 社交功能模块迁移 v2.0
-- ============================================================================
-- 
-- 功能职责：
-- - 好友关系管理
-- - 用户屏蔽功能
-- - 瞬间（动态）发布
-- - 评论和互动
--
-- 依赖关系：
-- - 依赖 00_foundation.sql 的基础设施
-- - 依赖 01_users_core.sql 的用户系统
-- - 依赖 02_groups_core.sql 的群组系统
-- - 依赖 03_messages_core.sql 的消息系统
--
-- ============================================================================

-- 好友关系表
CREATE TABLE friendships (
    id BIGSERIAL PRIMARY KEY,
    
    -- 好友关系双方
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 好友关系状态
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'accepted', 'blocked', 'declined'
    
    -- 关系元数据
    nickname VARCHAR(100), -- 用户给好友设置的昵称
    notes TEXT, -- 用户对好友的备注
    
    -- 隐私设置
    show_online_status BOOLEAN DEFAULT true,
    show_location BOOLEAN DEFAULT true,
    allow_messages BOOLEAN DEFAULT true,
    
    -- 关系统计
    interaction_count INTEGER DEFAULT 0,
    last_interaction_at TIMESTAMP WITH TIME ZONE,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    accepted_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    CONSTRAINT friendships_different_users CHECK (user_id != friend_id),
    CONSTRAINT friendships_interaction_count_non_negative CHECK (interaction_count >= 0),
    
    -- 唯一约束（防止重复好友关系）
    UNIQUE(user_id, friend_id)
);

-- 用户屏蔽表
CREATE TABLE user_blocks (
    id BIGSERIAL PRIMARY KEY,
    
    -- 屏蔽关系双方
    blocker_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 屏蔽类型和原因
    block_type VARCHAR(20) DEFAULT 'full', -- 'full', 'messages_only', 'groups_only'
    reason VARCHAR(100),
    
    -- 屏蔽设置
    block_messages BOOLEAN DEFAULT true,
    block_group_invites BOOLEAN DEFAULT true,
    block_friend_requests BOOLEAN DEFAULT true,
    hide_from_search BOOLEAN DEFAULT true,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE, -- NULL表示永久屏蔽
    
    -- 约束
    CONSTRAINT user_blocks_different_users CHECK (blocker_id != blocked_id),
    
    -- 唯一约束（防止重复屏蔽）
    UNIQUE(blocker_id, blocked_id)
);

-- 瞬间（动态）表
CREATE TABLE moments (
    id BIGSERIAL PRIMARY KEY,
    moment_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    
    -- 发布者信息
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 内容信息
    content TEXT NOT NULL,
    content_type VARCHAR(20) DEFAULT 'text', -- 'text', 'image', 'video', 'mixed'
    
    -- 位置信息
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    location_name VARCHAR(200),
    address TEXT,
    
    -- PostGIS地理点
    location GEOGRAPHY(POINT, 4326) GENERATED ALWAYS AS (
        CASE 
            WHEN latitude IS NOT NULL AND longitude IS NOT NULL 
            THEN ST_Point(longitude, latitude) 
            ELSE NULL 
        END
    ) STORED,
    
    -- 可见性设置
    visibility VARCHAR(20) DEFAULT 'public', -- 'public', 'friends', 'private', 'group'
    target_group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    
    -- 互动统计
    like_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    share_count INTEGER DEFAULT 0,
    view_count INTEGER DEFAULT 0,
    
    -- 内容设置
    allow_comments BOOLEAN DEFAULT true,
    allow_shares BOOLEAN DEFAULT true,
    
    -- 扩展数据
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- 状态
    is_deleted BOOLEAN DEFAULT false,
    is_pinned BOOLEAN DEFAULT false,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    CONSTRAINT moments_content_not_empty CHECK (char_length(content) > 0),
    CONSTRAINT moments_location_coordinates CHECK (
        (latitude IS NULL AND longitude IS NULL) OR
        (latitude IS NOT NULL AND longitude IS NOT NULL AND validate_coordinates(latitude, longitude))
    ),
    CONSTRAINT moments_like_count_non_negative CHECK (like_count >= 0),
    CONSTRAINT moments_comment_count_non_negative CHECK (comment_count >= 0),
    CONSTRAINT moments_share_count_non_negative CHECK (share_count >= 0),
    CONSTRAINT moments_view_count_non_negative CHECK (view_count >= 0)
);

-- 瞬间媒体表
CREATE TABLE moment_media (
    id BIGSERIAL PRIMARY KEY,
    moment_id BIGINT NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
    
    -- 媒体信息
    media_type VARCHAR(50) NOT NULL, -- 'image', 'video', 'audio'
    file_name VARCHAR(255),
    file_size BIGINT,
    mime_type VARCHAR(100),
    
    -- 存储信息
    file_url TEXT NOT NULL,
    thumbnail_url TEXT,
    
    -- 媒体元数据
    width INTEGER,
    height INTEGER,
    duration_seconds INTEGER,
    
    -- 排序和显示
    display_order INTEGER DEFAULT 0,
    
    -- 扩展元数据
    metadata JSONB DEFAULT '{}',
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT moment_media_file_size_positive CHECK (file_size > 0),
    CONSTRAINT moment_media_dimensions_positive CHECK (
        (width IS NULL OR width > 0) AND (height IS NULL OR height > 0)
    ),
    CONSTRAINT moment_media_duration_positive CHECK (duration_seconds IS NULL OR duration_seconds > 0),
    CONSTRAINT moment_media_display_order_non_negative CHECK (display_order >= 0)
);

-- 评论表
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    comment_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(),
    
    -- 评论关联
    moment_id BIGINT NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 评论内容
    content TEXT NOT NULL,
    
    -- 回复关系
    parent_comment_id BIGINT REFERENCES comments(id) ON DELETE CASCADE,
    reply_to_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    
    -- 评论统计
    like_count INTEGER DEFAULT 0,
    reply_count INTEGER DEFAULT 0,
    
    -- 评论状态
    is_deleted BOOLEAN DEFAULT false,
    is_pinned BOOLEAN DEFAULT false,
    
    -- 扩展数据
    metadata JSONB DEFAULT '{}',
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    CONSTRAINT comments_content_not_empty CHECK (char_length(content) > 0),
    CONSTRAINT comments_like_count_non_negative CHECK (like_count >= 0),
    CONSTRAINT comments_reply_count_non_negative CHECK (reply_count >= 0),
    CONSTRAINT comments_no_self_reply CHECK (id != parent_comment_id)
);

-- 点赞表（瞬间和评论通用）
CREATE TABLE likes (
    id BIGSERIAL PRIMARY KEY,
    
    -- 点赞用户
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 点赞目标（瞬间或评论）
    target_type VARCHAR(20) NOT NULL, -- 'moment', 'comment'
    target_id BIGINT NOT NULL,
    
    -- 点赞类型
    like_type VARCHAR(20) DEFAULT 'like', -- 'like', 'love', 'laugh', 'angry', 'sad', 'wow'
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT likes_target_type_check CHECK (target_type IN ('moment', 'comment')),
    
    -- 唯一约束（每个用户对每个目标只能点赞一次）
    UNIQUE(user_id, target_type, target_id)
);

-- 分享表
CREATE TABLE shares (
    id BIGSERIAL PRIMARY KEY,
    
    -- 分享用户
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- 分享的瞬间
    moment_id BIGINT NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
    
    -- 分享信息
    share_comment TEXT,
    share_type VARCHAR(20) DEFAULT 'repost', -- 'repost', 'quote', 'forward'
    
    -- 分享目标
    target_type VARCHAR(20) DEFAULT 'timeline', -- 'timeline', 'group', 'private'
    target_group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束
    CONSTRAINT shares_target_type_check CHECK (target_type IN ('timeline', 'group', 'private'))
);

-- ============================================================================
-- 索引创建
-- ============================================================================

-- 好友关系表索引
CREATE INDEX idx_friendships_user_id ON friendships(user_id);
CREATE INDEX idx_friendships_friend_id ON friendships(friend_id);
CREATE INDEX idx_friendships_status ON friendships(status);
CREATE INDEX idx_friendships_created_at ON friendships(created_at);
CREATE INDEX idx_friendships_last_interaction ON friendships(last_interaction_at);

-- 用户屏蔽表索引
CREATE INDEX idx_user_blocks_blocker_id ON user_blocks(blocker_id);
CREATE INDEX idx_user_blocks_blocked_id ON user_blocks(blocked_id);
CREATE INDEX idx_user_blocks_expires_at ON user_blocks(expires_at);

-- 瞬间表索引
CREATE INDEX idx_moments_moment_id ON moments(moment_id);
CREATE INDEX idx_moments_user_id ON moments(user_id);
CREATE INDEX idx_moments_visibility ON moments(visibility);
CREATE INDEX idx_moments_target_group_id ON moments(target_group_id);
CREATE INDEX idx_moments_created_at ON moments(created_at);
CREATE INDEX idx_moments_is_deleted ON moments(is_deleted);
CREATE INDEX idx_moments_location ON moments USING GIST(location) WHERE location IS NOT NULL;
CREATE INDEX idx_moments_tags ON moments USING GIN(tags);

-- 瞬间媒体表索引
CREATE INDEX idx_moment_media_moment_id ON moment_media(moment_id);
CREATE INDEX idx_moment_media_type ON moment_media(media_type);
CREATE INDEX idx_moment_media_display_order ON moment_media(display_order);

-- 评论表索引
CREATE INDEX idx_comments_comment_id ON comments(comment_id);
CREATE INDEX idx_comments_moment_id ON comments(moment_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_parent_comment_id ON comments(parent_comment_id);
CREATE INDEX idx_comments_created_at ON comments(created_at);
CREATE INDEX idx_comments_is_deleted ON comments(is_deleted);

-- 点赞表索引
CREATE INDEX idx_likes_user_id ON likes(user_id);
CREATE INDEX idx_likes_target ON likes(target_type, target_id);
CREATE INDEX idx_likes_created_at ON likes(created_at);

-- 分享表索引
CREATE INDEX idx_shares_user_id ON shares(user_id);
CREATE INDEX idx_shares_moment_id ON shares(moment_id);
CREATE INDEX idx_shares_target_type ON shares(target_type);
CREATE INDEX idx_shares_target_group_id ON shares(target_group_id);
CREATE INDEX idx_shares_created_at ON shares(created_at);

-- ============================================================================
-- 触发器创建
-- ============================================================================

-- 好友关系表更新时间戳触发器
CREATE TRIGGER friendships_updated_at_trigger
    BEFORE UPDATE ON friendships
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 瞬间表更新时间戳触发器
CREATE TRIGGER moments_updated_at_trigger
    BEFORE UPDATE ON moments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 评论表更新时间戳触发器
CREATE TRIGGER comments_updated_at_trigger
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 社交功能审计触发器
CREATE TRIGGER moments_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON moments
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();

CREATE TRIGGER comments_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION audit_trigger_function();

-- ============================================================================
-- 统计维护函数和触发器
-- ============================================================================

-- 更新瞬间统计函数
CREATE OR REPLACE FUNCTION update_moment_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'likes' THEN
        IF TG_OP = 'INSERT' AND NEW.target_type = 'moment' THEN
            UPDATE moments SET like_count = like_count + 1 WHERE id = NEW.target_id;
        ELSIF TG_OP = 'DELETE' AND OLD.target_type = 'moment' THEN
            UPDATE moments SET like_count = like_count - 1 WHERE id = OLD.target_id;
        END IF;
    ELSIF TG_TABLE_NAME = 'comments' THEN
        IF TG_OP = 'INSERT' THEN
            UPDATE moments SET comment_count = comment_count + 1 WHERE id = NEW.moment_id;
        ELSIF TG_OP = 'DELETE' THEN
            UPDATE moments SET comment_count = comment_count - 1 WHERE id = OLD.moment_id;
        END IF;
    ELSIF TG_TABLE_NAME = 'shares' THEN
        IF TG_OP = 'INSERT' THEN
            UPDATE moments SET share_count = share_count + 1 WHERE id = NEW.moment_id;
        ELSIF TG_OP = 'DELETE' THEN
            UPDATE moments SET share_count = share_count - 1 WHERE id = OLD.moment_id;
        END IF;
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- 瞬间统计维护触发器
CREATE TRIGGER moment_like_stats_trigger
    AFTER INSERT OR DELETE ON likes
    FOR EACH ROW
    EXECUTE FUNCTION update_moment_stats();

CREATE TRIGGER moment_comment_stats_trigger
    AFTER INSERT OR DELETE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_moment_stats();

CREATE TRIGGER moment_share_stats_trigger
    AFTER INSERT OR DELETE ON shares
    FOR EACH ROW
    EXECUTE FUNCTION update_moment_stats();

-- 更新评论统计函数
CREATE OR REPLACE FUNCTION update_comment_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'likes' THEN
        IF TG_OP = 'INSERT' AND NEW.target_type = 'comment' THEN
            UPDATE comments SET like_count = like_count + 1 WHERE id = NEW.target_id;
        ELSIF TG_OP = 'DELETE' AND OLD.target_type = 'comment' THEN
            UPDATE comments SET like_count = like_count - 1 WHERE id = OLD.target_id;
        END IF;
    ELSIF TG_TABLE_NAME = 'comments' THEN
        IF TG_OP = 'INSERT' AND NEW.parent_comment_id IS NOT NULL THEN
            UPDATE comments SET reply_count = reply_count + 1 WHERE id = NEW.parent_comment_id;
        ELSIF TG_OP = 'DELETE' AND OLD.parent_comment_id IS NOT NULL THEN
            UPDATE comments SET reply_count = reply_count - 1 WHERE id = OLD.parent_comment_id;
        END IF;
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- 评论统计维护触发器
CREATE TRIGGER comment_like_stats_trigger
    AFTER INSERT OR DELETE ON likes
    FOR EACH ROW
    EXECUTE FUNCTION update_comment_stats();

CREATE TRIGGER comment_reply_stats_trigger
    AFTER INSERT OR DELETE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_comment_stats();
