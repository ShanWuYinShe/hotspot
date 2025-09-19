-- ============================================================================
-- HotSpot 软删除触发器迁移
-- ============================================================================
-- 
-- 功能职责：
-- - 实现软删除机制的触发器
-- - 自动维护相关数据的一致性
-- - 级联软删除相关记录
--
-- 依赖关系：
-- - 依赖现有的表结构
--
-- ============================================================================

-- 1. 软删除级联触发器函数
CREATE OR REPLACE FUNCTION soft_delete_cascade()
RETURNS TRIGGER AS $$
BEGIN
    -- 当用户被软删除时，级联软删除相关数据
    IF TG_TABLE_NAME = 'users' AND NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        -- 软删除用户的群组成员关系
        UPDATE group_members 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE user_id = NEW.id AND deleted_at IS NULL;
        
        -- 软删除用户发送的消息
        UPDATE direct_messages 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE sender_id = NEW.id AND deleted_at IS NULL;
        
        UPDATE group_messages 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE sender_id = NEW.id AND deleted_at IS NULL;
        
        -- 软删除用户的生活瞬间
        UPDATE moments 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE user_id = NEW.id AND deleted_at IS NULL;
        
        -- 软删除用户的评论
        UPDATE moment_comments 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE user_id = NEW.id AND deleted_at IS NULL;
        
        -- 软删除用户的点赞
        UPDATE moment_likes 
        SET deleted_at = NOW()
        WHERE user_id = NEW.id AND deleted_at IS NULL;
        
        -- 软删除用户关系
        UPDATE user_relationships 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE (from_user_id = NEW.id OR to_user_id = NEW.id) AND deleted_at IS NULL;
        
        RAISE NOTICE '用户 % 的相关数据已软删除', NEW.id;
    END IF;
    
    -- 当群组被软删除时，级联软删除相关数据
    IF TG_TABLE_NAME = 'groups' AND NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        -- 软删除群组成员
        UPDATE group_members 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE group_id = NEW.id AND deleted_at IS NULL;
        
        -- 软删除群组消息
        UPDATE group_messages 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE group_id = NEW.id AND deleted_at IS NULL;
        
        RAISE NOTICE '群组 % 的相关数据已软删除', NEW.id;
    END IF;
    
    -- 当生活瞬间被软删除时，级联软删除相关数据
    IF TG_TABLE_NAME = 'moments' AND NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        -- 软删除瞬间的评论
        UPDATE moment_comments 
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE moment_id = NEW.id AND deleted_at IS NULL;
        
        -- 软删除瞬间的点赞
        UPDATE moment_likes 
        SET deleted_at = NOW()
        WHERE moment_id = NEW.id AND deleted_at IS NULL;
        
        RAISE NOTICE '生活瞬间 % 的相关数据已软删除', NEW.id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. 自动更新 updated_at 字段的触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 3. 为主要表创建软删除触发器
-- 用户表软删除触发器
DROP TRIGGER IF EXISTS trigger_soft_delete_users ON users;
CREATE TRIGGER trigger_soft_delete_users
    AFTER UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION soft_delete_cascade();

-- 群组表软删除触发器
DROP TRIGGER IF EXISTS trigger_soft_delete_groups ON groups;
CREATE TRIGGER trigger_soft_delete_groups
    AFTER UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION soft_delete_cascade();

-- 生活瞬间表软删除触发器
DROP TRIGGER IF EXISTS trigger_soft_delete_moments ON moments;
CREATE TRIGGER trigger_soft_delete_moments
    AFTER UPDATE ON moments
    FOR EACH ROW EXECUTE FUNCTION soft_delete_cascade();

-- 4. 为主要表创建 updated_at 自动更新触发器
-- 用户表
DROP TRIGGER IF EXISTS trigger_update_users_updated_at ON users;
CREATE TRIGGER trigger_update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 群组表
DROP TRIGGER IF EXISTS trigger_update_groups_updated_at ON groups;
CREATE TRIGGER trigger_update_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 群组成员表
DROP TRIGGER IF EXISTS trigger_update_group_members_updated_at ON group_members;
CREATE TRIGGER trigger_update_group_members_updated_at
    BEFORE UPDATE ON group_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 私聊消息表
DROP TRIGGER IF EXISTS trigger_update_direct_messages_updated_at ON direct_messages;
CREATE TRIGGER trigger_update_direct_messages_updated_at
    BEFORE UPDATE ON direct_messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 群组消息表
DROP TRIGGER IF EXISTS trigger_update_group_messages_updated_at ON group_messages;
CREATE TRIGGER trigger_update_group_messages_updated_at
    BEFORE UPDATE ON group_messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 生活瞬间表
DROP TRIGGER IF EXISTS trigger_update_moments_updated_at ON moments;
CREATE TRIGGER trigger_update_moments_updated_at
    BEFORE UPDATE ON moments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 生活瞬间评论表
DROP TRIGGER IF EXISTS trigger_update_moment_comments_updated_at ON moment_comments;
CREATE TRIGGER trigger_update_moment_comments_updated_at
    BEFORE UPDATE ON moment_comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 用户关系表
