-- ============================================================================
-- 06. 统一消息系统表
-- ============================================================================
-- 
-- 功能职责：
-- - 实现统一消息架构（私聊和群聊消息统一存储）
-- - 支持多种消息类型（文本、媒体、位置、系统消息）
-- - 提供消息状态管理和已读回执
-- - 支持消息反应和回复功能
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql, 04_users.sql, 05_groups.sql
--
-- ============================================================================

-- 消息表（分区表，统一存储所有消息）
CREATE TABLE messages (
    id BIGSERIAL,                               -- 消息ID（自增主键，用于高性能分页）
    message_id TEXT NOT NULL DEFAULT generate_unique_id(), -- 消息唯一标识符（对外API使用，保护数据隐私）
    group_id TEXT NOT NULL,                    -- 群组ID（统一架构：私聊也通过群组管理）
    sender_id BIGINT NOT NULL,                 -- 发送者ID
    content TEXT NOT NULL,                     -- 消息内容
    message_type message_type_enum NOT NULL DEFAULT 'text', -- 消息类型
    -- 媒体相关字段
    media_type media_type_enum,                -- 媒体类型（仅媒体消息）
    media_url TEXT,                            -- 媒体文件URL
    media_size BIGINT,                         -- 媒体文件大小（字节）
    media_duration INTEGER,                    -- 媒体时长（秒，仅音频/视频）
    media_width INTEGER,                       -- 媒体宽度（像素，仅图片/视频）
    media_height INTEGER,                      -- 媒体高度（像素，仅图片/视频）
    thumbnail_url TEXT,                        -- 缩略图URL
    -- 位置相关字段
    latitude DOUBLE PRECISION,                 -- 位置纬度
    longitude DOUBLE PRECISION,                -- 位置经度
    location_name TEXT,                        -- 位置名称
    -- 回复和引用
    reply_to_message_id TEXT,                  -- 回复的消息ID
    forward_from_message_id TEXT,              -- 转发的原消息ID
    -- 提及和标签
    mentioned_users BIGINT[],                  -- 提及的用户ID列表
    hashtags TEXT[],                           -- 话题标签列表
    -- 状态和元数据
    is_edited BOOLEAN NOT NULL DEFAULT false, -- 是否已编辑
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    is_pinned BOOLEAN NOT NULL DEFAULT false,  -- 是否置顶
    metadata JSONB NOT NULL DEFAULT '{}',     -- 消息元数据（设备信息、编辑历史等）
    -- 时间字段
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    edited_at TIMESTAMP WITH TIME ZONE,       -- 编辑时间
    deleted_at TIMESTAMP WITH TIME ZONE,      -- 删除时间
    
    PRIMARY KEY (id, created_at)               -- 复合主键，支持分区
    -- 注意：分区表的唯一约束必须包含分区键，通过索引实现message_id唯一性
) PARTITION BY RANGE (created_at);

-- 消息状态表（已读回执、送达状态等）
CREATE TABLE message_status (
    id BIGSERIAL PRIMARY KEY,                   -- 状态记录ID
    message_id TEXT NOT NULL,                  -- 消息ID
    user_id BIGINT NOT NULL,                   -- 用户ID
    status TEXT NOT NULL,                      -- 状态：'sent'|'delivered'|'read'|'failed'
    status_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 状态时间
    device_info JSONB,                         -- 设备信息
    
    UNIQUE(message_id, user_id, status)        -- 防止重复状态记录
);

-- 消息反应表（点赞、表情等）
CREATE TABLE message_reactions (
    id BIGSERIAL PRIMARY KEY,                   -- 反应记录ID
    reaction_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 反应唯一标识符
    message_id TEXT NOT NULL,                  -- 消息ID
    user_id BIGINT NOT NULL,                   -- 用户ID
    reaction_type reaction_type_enum NOT NULL, -- 反应类型
    is_deleted BOOLEAN NOT NULL DEFAULT false, -- 软删除标记
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    deleted_at TIMESTAMP WITH TIME ZONE,       -- 删除时间
    
    UNIQUE(message_id, user_id, reaction_type) -- 防止重复反应
);

-- 消息编辑历史表
CREATE TABLE message_edit_history (
    id BIGSERIAL PRIMARY KEY,                   -- 编辑记录ID
    message_id TEXT NOT NULL,                  -- 消息ID
    old_content TEXT NOT NULL,                 -- 原内容
    new_content TEXT NOT NULL,                 -- 新内容
    edited_by BIGINT NOT NULL,                 -- 编辑者ID
    edit_reason TEXT,                          -- 编辑原因
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() -- 编辑时间
);

-- 消息附件表（支持多个附件）
CREATE TABLE message_attachments (
    id BIGSERIAL PRIMARY KEY,                   -- 附件ID
    attachment_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 附件唯一标识符
    message_id TEXT NOT NULL,                  -- 消息ID
    file_name TEXT NOT NULL,                   -- 文件名
    file_url TEXT NOT NULL,                    -- 文件URL
    file_type TEXT NOT NULL,                   -- 文件类型（MIME类型）
    file_size BIGINT NOT NULL,                 -- 文件大小（字节）
    thumbnail_url TEXT,                        -- 缩略图URL
    width INTEGER,                             -- 宽度（图片/视频）
    height INTEGER,                            -- 高度（图片/视频）
    duration INTEGER,                          -- 时长（音频/视频）
    metadata JSONB NOT NULL DEFAULT '{}',     -- 附件元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() -- 创建时间
);

