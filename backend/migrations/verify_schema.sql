-- ============================================================================
-- HotSpot 数据库架构验证脚本
-- ============================================================================
-- 
-- 功能：验证数据库架构是否正确安装
-- 用法：psql -d hotspot -f verify_schema.sql
-- 
-- ============================================================================

\echo '🔍 开始验证 HotSpot 数据库架构...'
\echo ''

-- ============================================================================
-- 1. 基础信息检查
-- ============================================================================

\echo '📊 基础信息检查'
\echo '----------------------------------------'

-- 检查数据库版本
SELECT 'PostgreSQL版本: ' || version() as info;

-- 检查当前数据库
SELECT '当前数据库: ' || current_database() as info;

-- 检查当前用户
SELECT '当前用户: ' || current_user as info;

\echo ''

-- ============================================================================
-- 2. 扩展检查
-- ============================================================================

\echo '🔌 扩展检查'
\echo '----------------------------------------'

SELECT 
    extname as "扩展名",
    CASE 
        WHEN extname IN ('postgis', 'pgcrypto', 'uuid-ossp', 'pg_stat_statements') 
        THEN '✅ 已安装'
        ELSE '❌ 未安装'
    END as "状态"
FROM pg_extension 
WHERE extname IN ('postgis', 'pgcrypto', 'uuid-ossp', 'pg_stat_statements')
ORDER BY extname;

\echo ''

-- ============================================================================
-- 3. 表结构检查
-- ============================================================================

\echo '📋 表结构检查'
\echo '----------------------------------------'

WITH table_info AS (
    SELECT 
        t.table_name,
        CASE 
            WHEN t.table_name IN (
                'users', 'user_settings', 'user_activity', 'user_relationships',
                'groups', 'group_members',
                'messages', 'message_media', 'message_reactions',
                'moments', 'tags', 'moment_tags', 'moment_likes', 'moment_comments',
                'reports', 'notifications'
            ) THEN '✅ 核心表'
            WHEN t.table_name LIKE 'messages_%' OR t.table_name LIKE 'user_activity_%' 
            THEN '📊 分区表'
            ELSE '❓ 其他表'
        END as table_type,
        (SELECT COUNT(*) FROM information_schema.columns c WHERE c.table_name = t.table_name) as column_count
    FROM information_schema.tables t
    WHERE t.table_schema = 'public' 
      AND t.table_type = 'BASE TABLE'
)
SELECT 
    table_type as "表类型",
    COUNT(*) as "数量",
    string_agg(table_name, ', ' ORDER BY table_name) as "表名列表"
FROM table_info
GROUP BY table_type
ORDER BY 
    CASE table_type 
        WHEN '✅ 核心表' THEN 1 
        WHEN '📊 分区表' THEN 2 
        ELSE 3 
    END;

\echo ''

-- ============================================================================
-- 4. 枚举类型检查
-- ============================================================================

\echo '🏷️  枚举类型检查'
\echo '----------------------------------------'

SELECT 
    t.typname as "枚举类型",
    array_to_string(array_agg(e.enumlabel ORDER BY e.enumsortorder), ', ') as "枚举值"
FROM pg_type t
JOIN pg_enum e ON t.oid = e.enumtypid
WHERE t.typname LIKE '%_enum'
GROUP BY t.typname
ORDER BY t.typname;

\echo ''

-- ============================================================================
-- 5. 函数检查
-- ============================================================================

\echo '⚙️  函数检查'
\echo '----------------------------------------'

SELECT 
    p.proname as "函数名",
    pg_get_function_arguments(p.oid) as "参数",
    CASE 
        WHEN p.proname IN ('generate_user_id', 'get_user_type', 'create_private_chat_group', 'send_unified_message')
        THEN '✅ 核心函数'
        ELSE '🔧 辅助函数'
    END as "类型"
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = 'public'
  AND p.proname NOT LIKE 'update_%_column'
  AND p.proname NOT LIKE 'soft_delete_%'
ORDER BY 
    CASE 
        WHEN p.proname IN ('generate_user_id', 'get_user_type', 'create_private_chat_group', 'send_unified_message')
        THEN 1 ELSE 2 
    END,
    p.proname;