DROP TRIGGER IF EXISTS trigger_update_user_relationships_updated_at ON user_relationships;
CREATE TRIGGER trigger_update_user_relationships_updated_at
    BEFORE UPDATE ON user_relationships
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 5. 创建软删除恢复函数
CREATE OR REPLACE FUNCTION restore_soft_deleted_user(user_id_param BIGINT)
RETURNS VOID AS $$
BEGIN
    -- 恢复用户
    UPDATE users 
    SET deleted_at = NULL, updated_at = NOW()
    WHERE id = user_id_param AND deleted_at IS NOT NULL;
    
    -- 恢复用户的群组成员关系
    UPDATE group_members 
    SET deleted_at = NULL, updated_at = NOW()
    WHERE user_id = user_id_param AND deleted_at IS NOT NULL;
    
    -- 恢复用户发送的消息
    UPDATE direct_messages 
    SET deleted_at = NULL, updated_at = NOW()
    WHERE sender_id = user_id_param AND deleted_at IS NOT NULL;
    
    UPDATE group_messages 
    SET deleted_at = NULL, updated_at = NOW()
    WHERE sender_id = user_id_param AND deleted_at IS NOT NULL;
    
    -- 恢复用户的生活瞬间
    UPDATE moments 
    SET deleted_at = NULL, updated_at = NOW()
    WHERE user_id = user_id_param AND deleted_at IS NOT NULL;
    
    -- 恢复用户的评论
    UPDATE moment_comments 
    SET deleted_at = NULL, updated_at = NOW()
    WHERE user_id = user_id_param AND deleted_at IS NOT NULL;
    
    -- 恢复用户的点赞
    UPDATE moment_likes 
    SET deleted_at = NULL
    WHERE user_id = user_id_param AND deleted_at IS NOT NULL;
    
    -- 恢复用户关系
    UPDATE user_relationships 
    SET deleted_at = NULL, updated_at = NOW()
    WHERE (from_user_id = user_id_param OR to_user_id = user_id_param) AND deleted_at IS NOT NULL;
    
    RAISE NOTICE '用户 % 及其相关数据已恢复', user_id_param;
END;
$$ LANGUAGE plpgsql;

-- 6. 创建清理过期软删除数据的函数
CREATE OR REPLACE FUNCTION cleanup_expired_soft_deleted_data(days_old INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := NOW() - INTERVAL '1 day' * days_old;
    
    -- 物理删除过期的软删除用户
    DELETE FROM users 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- 物理删除过期的软删除群组
    DELETE FROM groups 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    
    -- 物理删除过期的软删除消息
    DELETE FROM direct_messages 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    
    DELETE FROM group_messages 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    
    -- 物理删除过期的软删除生活瞬间
    DELETE FROM moments 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    
    -- 物理删除过期的软删除评论
    DELETE FROM moment_comments 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    
    -- 物理删除过期的软删除点赞
    DELETE FROM moment_likes 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    
    -- 物理删除过期的软删除用户关系
    DELETE FROM user_relationships 
    WHERE deleted_at IS NOT NULL AND deleted_at < cutoff_date;
    
    RAISE NOTICE '清理了 % 天前的软删除数据，共删除 % 条用户记录', days_old, deleted_count;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- 7. 创建软删除统计视图
CREATE OR REPLACE VIEW soft_delete_stats AS
SELECT 
    'users' as table_name,
    COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) as deleted_count,
    COUNT(*) as total_count
FROM users
UNION ALL
SELECT 
    'groups' as table_name,
    COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) as deleted_count,
    COUNT(*) as total_count
FROM groups
UNION ALL
SELECT 
    'direct_messages' as table_name,
    COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) as deleted_count,
    COUNT(*) as total_count
FROM direct_messages
UNION ALL
SELECT 
    'group_messages' as table_name,
    COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) as deleted_count,
    COUNT(*) as total_count
FROM group_messages
UNION ALL
SELECT 
    'moments' as table_name,
    COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count,
    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) as deleted_count,
    COUNT(*) as total_count
FROM moments;

-- 8. 添加注释
COMMENT ON FUNCTION soft_delete_cascade() IS '软删除级联触发器函数，自动处理相关数据的软删除';
COMMENT ON FUNCTION update_updated_at_column() IS '自动更新updated_at字段的触发器函数';
COMMENT ON FUNCTION restore_soft_deleted_user(BIGINT) IS '恢复软删除用户及其相关数据';
COMMENT ON FUNCTION cleanup_expired_soft_deleted_data(INTEGER) IS '清理过期的软删除数据（默认365天）';
COMMENT ON VIEW soft_delete_stats IS '软删除数据统计视图';

-- 9. 创建定期清理任务的示例（需要pg_cron扩展）
-- 注意：这个部分需要安装pg_cron扩展才能使用
-- CREATE EXTENSION IF NOT EXISTS pg_cron;
-- SELECT cron.schedule('cleanup-soft-deleted', '0 2 * * 0', 'SELECT cleanup_expired_soft_deleted_data(365);');

RAISE NOTICE '软删除触发器系统已安装完成';
