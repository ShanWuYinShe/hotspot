-- ============================================================================
-- 12. 触发器创建
-- ============================================================================
-- 
-- 功能职责：
-- - 自动更新时间戳
-- - 数据完整性约束
-- - 业务逻辑自动化
-- - 审计日志记录
-- - 缓存失效通知
--
-- 依赖关系：
-- - 依赖：所有前置模块（01-11）
--
-- ============================================================================

-- ============================================================================
-- 通用触发器函数
-- ============================================================================

-- 自动更新 updated_at 字段的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 软删除触发器函数
CREATE OR REPLACE FUNCTION soft_delete_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- 设置删除时间而不是真正删除记录
    NEW.deleted_at = CURRENT_TIMESTAMP;
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 验证用户ID类型的触发器函数
CREATE OR REPLACE FUNCTION validate_user_id_type()
RETURNS TRIGGER AS $$
BEGIN
    -- 验证用户ID是否有效
    IF NOT is_valid_user_id(NEW.id) THEN
        RAISE EXCEPTION '无效的用户ID: %', NEW.id;
    END IF;
    
    -- 验证用户类型与ID编码是否匹配
    IF get_user_type(NEW.id) != NEW.user_type::INTEGER THEN
        RAISE EXCEPTION '用户ID类型不匹配: ID=%, 类型=%', NEW.id, NEW.user_type;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 用户相关触发器
-- ============================================================================

-- 用户表 updated_at 自动更新触发器
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 用户ID类型验证触发器
CREATE TRIGGER trigger_users_validate_id_type
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION validate_user_id_type();

-- 用户活动记录触发器函数
CREATE OR REPLACE FUNCTION log_user_activity()
RETURNS TRIGGER AS $$
BEGIN
    -- 记录用户登录活动
    IF TG_OP = 'UPDATE' AND OLD.last_active_at != NEW.last_active_at THEN
        INSERT INTO user_activity (user_id, activity_type, details, created_at)
        VALUES (NEW.id, 'login', '{"ip": "unknown", "user_agent": "unknown"}', NEW.last_active_at);
    END IF;
    
    -- 记录用户注册活动
    IF TG_OP = 'INSERT' THEN
        INSERT INTO user_activity (user_id, activity_type, details, created_at)
        VALUES (NEW.id, 'register', '{"registration_method": "unknown"}', NEW.created_at);
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 用户活动记录触发器
CREATE TRIGGER trigger_users_activity_log
    AFTER INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION log_user_activity();