-- 消息统计表（用于性能优化）
CREATE TABLE message_statistics (
    id BIGSERIAL PRIMARY KEY,                   -- 统计ID
    group_id TEXT NOT NULL,                    -- 群组ID
    date DATE NOT NULL,                        -- 统计日期
    total_messages INTEGER NOT NULL DEFAULT 0, -- 当日总消息数
    unique_senders INTEGER NOT NULL DEFAULT 0, -- 当日发言人数
    text_messages INTEGER NOT NULL DEFAULT 0,  -- 文本消息数
    media_messages INTEGER NOT NULL DEFAULT 0, -- 媒体消息数
    system_messages INTEGER NOT NULL DEFAULT 0, -- 系统消息数
    total_reactions INTEGER NOT NULL DEFAULT 0, -- 总反应数
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    
    UNIQUE(group_id, date)                     -- 每个群组每天一条记录
);

-- 添加外键约束
ALTER TABLE messages ADD CONSTRAINT fk_messages_group_id 
    FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE;

ALTER TABLE messages ADD CONSTRAINT fk_messages_sender_id 
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE;

-- 注意：分区表的外键约束已移除，使用应用层逻辑和触发器来维护引用完整性
-- ALTER TABLE messages ADD CONSTRAINT fk_messages_reply_to_message_id 
--     FOREIGN KEY (reply_to_message_id) REFERENCES messages(message_id) ON DELETE SET NULL;

-- ALTER TABLE message_status ADD CONSTRAINT fk_message_status_message_id 
--     FOREIGN KEY (message_id) REFERENCES messages(message_id) ON DELETE CASCADE;

ALTER TABLE message_status ADD CONSTRAINT fk_message_status_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ALTER TABLE message_reactions ADD CONSTRAINT fk_message_reactions_message_id 
--     FOREIGN KEY (message_id) REFERENCES messages(message_id) ON DELETE CASCADE;

ALTER TABLE message_reactions ADD CONSTRAINT fk_message_reactions_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ALTER TABLE message_edit_history ADD CONSTRAINT fk_message_edit_history_message_id 
--     FOREIGN KEY (message_id) REFERENCES messages(message_id) ON DELETE CASCADE;

ALTER TABLE message_edit_history ADD CONSTRAINT fk_message_edit_history_edited_by 
    FOREIGN KEY (edited_by) REFERENCES users(id) ON DELETE CASCADE;

-- ALTER TABLE message_attachments ADD CONSTRAINT fk_message_attachments_message_id 
--     FOREIGN KEY (message_id) REFERENCES messages(message_id) ON DELETE CASCADE;

ALTER TABLE message_statistics ADD CONSTRAINT fk_message_statistics_group_id 
    FOREIGN KEY (group_id) REFERENCES groups(group_id) ON DELETE CASCADE;

-- 添加检查约束
ALTER TABLE messages ADD CONSTRAINT check_messages_content_length 
    CHECK (length(content) >= 1 AND length(content) <= 10000);

ALTER TABLE messages ADD CONSTRAINT check_messages_location_coordinates 
    CHECK ((latitude IS NULL AND longitude IS NULL) OR 
           (latitude IS NOT NULL AND longitude IS NOT NULL AND 
            latitude >= -90 AND latitude <= 90 AND 
            longitude >= -180 AND longitude <= 180));

ALTER TABLE messages ADD CONSTRAINT check_messages_media_fields 
    CHECK ((message_type NOT IN ('image', 'video', 'audio', 'file') AND media_type IS NULL) OR 
           (message_type IN ('image', 'video', 'audio', 'file') AND media_type IS NOT NULL));

ALTER TABLE message_attachments ADD CONSTRAINT check_message_attachments_file_size 
    CHECK (file_size > 0 AND file_size <= 1073741824); -- 最大1GB

ALTER TABLE message_attachments ADD CONSTRAINT check_message_attachments_dimensions 
    CHECK ((width IS NULL AND height IS NULL) OR (width > 0 AND height > 0));

-- 添加表注释
COMMENT ON TABLE messages IS '统一消息表，存储所有类型的消息（私聊和群聊）';
COMMENT ON TABLE message_status IS '消息状态表，记录消息的送达和已读状态';
COMMENT ON TABLE message_reactions IS '消息反应表，记录用户对消息的反应（点赞、表情等）';
COMMENT ON TABLE message_edit_history IS '消息编辑历史表，记录消息的修改历史';
COMMENT ON TABLE message_attachments IS '消息附件表，支持消息的多个附件';
COMMENT ON TABLE message_statistics IS '消息统计表，用于性能优化和数据分析';

-- 添加列注释
COMMENT ON COLUMN messages.message_id IS '消息唯一标识符，对外API使用';
COMMENT ON COLUMN messages.group_id IS '群组ID，统一架构中私聊也通过群组管理';
COMMENT ON COLUMN messages.message_type IS '消息类型：text=文本, image=图片, video=视频, audio=音频, file=文件, location=位置, system=系统消息';
COMMENT ON COLUMN messages.mentioned_users IS '提及的用户ID数组，用于@功能';
COMMENT ON COLUMN messages.metadata IS '消息元数据，包含设备信息、编辑历史等';
COMMENT ON COLUMN message_status.status IS '消息状态：sent=已发送, delivered=已送达, read=已读, failed=发送失败';
COMMENT ON COLUMN message_reactions.reaction_type IS '反应类型：like=点赞, love=喜爱, laugh=大笑, angry=愤怒, sad=悲伤, wow=惊讶';
