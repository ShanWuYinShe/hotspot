-- ============================================================================
-- 07. 生活瞬间系统表
-- ============================================================================
-- 
-- 功能职责：
-- - 定义生活瞬间相关的所有表结构
-- - 支持地理位置瞬间功能
-- - 提供点赞、评论、标签等互动功能
-- - 支持可见性控制和内容管理
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql, 04_users.sql
--
-- ============================================================================

-- 生活瞬间表
CREATE TABLE moments (
    id BIGSERIAL PRIMARY KEY,                   -- 瞬间ID（自增主键，用于高性能分页）
    moment_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 瞬间唯一标识符（对外API使用，保护数据隐私）
    user_id BIGINT NOT NULL,                   -- 用户ID
    content TEXT NOT NULL,                     -- 瞬间内容/描述
    visibility visibility_enum NOT NULL DEFAULT 'public', -- 可见性
    -- 地理位置信息
    latitude DOUBLE PRECISION NOT NULL,        -- 纬度
    longitude DOUBLE PRECISION NOT NULL,       -- 经度
    location GEOGRAPHY(POINT, 4326) NOT NULL,  -- PostGIS地理位置点
    location_name TEXT NOT NULL,               -- 位置名称
    location_accuracy DOUBLE PRECISION,       -- 位置精度（米）
    -- 媒体和标签
    image_urls TEXT[] NOT NULL DEFAULT '{}',   -- 图片URL数组
    video_urls TEXT[] NOT NULL DEFAULT '{}',   -- 视频URL数组
    tags TEXT[] NOT NULL DEFAULT '{}',         -- 标签数组
    -- 统计信息
    likes_count INTEGER NOT NULL DEFAULT 0,    -- 点赞数
    comments_count INTEGER NOT NULL DEFAULT 0, -- 评论数
    views_count INTEGER NOT NULL DEFAULT 0,    -- 浏览数
    shares_count INTEGER NOT NULL DEFAULT 0,   -- 分享数
    -- 状态和时间
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    is_featured BOOLEAN NOT NULL DEFAULT false, -- 是否精选
    moderation_status moderation_status_enum NOT NULL DEFAULT 'pending', -- 审核状态
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 瞬间点赞表
CREATE TABLE moment_likes (
    id BIGSERIAL PRIMARY KEY,                   -- 点赞记录ID
    like_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 点赞唯一标识符
    moment_id TEXT NOT NULL,                   -- 瞬间ID
    user_id BIGINT NOT NULL,                   -- 用户ID
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(moment_id, user_id)                 -- 防止重复点赞
);

-- 瞬间评论表
CREATE TABLE moment_comments (
    id BIGSERIAL PRIMARY KEY,                   -- 评论ID（自增主键，用于高性能分页）
    comment_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 评论唯一标识符（对外API使用，保护数据隐私）
    moment_id TEXT NOT NULL,                   -- 瞬间ID
    user_id BIGINT NOT NULL,                   -- 评论用户ID
    parent_comment_id TEXT,                    -- 父评论ID（用于回复）
    content TEXT NOT NULL,                     -- 评论内容
    mentioned_users BIGINT[],                  -- 提及的用户ID列表
    likes_count INTEGER NOT NULL DEFAULT 0,    -- 评论点赞数
    replies_count INTEGER NOT NULL DEFAULT 0,  -- 回复数
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    moderation_status moderation_status_enum NOT NULL DEFAULT 'pending', -- 审核状态
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    deleted_at TIMESTAMP WITH TIME ZONE        -- 删除时间
);

-- 评论点赞表
CREATE TABLE comment_likes (
    id BIGSERIAL PRIMARY KEY,                   -- 点赞记录ID
    like_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 点赞唯一标识符
    comment_id TEXT NOT NULL,                  -- 评论ID
    user_id BIGINT NOT NULL,                   -- 用户ID
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(comment_id, user_id)                -- 防止重复点赞
);

-- 标签表（热门标签管理）
CREATE TABLE moment_tags (
    id BIGSERIAL PRIMARY KEY,                   -- 标签ID
    tag_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 标签唯一标识符
    name TEXT UNIQUE NOT NULL,                 -- 标签名称
    description TEXT,                          -- 标签描述
    usage_count INTEGER NOT NULL DEFAULT 0,    -- 使用次数
    is_trending BOOLEAN NOT NULL DEFAULT false, -- 是否热门
    is_official BOOLEAN NOT NULL DEFAULT false, -- 是否官方标签
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 瞬间标签关联表
CREATE TABLE moment_tag_relations (
    id BIGSERIAL PRIMARY KEY,                   -- 关联ID
    moment_id TEXT NOT NULL,                   -- 瞬间ID
    tag_id TEXT NOT NULL,                      -- 标签ID
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    
    UNIQUE(moment_id, tag_id)                  -- 防止重复关联
);

-- 瞬间浏览记录表（用于推荐算法）
CREATE TABLE moment_views (
    id BIGSERIAL PRIMARY KEY,                   -- 浏览记录ID
    moment_id TEXT NOT NULL,                   -- 瞬间ID
    user_id BIGINT,                            -- 用户ID（可为NULL，匿名浏览）
    ip_address INET,                           -- IP地址
    user_agent TEXT,                           -- 用户代理
    view_duration INTEGER,                     -- 浏览时长（秒）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() -- 浏览时间
);

-- 瞬间统计表（用于性能优化）
CREATE TABLE moment_statistics (
    id BIGSERIAL PRIMARY KEY,                   -- 统计ID
    moment_id TEXT UNIQUE NOT NULL,            -- 瞬间ID
    total_likes INTEGER NOT NULL DEFAULT 0,    -- 总点赞数
    total_comments INTEGER NOT NULL DEFAULT 0, -- 总评论数
    total_views INTEGER NOT NULL DEFAULT 0,    -- 总浏览数
    total_shares INTEGER NOT NULL DEFAULT 0,   -- 总分享数
    unique_viewers INTEGER NOT NULL DEFAULT 0, -- 独立浏览者数
    engagement_score DOUBLE PRECISION NOT NULL DEFAULT 0, -- 互动分数
    trending_score DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 热度分数
    last_calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 最后计算时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 添加外键约束
ALTER TABLE moments ADD CONSTRAINT fk_moments_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE moment_likes ADD CONSTRAINT fk_moment_likes_moment_id 
    FOREIGN KEY (moment_id) REFERENCES moments(moment_id) ON DELETE CASCADE;

ALTER TABLE moment_likes ADD CONSTRAINT fk_moment_likes_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE moment_comments ADD CONSTRAINT fk_moment_comments_moment_id 
    FOREIGN KEY (moment_id) REFERENCES moments(moment_id) ON DELETE CASCADE;

ALTER TABLE moment_comments ADD CONSTRAINT fk_moment_comments_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE moment_comments ADD CONSTRAINT fk_moment_comments_parent_comment_id 
    FOREIGN KEY (parent_comment_id) REFERENCES moment_comments(comment_id) ON DELETE CASCADE;

ALTER TABLE comment_likes ADD CONSTRAINT fk_comment_likes_comment_id 
    FOREIGN KEY (comment_id) REFERENCES moment_comments(comment_id) ON DELETE CASCADE;

ALTER TABLE comment_likes ADD CONSTRAINT fk_comment_likes_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE moment_tag_relations ADD CONSTRAINT fk_moment_tag_relations_moment_id 
    FOREIGN KEY (moment_id) REFERENCES moments(moment_id) ON DELETE CASCADE;

ALTER TABLE moment_tag_relations ADD CONSTRAINT fk_moment_tag_relations_tag_id 
    FOREIGN KEY (tag_id) REFERENCES moment_tags(tag_id) ON DELETE CASCADE;

ALTER TABLE moment_views ADD CONSTRAINT fk_moment_views_moment_id 
    FOREIGN KEY (moment_id) REFERENCES moments(moment_id) ON DELETE CASCADE;

ALTER TABLE moment_views ADD CONSTRAINT fk_moment_views_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE moment_statistics ADD CONSTRAINT fk_moment_statistics_moment_id 
    FOREIGN KEY (moment_id) REFERENCES moments(moment_id) ON DELETE CASCADE;

-- 添加检查约束
ALTER TABLE moments ADD CONSTRAINT check_moments_content_length 
    CHECK (length(content) >= 1 AND length(content) <= 2000);

ALTER TABLE moments ADD CONSTRAINT check_moments_location_coordinates 
    CHECK (latitude >= -90 AND latitude <= 90 AND longitude >= -180 AND longitude <= 180);

ALTER TABLE moments ADD CONSTRAINT check_moments_image_urls_count 
    CHECK (array_length(image_urls, 1) IS NULL OR array_length(image_urls, 1) <= 9);

ALTER TABLE moments ADD CONSTRAINT check_moments_video_urls_count 
    CHECK (array_length(video_urls, 1) IS NULL OR array_length(video_urls, 1) <= 3);

ALTER TABLE moments ADD CONSTRAINT check_moments_tags_count 
    CHECK (array_length(tags, 1) IS NULL OR array_length(tags, 1) <= 10);

ALTER TABLE moment_comments ADD CONSTRAINT check_moment_comments_content_length 
    CHECK (length(content) >= 1 AND length(content) <= 500);

ALTER TABLE moment_tags ADD CONSTRAINT check_moment_tags_name_length 
    CHECK (length(name) >= 1 AND length(name) <= 50);

ALTER TABLE moment_views ADD CONSTRAINT check_moment_views_duration 
    CHECK (view_duration IS NULL OR view_duration >= 0);

-- 添加表注释
COMMENT ON TABLE moments IS '生活瞬间表，存储用户发布的地理位置瞬间';
COMMENT ON TABLE moment_likes IS '瞬间点赞表，记录用户对瞬间的点赞';
COMMENT ON TABLE moment_comments IS '瞬间评论表，支持多级评论和回复';
COMMENT ON TABLE comment_likes IS '评论点赞表，记录用户对评论的点赞';
COMMENT ON TABLE moment_tags IS '标签表，管理瞬间标签和热门话题';
COMMENT ON TABLE moment_tag_relations IS '瞬间标签关联表，建立瞬间和标签的多对多关系';
COMMENT ON TABLE moment_views IS '瞬间浏览记录表，用于统计和推荐算法';
COMMENT ON TABLE moment_statistics IS '瞬间统计表，用于性能优化和热度计算';

-- 添加列注释
COMMENT ON COLUMN moments.moment_id IS '瞬间唯一标识符，对外API使用';
COMMENT ON COLUMN moments.location IS 'PostGIS地理位置点，用于地理查询和距离计算';
COMMENT ON COLUMN moments.visibility IS '可见性：public=公开, friends=好友可见, private=私有';
COMMENT ON COLUMN moments.moderation_status IS '审核状态：pending=待审核, approved=已通过, rejected=已拒绝, flagged=已标记';
COMMENT ON COLUMN moment_comments.mentioned_users IS '评论中提及的用户ID数组，用于@功能';
COMMENT ON COLUMN moment_statistics.engagement_score IS '互动分数，基于点赞、评论、分享等计算';
COMMENT ON COLUMN moment_statistics.trending_score IS '热度分数，用于热门瞬间排序';