-- 用户设置自动创建触发器函数
CREATE OR REPLACE FUNCTION create_user_settings()
RETURNS TRIGGER AS $$
BEGIN
    -- 为新用户创建默认设置
    INSERT INTO user_settings (user_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 用户设置自动创建触发器
CREATE TRIGGER trigger_users_create_settings
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION create_user_settings();

-- ============================================================================
-- 群组相关触发器
-- ============================================================================

-- 群组表 updated_at 自动更新触发器
CREATE TRIGGER trigger_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 群组成员数量更新触发器函数
CREATE OR REPLACE FUNCTION update_group_member_count()
RETURNS TRIGGER AS $$
DECLARE
    group_id_val BIGINT;
    new_count INTEGER;
BEGIN
    -- 确定要更新的群组ID
    IF TG_OP = 'DELETE' THEN
        group_id_val := OLD.group_id;
    ELSE
        group_id_val := NEW.group_id;
    END IF;
    
    -- 计算活跃成员数量
    SELECT COUNT(*) INTO new_count
    FROM group_members
    WHERE group_id = group_id_val AND status = 'active';
    
    -- 更新群组成员数量
    UPDATE groups
    SET member_count = new_count,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = group_id_val;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 群组成员数量更新触发器
CREATE TRIGGER trigger_group_members_count_update
    AFTER INSERT OR UPDATE OR DELETE ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION update_group_member_count();

-- 群组创建者自动加入触发器函数
CREATE OR REPLACE FUNCTION add_owner_to_group()
RETURNS TRIGGER AS $$
BEGIN
    -- 将群组创建者自动添加为管理员
    INSERT INTO group_members (group_id, user_id, role, status, joined_at)
    VALUES (NEW.id, NEW.owner_id, 'admin', 'active', NEW.created_at);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 群组创建者自动加入触发器
CREATE TRIGGER trigger_groups_add_owner
    AFTER INSERT ON groups
    FOR EACH ROW
    EXECUTE FUNCTION add_owner_to_group();

-- 群组邀请过期检查触发器函数
CREATE OR REPLACE FUNCTION check_invitation_expiry()
RETURNS TRIGGER AS $$
BEGIN
    -- 检查邀请是否已过期
    IF NEW.expires_at <= CURRENT_TIMESTAMP THEN
        RAISE EXCEPTION '邀请已过期';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 群组邀请过期检查触发器
CREATE TRIGGER trigger_group_invitations_expiry_check
    BEFORE INSERT OR UPDATE ON group_invitations
    FOR EACH ROW
    EXECUTE FUNCTION check_invitation_expiry();

-- ============================================================================
-- 消息相关触发器
-- ============================================================================

-- 消息表 updated_at 自动更新触发器
CREATE TRIGGER trigger_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 消息发送权限检查触发器函数
CREATE OR REPLACE FUNCTION check_message_send_permission()
RETURNS TRIGGER AS $$
DECLARE
    member_status member_status_enum;
    group_type_val group_type_enum;
BEGIN
    -- 检查发送者是否是群组成员
    SELECT gm.status, g.group_type INTO member_status, group_type_val
    FROM group_members gm
    JOIN groups g ON g.id = gm.group_id
    WHERE gm.group_id = NEW.group_id AND gm.user_id = NEW.sender_id;
    
    -- 如果不是成员，检查是否是私聊群组
    IF member_status IS NULL THEN
        IF group_type_val != 'private_chat' THEN
            RAISE EXCEPTION '用户不是群组成员，无法发送消息';
        END IF;
    ELSIF member_status != 'active' THEN
        RAISE EXCEPTION '用户状态异常，无法发送消息';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 消息发送权限检查触发器
CREATE TRIGGER trigger_messages_permission_check
    BEFORE INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION check_message_send_permission();

-- 群组最后消息时间更新触发器函数
CREATE OR REPLACE FUNCTION update_group_last_message_time()
RETURNS TRIGGER AS $$
BEGIN
    -- 更新群组的最后消息时间
    UPDATE groups
    SET updated_at = NEW.created_at
    WHERE id = NEW.group_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 群组最后消息时间更新触发器
CREATE TRIGGER trigger_messages_update_group_time
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_group_last_message_time();

-- ============================================================================
-- 动态相关触发器
-- ============================================================================

-- 动态表 updated_at 自动更新触发器
CREATE TRIGGER trigger_moments_updated_at
    BEFORE UPDATE ON moments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 动态点赞数量更新触发器函数
CREATE OR REPLACE FUNCTION update_moment_like_count()
RETURNS TRIGGER AS $$
DECLARE
    moment_id_val BIGINT;
    new_count INTEGER;
BEGIN
    -- 确定要更新的动态ID
    IF TG_OP = 'DELETE' THEN
        moment_id_val := OLD.moment_id;
    ELSE
        moment_id_val := NEW.moment_id;
    END IF;
    
    -- 计算点赞数量
    SELECT COUNT(*) INTO new_count
    FROM moment_likes
    WHERE moment_id = moment_id_val;
    
    -- 更新动态点赞数量
    UPDATE moments
    SET like_count = new_count,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = moment_id_val;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 动态点赞数量更新触发器
CREATE TRIGGER trigger_moment_likes_count_update
    AFTER INSERT OR DELETE ON moment_likes
    FOR EACH ROW
    EXECUTE FUNCTION update_moment_like_count();

-- 动态评论数量更新触发器函数
CREATE OR REPLACE FUNCTION update_moment_comment_count()
RETURNS TRIGGER AS $$
DECLARE
    moment_id_val BIGINT;
    new_count INTEGER;
BEGIN
    -- 确定要更新的动态ID
    IF TG_OP = 'DELETE' THEN
        moment_id_val := OLD.moment_id;
    ELSE
        moment_id_val := NEW.moment_id;
    END IF;
    
    -- 计算评论数量（不包括已删除的）
    SELECT COUNT(*) INTO new_count
    FROM moment_comments
    WHERE moment_id = moment_id_val AND deleted_at IS NULL;
    
    -- 更新动态评论数量
    UPDATE moments
    SET comment_count = new_count,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = moment_id_val;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 动态评论数量更新触发器
CREATE TRIGGER trigger_comments_count_update
    AFTER INSERT OR UPDATE OR DELETE ON moment_comments
    FOR EACH ROW
    EXECUTE FUNCTION update_moment_comment_count();

-- ============================================================================
-- 通知相关触发器
-- ============================================================================

-- 通知自动创建触发器函数
CREATE OR REPLACE FUNCTION create_notification()
RETURNS TRIGGER AS $$
BEGIN
    -- 根据不同的操作创建相应的通知
    CASE TG_TABLE_NAME
        WHEN 'group_invitations' THEN
            -- 群组邀请通知
            IF TG_OP = 'INSERT' AND NEW.status = 'pending' THEN
                INSERT INTO notifications (user_id, type, title, content, related_id, created_at)
                VALUES (
                    NEW.invitee_id,
                    'group_invitation',
                    '群组邀请',
                    '您收到了一个群组邀请',
                    NEW.id,
                    NEW.created_at
                );
            END IF;
            
        WHEN 'messages' THEN
            -- 新消息通知（需要根据群组成员创建多个通知）
            IF TG_OP = 'INSERT' THEN
                INSERT INTO notifications (user_id, type, title, content, related_id, created_at)
                SELECT 
                    gm.user_id,
                    'new_message',
                    '新消息',
                    '您有新的消息',
                    NEW.id,
                    NEW.created_at
                FROM group_members gm
                WHERE gm.group_id = NEW.group_id 
                AND gm.user_id != NEW.sender_id 
                AND gm.status = 'active';
            END IF;
            
        WHEN 'moment_likes' THEN
            -- 动态点赞通知
            IF TG_OP = 'INSERT' THEN
                INSERT INTO notifications (user_id, type, title, content, related_id, created_at)
                SELECT 
                    m.user_id,
                    'moment_like',
                    '动态点赞',
                    '有人点赞了您的动态',
                    NEW.moment_id,
                    NEW.created_at
                FROM moments m
                WHERE m.id = NEW.moment_id AND m.user_id != NEW.user_id;
            END IF;
            
        WHEN 'moment_comments' THEN
            -- 动态评论通知
            IF TG_OP = 'INSERT' THEN
                INSERT INTO notifications (user_id, type, title, content, related_id, created_at)
                SELECT 
                    m.user_id,
                    'moment_comment',
                    '动态评论',
                    '有人评论了您的动态',
                    NEW.moment_id,
                    NEW.created_at
                FROM moments m
                WHERE m.id = NEW.moment_id AND m.user_id != NEW.user_id;
            END IF;
    END CASE;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 各种通知触发器
CREATE TRIGGER trigger_group_invitations_notification
    AFTER INSERT OR UPDATE ON group_invitations
    FOR EACH ROW
    EXECUTE FUNCTION create_notification();

CREATE TRIGGER trigger_messages_notification
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION create_notification();

CREATE TRIGGER trigger_moment_likes_notification
    AFTER INSERT ON moment_likes
    FOR EACH ROW
    EXECUTE FUNCTION create_notification();

CREATE TRIGGER trigger_comments_notification
    AFTER INSERT ON moment_comments
    FOR EACH ROW
    EXECUTE FUNCTION create_notification();

-- ============================================================================
-- 数据完整性触发器
-- ============================================================================

-- 防止删除最后一个群组管理员的触发器函数
CREATE OR REPLACE FUNCTION prevent_last_admin_removal()
RETURNS TRIGGER AS $$
DECLARE
    admin_count INTEGER;
BEGIN
    -- 如果是删除操作或者将管理员角色改为其他角色
    IF (TG_OP = 'DELETE' AND OLD.role = 'admin') OR 
       (TG_OP = 'UPDATE' AND OLD.role = 'admin' AND NEW.role != 'admin') THEN
        
        -- 计算剩余管理员数量
        SELECT COUNT(*) INTO admin_count
        FROM group_members
        WHERE group_id = COALESCE(OLD.group_id, NEW.group_id)
        AND role = 'admin'
        AND status = 'active'
        AND id != COALESCE(OLD.id, NEW.id);
        
        -- 如果没有其他管理员，阻止操作
        IF admin_count = 0 THEN
            RAISE EXCEPTION '不能删除群组的最后一个管理员';
        END IF;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 防止删除最后一个群组管理员触发器
CREATE TRIGGER trigger_group_members_prevent_last_admin
    BEFORE UPDATE OR DELETE ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION prevent_last_admin_removal();

-- 消息回复链检查触发器函数
CREATE OR REPLACE FUNCTION check_reply_chain()
RETURNS TRIGGER AS $$
DECLARE
    reply_group_id BIGINT;
    chain_depth INTEGER := 0;
    current_reply_id BIGINT;
BEGIN
    -- 如果是回复消息
    IF NEW.reply_to_id IS NOT NULL THEN
        -- 检查被回复的消息是否在同一个群组
        SELECT group_id INTO reply_group_id
        FROM messages
        WHERE id = NEW.reply_to_id;
        
        IF reply_group_id != NEW.group_id THEN
            RAISE EXCEPTION '不能回复其他群组的消息';
        END IF;
        
        -- 检查回复链深度，防止过深的嵌套
        current_reply_id := NEW.reply_to_id;
        WHILE current_reply_id IS NOT NULL AND chain_depth < 10 LOOP
            SELECT reply_to_id INTO current_reply_id
            FROM messages
            WHERE id = current_reply_id;
            chain_depth := chain_depth + 1;
        END LOOP;
        
        IF chain_depth >= 10 THEN
            RAISE EXCEPTION '回复链过深，最多支持10层嵌套';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 消息回复链检查触发器
CREATE TRIGGER trigger_messages_reply_chain_check
    BEFORE INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION check_reply_chain();

-- ============================================================================
-- 缓存失效触发器
-- ============================================================================

-- 缓存失效通知触发器函数
CREATE OR REPLACE FUNCTION notify_cache_invalidation()
RETURNS TRIGGER AS $$
DECLARE
    cache_key TEXT;
    notification_payload JSON;
BEGIN
    -- 根据表名和操作类型构建缓存失效通知
    CASE TG_TABLE_NAME
        WHEN 'users' THEN
            cache_key := 'user:' || COALESCE(NEW.id, OLD.id)::TEXT;
        WHEN 'groups' THEN
            cache_key := 'group:' || COALESCE(NEW.id, OLD.id)::TEXT;
        WHEN 'messages' THEN
            cache_key := 'group_messages:' || COALESCE(NEW.group_id, OLD.group_id)::TEXT;
        WHEN 'group_members' THEN
            cache_key := 'group_members:' || COALESCE(NEW.group_id, OLD.group_id)::TEXT;
        ELSE
            RETURN COALESCE(NEW, OLD);
    END CASE;
    
    -- 构建通知载荷
    notification_payload := json_build_object(
        'action', TG_OP,
        'table', TG_TABLE_NAME,
        'cache_key', cache_key,
        'timestamp', EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)
    );
    
    -- 发送缓存失效通知
    PERFORM pg_notify('cache_invalidation', notification_payload::TEXT);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 各表的缓存失效触发器