\echo ''

-- ============================================================================
-- 6. 索引检查
-- ============================================================================

\echo '🗂️  索引检查'
\echo '----------------------------------------'

SELECT 
    schemaname as "模式",
    tablename as "表名",
    COUNT(*) as "索引数量"
FROM pg_indexes 
WHERE schemaname = 'public'
  AND indexname LIKE 'idx_%'
GROUP BY schemaname, tablename
ORDER BY COUNT(*) DESC, tablename;

\echo ''

-- ============================================================================
-- 7. 功能测试
-- ============================================================================

\echo '🧪 功能测试'
\echo '----------------------------------------'

-- 测试用户ID编码
\echo '测试用户ID编码功能:'
SELECT 
    '临时用户ID: ' || generate_user_id(1) as "结果"
UNION ALL
SELECT 
    '普通用户ID: ' || generate_user_id(2) as "结果"
UNION ALL
SELECT 
    '系统用户ID: ' || generate_user_id(3) as "结果";

\echo ''

-- 测试用户类型提取
\echo '测试用户类型提取:'
WITH test_users AS (
    SELECT generate_user_id(1) as temp_id, generate_user_id(2) as regular_id, generate_user_id(3) as system_id
)
SELECT 
    '临时用户类型: ' || get_user_type(temp_id) as "结果"
FROM test_users
UNION ALL
SELECT 
    '普通用户类型: ' || get_user_type(regular_id) as "结果"
FROM test_users
UNION ALL
SELECT 
    '系统用户类型: ' || get_user_type(system_id) as "结果"
FROM test_users;

\echo ''

-- 测试私聊群组创建
\echo '测试私聊群组创建:'
WITH test_group AS (
    SELECT create_private_chat_group(generate_user_id(2), generate_user_id(2)) as group_id
)
SELECT 
    '私聊群组ID: ' || group_id as "结果"
FROM test_group;

\echo ''

-- ============================================================================
-- 8. 数据完整性检查
-- ============================================================================

\echo '🔒 数据完整性检查'
\echo '----------------------------------------'

-- 检查分区表
SELECT 
    'messages分区数量: ' || COUNT(*) as "结果"
FROM information_schema.tables 
WHERE table_name LIKE 'messages_%' AND table_schema = 'public';

-- 检查触发器
SELECT 
    '触发器数量: ' || COUNT(*) as "结果"
FROM information_schema.triggers 
WHERE trigger_schema = 'public';

-- 检查视图
SELECT 
    '视图数量: ' || COUNT(*) as "结果"
FROM information_schema.views 
WHERE table_schema = 'public';

\echo ''

-- ============================================================================
-- 9. 总结
-- ============================================================================

\echo '📈 架构总结'
\echo '----------------------------------------'

WITH summary AS (
    SELECT 
        (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE') as total_tables,
        (SELECT COUNT(*) FROM pg_extension WHERE extname IN ('postgis', 'pgcrypto', 'uuid-ossp')) as extensions,
        (SELECT COUNT(*) FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'public' AND p.proname IN ('generate_user_id', 'get_user_type', 'create_private_chat_group', 'send_unified_message')) as core_functions,
        (SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname LIKE 'idx_%') as indexes
)
SELECT 
    '总表数: ' || total_tables || ' (预期: ≥20)' as "统计信息"
FROM summary
UNION ALL
SELECT 
    '核心扩展: ' || extensions || ' (预期: 3)' as "统计信息"
FROM summary
UNION ALL
SELECT 
    '核心函数: ' || core_functions || ' (预期: 4)' as "统计信息"
FROM summary
UNION ALL
SELECT 
    '索引数量: ' || indexes || ' (预期: ≥30)' as "统计信息"
FROM summary;

\echo ''
\echo '✅ HotSpot 数据库架构验证完成！'
\echo ''

-- 显示连接信息
\echo '🔗 连接信息:'
\echo '数据库: ' :'DBNAME'
\echo '用户: ' :'USER'
\echo '主机: ' :'HOST'
\echo '端口: ' :'PORT'
\echo ''
