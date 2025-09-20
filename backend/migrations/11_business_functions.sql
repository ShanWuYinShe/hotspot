-- ============================================================================
-- 13. 业务函数
-- ============================================================================
-- 
-- 功能职责：
-- - 复杂业务逻辑封装
-- - 数据查询优化
-- - 统计分析功能
-- - 地理位置计算
-- - 权限检查函数
--
-- 依赖关系：
-- - 依赖：所有前置模块（01-12）
--
-- ============================================================================

-- ============================================================================
-- 用户相关业务函数
-- ============================================================================

-- 检查用户是否存在且活跃
CREATE OR REPLACE FUNCTION is_user_active(user_id_param BIGINT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM users 
        WHERE id = user_id_param 
        AND deleted_at IS NULL
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- 获取用户基本信息
CREATE OR REPLACE FUNCTION get_user_info(user_id_param BIGINT)
RETURNS TABLE(
    id BIGINT,
    username VARCHAR(50),
    display_name VARCHAR(100),
    avatar_url TEXT,
    user_type INTEGER,
    created_at TIMESTAMP WITH TIME ZONE,
    last_active_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        u.id,
        u.nickname,
        u.nickname,
        u.avatar_url,
        get_user_type(u.id),
        u.created_at,
        u.updated_at
    FROM users u
    WHERE u.id = user_id_param AND u.deleted_at IS NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- 搜索用户
CREATE OR REPLACE FUNCTION search_users(
    search_term TEXT,
    limit_count INTEGER DEFAULT 20,
    offset_count INTEGER DEFAULT 0
)
RETURNS TABLE(
    id BIGINT,
    username VARCHAR(50),
    display_name VARCHAR(100),
    avatar_url TEXT,
    user_type INTEGER,
    similarity REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        u.id,
        u.nickname,
        u.nickname,
        u.avatar_url,
        get_user_type(u.id),
        GREATEST(
            similarity(u.nickname, search_term),
            similarity(COALESCE(u.bio, ''), search_term)
        ) as similarity
    FROM users u
    WHERE u.deleted_at IS NULL
    AND (
        u.nickname ILIKE '%' || search_term || '%' OR
        u.bio ILIKE '%' || search_term || '%' OR
        to_tsvector('simple', COALESCE(u.nickname, '') || ' ' || COALESCE(u.bio, '')) @@ plainto_tsquery('simple', search_term)
    )
    ORDER BY similarity DESC, u.created_at DESC
    LIMIT limit_count OFFSET offset_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- 获取用户统计信息
CREATE OR REPLACE FUNCTION get_user_statistics(user_id_param BIGINT)
RETURNS TABLE(
    groups_count INTEGER,
    messages_count INTEGER,
    moments_count INTEGER,
    friends_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*)::INTEGER FROM group_members gm 
         WHERE gm.user_id = user_id_param AND gm.status = 'active') as groups_count,
        (SELECT COUNT(*)::INTEGER FROM messages m 
         WHERE m.sender_id = user_id_param AND m.deleted_at IS NULL) as messages_count,
        (SELECT COUNT(*)::INTEGER FROM moments mo 
         WHERE mo.user_id = user_id_param AND mo.deleted_at IS NULL) as moments_count,
        (SELECT COUNT(*)::INTEGER FROM group_members gm1
         JOIN group_members gm2 ON gm1.group_id = gm2.group_id
         JOIN groups g ON g.id = gm1.group_id
         WHERE gm1.user_id = user_id_param 
         AND gm2.user_id != user_id_param
         AND g.group_type = 'private_chat'
         AND gm1.status = 'active' 
         AND gm2.status = 'active') as friends_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- 群组相关业务函数
-- ============================================================================

-- 检查用户是否是群组成员
CREATE OR REPLACE FUNCTION is_group_member(group_id_param BIGINT, user_id_param BIGINT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM group_members gm
        WHERE gm.group_id = group_id_param 
        AND gm.user_id = user_id_param 
        AND gm.status = 'active'
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- 检查用户在群组中的角色
CREATE OR REPLACE FUNCTION get_user_role_in_group(group_id_param BIGINT, user_id_param BIGINT)
RETURNS member_role_enum AS $$
DECLARE
    user_role member_role_enum;
BEGIN
    SELECT role INTO user_role
    FROM group_members
    WHERE group_id = group_id_param 
    AND user_id = user_id_param 
    AND status = 'active';
    
    RETURN user_role;
END;
$$ LANGUAGE plpgsql STABLE;

-- 检查用户是否有群组管理权限
CREATE OR REPLACE FUNCTION has_group_admin_permission(group_id_param BIGINT, user_id_param BIGINT)
RETURNS BOOLEAN AS $$
DECLARE
    user_role member_role_enum;
    is_owner BOOLEAN;
BEGIN
    -- 检查是否是群组拥有者
    SELECT (owner_id = user_id_param) INTO is_owner
    FROM groups
    WHERE id = group_id_param;
    
    IF is_owner THEN
        RETURN TRUE;
    END IF;
    
    -- 检查是否是管理员
    user_role := get_user_role_in_group(group_id_param, user_id_param);
    RETURN user_role = 'admin';
END;
$$ LANGUAGE plpgsql STABLE;

-- 搜索群组
CREATE OR REPLACE FUNCTION search_groups(
    search_term TEXT,
    user_id_param BIGINT DEFAULT NULL,
    limit_count INTEGER DEFAULT 20,
    offset_count INTEGER DEFAULT 0
)
RETURNS TABLE(
    id BIGINT,
    name VARCHAR(100),
    description TEXT,
    group_type group_type_enum,
    visibility visibility_enum,
    member_count INTEGER,
    avatar_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    similarity REAL,
    is_member BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        g.id,
        g.name,
        g.description,
        g.group_type,
        g.visibility,
        g.member_count,
        g.avatar_url,
        g.created_at,
        GREATEST(
            similarity(g.name, search_term),
            similarity(COALESCE(g.description, ''), search_term)
        ) as similarity,
        CASE 
            WHEN user_id_param IS NOT NULL THEN is_group_member(g.id, user_id_param)
            ELSE FALSE
        END as is_member
    FROM groups g
    WHERE g.deleted_at IS NULL
    AND g.visibility = 'public'
    AND (
        g.name ILIKE '%' || search_term || '%' OR
        g.description ILIKE '%' || search_term || '%' OR
        to_tsvector('simple', COALESCE(g.name, '') || ' ' || COALESCE(g.description, '')) @@ plainto_tsquery('simple', search_term)
    )
    ORDER BY similarity DESC, g.member_count DESC, g.created_at DESC
    LIMIT limit_count OFFSET offset_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- 获取用户的群组列表
CREATE OR REPLACE FUNCTION get_user_groups(
    user_id_param BIGINT,
    group_type_filter group_type_enum DEFAULT NULL,
    limit_count INTEGER DEFAULT 50,
    offset_count INTEGER DEFAULT 0
)
RETURNS TABLE(
    id BIGINT,
    name VARCHAR(100),
    group_type group_type_enum,
    visibility visibility_enum,
    member_count INTEGER,
    avatar_url TEXT,
    last_message_at TIMESTAMP WITH TIME ZONE,
    unread_count INTEGER,
    user_role member_role_enum,
    joined_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        g.id,
        g.name,
        g.group_type,
        g.visibility,
        g.member_count,
        g.avatar_url,
        g.updated_at as last_message_at,
        COALESCE(
            (SELECT COUNT(*)::INTEGER 
             FROM messages m 
             LEFT JOIN user_message_read_status umrs ON umrs.user_id = user_id_param AND umrs.group_id = g.id
             WHERE m.group_id = g.id 
             AND m.deleted_at IS NULL
             AND (umrs.last_read_at IS NULL OR m.created_at > umrs.last_read_at)
            ), 0
        ) as unread_count,
        gm.role as user_role,
        gm.joined_at
    FROM groups g
    JOIN group_members gm ON g.id = gm.group_id
    WHERE gm.user_id = user_id_param 
    AND gm.status = 'active'
    AND g.deleted_at IS NULL
    AND (group_type_filter IS NULL OR g.group_type = group_type_filter)
    ORDER BY g.updated_at DESC, g.created_at DESC
    LIMIT limit_count OFFSET offset_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- 消息相关业务函数
-- ============================================================================

-- 获取群组消息列表（支持游标分页）
CREATE OR REPLACE FUNCTION get_group_messages(
    group_id_param BIGINT,
    user_id_param BIGINT,
    cursor_message_id BIGINT DEFAULT NULL,
    limit_count INTEGER DEFAULT 50,
    direction TEXT DEFAULT 'before' -- 'before' 或 'after'
)
RETURNS TABLE(
    id BIGINT,
    sender_id BIGINT,
    sender_username VARCHAR(50),
    sender_display_name VARCHAR(100),
    sender_avatar_url TEXT,
    message_type message_type_enum,
    content TEXT,
    reply_to_id BIGINT,
    reply_to_content TEXT,
    reply_to_sender_name VARCHAR(100),
    attachments JSON,
    reactions JSON,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    -- 检查用户是否有权限查看消息
    IF NOT is_group_member(group_id_param, user_id_param) THEN
        -- 检查是否是私聊群组
        IF NOT EXISTS (SELECT 1 FROM groups WHERE id = group_id_param AND group_type = 'private_chat') THEN
            RAISE EXCEPTION '无权限查看该群组消息';
        END IF;
    END IF;
    
    RETURN QUERY
    SELECT 
        m.id,
        m.sender_id,
        u.username as sender_username,
        u.display_name as sender_display_name,
        u.avatar_url as sender_avatar_url,
        m.message_type,
        m.content,
        m.reply_to_id,
        rm.content as reply_to_content,
        ru.display_name as reply_to_sender_name,
        COALESCE(
            (SELECT json_agg(
                json_build_object(
                    'id', ma.id,
                    'media_type', ma.media_type,
                    'file_url', ma.file_url,
                    'file_name', ma.file_name,
                    'file_size', ma.file_size,
                    'thumbnail_url', ma.thumbnail_url
                )
             ) FROM message_attachments ma WHERE ma.message_id = m.id),
            '[]'::json
        ) as attachments,
        COALESCE(
            (SELECT json_agg(
                json_build_object(
                    'reaction_type', mr.reaction_type,
                    'count', COUNT(*),
                    'users', json_agg(
                        json_build_object(
                            'user_id', mr.user_id,
                            'username', mru.username
                        )
                    )
                )
             ) 
             FROM message_reactions mr
             JOIN users mru ON mru.id = mr.user_id
             WHERE mr.message_id = m.id
             GROUP BY mr.reaction_type),
            '[]'::json
        ) as reactions,
        m.created_at,
        m.updated_at
    FROM messages m
    JOIN users u ON u.id = m.sender_id
    LEFT JOIN messages rm ON rm.id = m.reply_to_id
    LEFT JOIN users ru ON ru.id = rm.sender_id
    WHERE m.group_id = group_id_param 
    AND m.deleted_at IS NULL
    AND (
        cursor_message_id IS NULL OR
        (direction = 'before' AND m.id < cursor_message_id) OR
        (direction = 'after' AND m.id > cursor_message_id)
    )
    ORDER BY 
        CASE WHEN direction = 'before' THEN m.created_at END DESC,
        CASE WHEN direction = 'after' THEN m.created_at END ASC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- 更新用户消息已读状态
CREATE OR REPLACE FUNCTION update_message_read_status(
    user_id_param BIGINT,
    group_id_param BIGINT,
    last_message_id BIGINT DEFAULT NULL
)
RETURNS VOID AS $$
DECLARE
    last_message_time TIMESTAMP WITH TIME ZONE;
BEGIN
    -- 检查用户是否有权限
    IF NOT is_group_member(group_id_param, user_id_param) THEN
        RAISE EXCEPTION '无权限更新该群组的已读状态';
    END IF;
    
    -- 获取最后一条消息的时间
    IF last_message_id IS NOT NULL THEN
        SELECT created_at INTO last_message_time
        FROM messages
        WHERE id = last_message_id AND group_id = group_id_param;
    ELSE
        SELECT MAX(created_at) INTO last_message_time
        FROM messages
        WHERE group_id = group_id_param AND deleted_at IS NULL;
    END IF;
    
    -- 更新或插入已读状态
    INSERT INTO user_message_read_status (user_id, group_id, last_read_at, updated_at)
    VALUES (user_id_param, group_id_param, COALESCE(last_message_time, CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)
    ON CONFLICT (user_id, group_id)
    DO UPDATE SET 
        last_read_at = COALESCE(last_message_time, CURRENT_TIMESTAMP),
        updated_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 动态相关业务函数
-- ============================================================================

-- 获取动态列表（支持多种排序方式）
CREATE OR REPLACE FUNCTION get_moments_feed(
    user_id_param BIGINT DEFAULT NULL,
    sort_by TEXT DEFAULT 'time', -- 'time', 'hot', 'nearby'
    user_location GEOMETRY DEFAULT NULL,
    max_distance_km REAL DEFAULT 50.0,
    limit_count INTEGER DEFAULT 20,
    offset_count INTEGER DEFAULT 0
)
RETURNS TABLE(
    id BIGINT,
    user_id BIGINT,
    username VARCHAR(50),
    display_name VARCHAR(100),
    avatar_url TEXT,
    content TEXT,
    visibility visibility_enum,
    location GEOMETRY,
    location_name VARCHAR(200),
    like_count INTEGER,
    comment_count INTEGER,
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE,
    is_liked BOOLEAN,
    distance_km REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.user_id,
        u.username,
        u.display_name,
        u.avatar_url,
        m.content,
        m.visibility,
        m.location,
        m.location_name,
        m.like_count,
        m.comment_count,
        COALESCE(
            (SELECT array_agg(mt.tag) FROM moment_tags mt WHERE mt.moment_id = m.id),
            ARRAY[]::TEXT[]
        ) as tags,
        m.created_at,
        CASE 
            WHEN user_id_param IS NOT NULL THEN 
                EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.id AND ml.user_id = user_id_param)
            ELSE FALSE
        END as is_liked,
        CASE 
            WHEN user_location IS NOT NULL AND m.location IS NOT NULL THEN
                ST_Distance(ST_Transform(user_location, 3857), ST_Transform(m.location, 3857)) / 1000.0
            ELSE NULL
        END as distance_km
    FROM moments m
    JOIN users u ON u.id = m.user_id
    WHERE m.deleted_at IS NULL
    AND m.visibility = 'public'
    AND (
        sort_by != 'nearby' OR 
        user_location IS NULL OR 
        m.location IS NULL OR
        ST_DWithin(ST_Transform(user_location, 3857), ST_Transform(m.location, 3857), max_distance_km * 1000)
    )
    ORDER BY 
        CASE WHEN sort_by = 'time' THEN m.created_at END DESC,
        CASE WHEN sort_by = 'hot' THEN (m.like_count * 2 + m.comment_count) END DESC,
        CASE WHEN sort_by = 'nearby' AND user_location IS NOT NULL AND m.location IS NOT NULL THEN
            ST_Distance(ST_Transform(user_location, 3857), ST_Transform(m.location, 3857))
        END ASC,
        m.created_at DESC
    LIMIT limit_count OFFSET offset_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- 获取动态详情（包含评论）
CREATE OR REPLACE FUNCTION get_moment_detail(
    moment_id_param BIGINT,
    user_id_param BIGINT DEFAULT NULL
)
RETURNS TABLE(
    id BIGINT,
    user_id BIGINT,
    username VARCHAR(50),
    display_name VARCHAR(100),
    avatar_url TEXT,
    content TEXT,
    visibility visibility_enum,
    location GEOMETRY,
    location_name VARCHAR(200),
    like_count INTEGER,
    comment_count INTEGER,
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE,
    is_liked BOOLEAN,
    comments JSON
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.user_id,
        u.username,
        u.display_name,
        u.avatar_url,
        m.content,
        m.visibility,
        m.location,
        m.location_name,
        m.like_count,
        m.comment_count,
        COALESCE(
            (SELECT array_agg(mt.tag) FROM moment_tags mt WHERE mt.moment_id = m.id),
            ARRAY[]::TEXT[]
        ) as tags,
        m.created_at,
        CASE 
            WHEN user_id_param IS NOT NULL THEN 
                EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.id AND ml.user_id = user_id_param)
            ELSE FALSE
        END as is_liked,
        COALESCE(
            (SELECT json_agg(
                json_build_object(
                    'id', c.id,
                    'user_id', c.user_id,
                    'username', cu.username,
                    'display_name', cu.display_name,
                    'avatar_url', cu.avatar_url,
                    'content', c.content,
                    'parent_id', c.parent_id,
                    'created_at', c.created_at
                ) ORDER BY c.created_at
             ) 
             FROM comments c
             JOIN users cu ON cu.id = c.user_id
             WHERE c.moment_id = m.id AND c.deleted_at IS NULL),
            '[]'::json
        ) as comments
    FROM moments m
    JOIN users u ON u.id = m.user_id
    WHERE m.id = moment_id_param 
    AND m.deleted_at IS NULL
    AND (m.visibility = 'public' OR (user_id_param IS NOT NULL AND m.user_id = user_id_param));
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- 地理位置相关函数
-- ============================================================================

-- 计算两点之间的距离（公里）
CREATE OR REPLACE FUNCTION calculate_distance_km(
    point1 GEOMETRY,
    point2 GEOMETRY
)
RETURNS REAL AS $$
BEGIN
    IF point1 IS NULL OR point2 IS NULL THEN
        RETURN NULL;
    END IF;
    
    RETURN ST_Distance(ST_Transform(point1, 3857), ST_Transform(point2, 3857)) / 1000.0;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 查找附近的用户
CREATE OR REPLACE FUNCTION find_nearby_users(
    center_location GEOMETRY,
    radius_km REAL DEFAULT 10.0,
    limit_count INTEGER DEFAULT 50,
    exclude_user_id BIGINT DEFAULT NULL
)
RETURNS TABLE(
    id BIGINT,
    username VARCHAR(50),
    display_name VARCHAR(100),
    avatar_url TEXT,
    last_location GEOMETRY,
    distance_km REAL,
    last_active_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        u.id,
        u.username,
        u.display_name,
        u.avatar_url,
        u.last_location,
        calculate_distance_km(center_location, u.last_location) as distance_km,
        u.last_active_at
    FROM users u
    WHERE u.deleted_at IS NULL
    AND u.last_location IS NOT NULL
    AND (exclude_user_id IS NULL OR u.id != exclude_user_id)
    AND ST_DWithin(ST_Transform(center_location, 3857), ST_Transform(u.last_location, 3857), radius_km * 1000)
    ORDER BY ST_Distance(ST_Transform(center_location, 3857), ST_Transform(u.last_location, 3857))
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- 查找附近的群组
CREATE OR REPLACE FUNCTION find_nearby_groups(
    center_location GEOMETRY,
    radius_km REAL DEFAULT 10.0,
    limit_count INTEGER DEFAULT 50
)
RETURNS TABLE(
    id BIGINT,
    name VARCHAR(100),
    description TEXT,
    group_type group_type_enum,
    member_count INTEGER,
    avatar_url TEXT,
    location GEOMETRY,
    distance_km REAL,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        g.id,
        g.name,
        g.description,
        g.group_type,
        g.member_count,
        g.avatar_url,
        g.location,
        calculate_distance_km(center_location, g.location) as distance_km,
        g.created_at
    FROM groups g
    WHERE g.deleted_at IS NULL
    AND g.visibility = 'public'
    AND g.location IS NOT NULL
    AND ST_DWithin(ST_Transform(center_location, 3857), ST_Transform(g.location, 3857), radius_km * 1000)
    ORDER BY ST_Distance(ST_Transform(center_location, 3857), ST_Transform(g.location, 3857))
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- 统计分析函数
-- ============================================================================

-- 获取系统统计信息
CREATE OR REPLACE FUNCTION get_system_statistics()
RETURNS TABLE(
    total_users INTEGER,
    active_users_today INTEGER,
    active_users_week INTEGER,
    total_groups INTEGER,
    total_messages INTEGER,
    total_moments INTEGER,
    messages_today INTEGER,
    moments_today INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*)::INTEGER FROM users WHERE deleted_at IS NULL) as total_users,
        (SELECT COUNT(*)::INTEGER FROM users 
         WHERE deleted_at IS NULL 
         AND last_active_at >= CURRENT_DATE) as active_users_today,
        (SELECT COUNT(*)::INTEGER FROM users 
         WHERE deleted_at IS NULL 
         AND last_active_at >= CURRENT_DATE - INTERVAL '7 days') as active_users_week,
        (SELECT COUNT(*)::INTEGER FROM groups WHERE deleted_at IS NULL) as total_groups,
        (SELECT COUNT(*)::INTEGER FROM messages WHERE deleted_at IS NULL) as total_messages,
        (SELECT COUNT(*)::INTEGER FROM moments WHERE deleted_at IS NULL) as total_moments,
        (SELECT COUNT(*)::INTEGER FROM messages 
         WHERE deleted_at IS NULL 
         AND created_at >= CURRENT_DATE) as messages_today,
        (SELECT COUNT(*)::INTEGER FROM moments 
         WHERE deleted_at IS NULL 
         AND created_at >= CURRENT_DATE) as moments_today;
END;
$$ LANGUAGE plpgsql STABLE;

-- 获取热门标签
CREATE OR REPLACE FUNCTION get_trending_tags(
    days_back INTEGER DEFAULT 7,
    limit_count INTEGER DEFAULT 20
)
RETURNS TABLE(
    tag TEXT,
    usage_count INTEGER,
    trend_score REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        mt.tag,
        COUNT(*)::INTEGER as usage_count,
        (COUNT(*) * 1.0 / GREATEST(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - MIN(m.created_at))) / 86400, 1))::REAL as trend_score
    FROM moment_tags mt
    JOIN moments m ON m.id = mt.moment_id
    WHERE m.deleted_at IS NULL
    AND m.visibility = 'public'
    AND m.created_at >= CURRENT_DATE - (days_back || ' days')::INTERVAL
    GROUP BY mt.tag
    HAVING COUNT(*) >= 2
    ORDER BY trend_score DESC, usage_count DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- 权限检查函数
-- ============================================================================

-- 检查用户是否被屏蔽
CREATE OR REPLACE FUNCTION is_user_blocked(
    blocker_id_param BIGINT,
    blocked_id_param BIGINT
)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM blocks
        WHERE blocker_id = blocker_id_param 
        AND blocked_id = blocked_id_param
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- 检查用户是否可以发送消息给另一个用户
CREATE OR REPLACE FUNCTION can_send_message_to_user(
    sender_id_param BIGINT,
    recipient_id_param BIGINT
)
RETURNS BOOLEAN AS $$
BEGIN
    -- 检查是否被屏蔽
    IF is_user_blocked(recipient_id_param, sender_id_param) THEN
        RETURN FALSE;
    END IF;
    
    -- 检查发送者是否活跃
    IF NOT is_user_active(sender_id_param) THEN
        RETURN FALSE;
    END IF;
    
    -- 检查接收者是否活跃
    IF NOT is_user_active(recipient_id_param) THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql STABLE;

-- 添加函数注释
COMMENT ON FUNCTION is_user_active(BIGINT) IS '检查用户是否存在且活跃';
COMMENT ON FUNCTION get_user_info(BIGINT) IS '获取用户基本信息';
COMMENT ON FUNCTION search_users(TEXT, INTEGER, INTEGER) IS '搜索用户';
COMMENT ON FUNCTION get_user_statistics(BIGINT) IS '获取用户统计信息';
COMMENT ON FUNCTION is_group_member(BIGINT, BIGINT) IS '检查用户是否是群组成员';
COMMENT ON FUNCTION get_user_role_in_group(BIGINT, BIGINT) IS '获取用户在群组中的角色';
COMMENT ON FUNCTION has_group_admin_permission(BIGINT, BIGINT) IS '检查用户是否有群组管理权限';
COMMENT ON FUNCTION search_groups(TEXT, BIGINT, INTEGER, INTEGER) IS '搜索群组';
COMMENT ON FUNCTION get_user_groups(BIGINT, group_type_enum, INTEGER, INTEGER) IS '获取用户的群组列表';
COMMENT ON FUNCTION get_group_messages(BIGINT, BIGINT, BIGINT, INTEGER, TEXT) IS '获取群组消息列表';
COMMENT ON FUNCTION update_message_read_status(BIGINT, BIGINT, BIGINT) IS '更新用户消息已读状态';
COMMENT ON FUNCTION get_moments_feed(BIGINT, TEXT, GEOMETRY, REAL, INTEGER, INTEGER) IS '获取动态列表';
COMMENT ON FUNCTION get_moment_detail(BIGINT, BIGINT) IS '获取动态详情';
COMMENT ON FUNCTION calculate_distance_km(GEOMETRY, GEOMETRY) IS '计算两点之间的距离';
COMMENT ON FUNCTION find_nearby_users(GEOMETRY, REAL, INTEGER, BIGINT) IS '查找附近的用户';
COMMENT ON FUNCTION find_nearby_groups(GEOMETRY, REAL, INTEGER) IS '查找附近的群组';
COMMENT ON FUNCTION get_system_statistics() IS '获取系统统计信息';
COMMENT ON FUNCTION get_trending_tags(INTEGER, INTEGER) IS '获取热门标签';
COMMENT ON FUNCTION is_user_blocked(BIGINT, BIGINT) IS '检查用户是否被屏蔽';
COMMENT ON FUNCTION can_send_message_to_user(BIGINT, BIGINT) IS '检查用户是否可以发送消息给另一个用户';

COMMENT ON SCHEMA public IS '包含完整业务逻辑函数的主要模式，支持复杂查询和业务规则';