CREATE TRIGGER trigger_users_cache_invalidation
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW
    EXECUTE FUNCTION notify_cache_invalidation();

CREATE TRIGGER trigger_groups_cache_invalidation
    AFTER INSERT OR UPDATE OR DELETE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION notify_cache_invalidation();

CREATE TRIGGER trigger_messages_cache_invalidation
    AFTER INSERT OR UPDATE OR DELETE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION notify_cache_invalidation();

CREATE TRIGGER trigger_group_members_cache_invalidation
    AFTER INSERT OR UPDATE OR DELETE ON group_members
    FOR EACH ROW
    EXECUTE FUNCTION notify_cache_invalidation();

-- ============================================================================
-- 触发器管理函数
-- ============================================================================

-- 获取所有触发器信息的函数
CREATE OR REPLACE FUNCTION get_trigger_info()
RETURNS TABLE(
    trigger_name TEXT,
    table_name TEXT,
    trigger_event TEXT,
    trigger_timing TEXT,
    function_name TEXT,
    is_enabled BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.trigger_name::TEXT,
        t.event_object_table::TEXT,
        string_agg(t.event_manipulation, ', ' ORDER BY t.event_manipulation)::TEXT,
        t.action_timing::TEXT,
        t.action_statement::TEXT,
        NOT t.is_disabled as is_enabled
    FROM information_schema.triggers t
    WHERE t.trigger_schema = 'public'
    GROUP BY t.trigger_name, t.event_object_table, t.action_timing, t.action_statement, t.is_disabled
    ORDER BY t.event_object_table, t.trigger_name;
END;
$$ LANGUAGE plpgsql;

-- 禁用/启用触发器的函数
CREATE OR REPLACE FUNCTION toggle_trigger(
    trigger_name_param TEXT,
    table_name_param TEXT,
    enable_trigger BOOLEAN DEFAULT TRUE
) RETURNS VOID AS $$
DECLARE
    action_text TEXT;
BEGIN
    action_text := CASE WHEN enable_trigger THEN 'ENABLE' ELSE 'DISABLE' END;
    
    EXECUTE format('ALTER TABLE %I %s TRIGGER %I', 
                   table_name_param, action_text, trigger_name_param);
    
    RAISE NOTICE '触发器 % 在表 % 上已%', 
                 trigger_name_param, table_name_param, 
                 CASE WHEN enable_trigger THEN '启用' ELSE '禁用' END;
END;
$$ LANGUAGE plpgsql;

-- 添加函数注释
COMMENT ON FUNCTION update_updated_at_column() IS '自动更新updated_at字段的通用触发器函数';
COMMENT ON FUNCTION soft_delete_trigger() IS '软删除触发器函数';
COMMENT ON FUNCTION validate_user_id_type() IS '验证用户ID类型的触发器函数';
COMMENT ON FUNCTION get_trigger_info() IS '获取所有触发器信息';
COMMENT ON FUNCTION toggle_trigger(TEXT, TEXT, BOOLEAN) IS '启用或禁用指定触发器';

COMMENT ON SCHEMA public IS '包含完整触发器系统的主要模式，支持自动化业务逻辑和数据完整性';
