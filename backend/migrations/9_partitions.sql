-- ============================================================================
-- 10. 分区表创建
-- ============================================================================
-- 
-- 功能职责：
-- - 为大数据量表创建分区
-- - 提高查询性能和维护效率
-- - 支持按时间和其他维度分区
-- - 自动化分区管理
--
-- 依赖关系：
-- - 依赖：所有前置模块（01-09）
--
-- ============================================================================

-- 为用户活动记录表创建按月分区
-- 当前月分区
CREATE TABLE user_activity_y2024m01 PARTITION OF user_activity
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE user_activity_y2024m02 PARTITION OF user_activity
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE user_activity_y2024m03 PARTITION OF user_activity
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

CREATE TABLE user_activity_y2024m04 PARTITION OF user_activity
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

CREATE TABLE user_activity_y2024m05 PARTITION OF user_activity
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

CREATE TABLE user_activity_y2024m06 PARTITION OF user_activity
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');

CREATE TABLE user_activity_y2024m07 PARTITION OF user_activity
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE user_activity_y2024m08 PARTITION OF user_activity
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');

CREATE TABLE user_activity_y2024m09 PARTITION OF user_activity
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');

CREATE TABLE user_activity_y2024m10 PARTITION OF user_activity
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');

CREATE TABLE user_activity_y2024m11 PARTITION OF user_activity
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');

CREATE TABLE user_activity_y2024m12 PARTITION OF user_activity
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- 2025年分区
CREATE TABLE user_activity_y2025m01 PARTITION OF user_activity
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE user_activity_y2025m02 PARTITION OF user_activity
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

CREATE TABLE user_activity_y2025m03 PARTITION OF user_activity
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

CREATE TABLE user_activity_y2025m04 PARTITION OF user_activity
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');

CREATE TABLE user_activity_y2025m05 PARTITION OF user_activity
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');

CREATE TABLE user_activity_y2025m06 PARTITION OF user_activity
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

CREATE TABLE user_activity_y2025m07 PARTITION OF user_activity
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');

CREATE TABLE user_activity_y2025m08 PARTITION OF user_activity
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');

CREATE TABLE user_activity_y2025m09 PARTITION OF user_activity
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');

CREATE TABLE user_activity_y2025m10 PARTITION OF user_activity
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');

CREATE TABLE user_activity_y2025m11 PARTITION OF user_activity
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

CREATE TABLE user_activity_y2025m12 PARTITION OF user_activity
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- 为消息表创建按月分区
-- 2024年分区
CREATE TABLE messages_y2024m01 PARTITION OF messages
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE messages_y2024m02 PARTITION OF messages
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE messages_y2024m03 PARTITION OF messages
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

CREATE TABLE messages_y2024m04 PARTITION OF messages
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

CREATE TABLE messages_y2024m05 PARTITION OF messages
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

CREATE TABLE messages_y2024m06 PARTITION OF messages
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');

CREATE TABLE messages_y2024m07 PARTITION OF messages
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE messages_y2024m08 PARTITION OF messages
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');

CREATE TABLE messages_y2024m09 PARTITION OF messages
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');

CREATE TABLE messages_y2024m10 PARTITION OF messages
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');

CREATE TABLE messages_y2024m11 PARTITION OF messages
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');

CREATE TABLE messages_y2024m12 PARTITION OF messages
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- 2025年分区
CREATE TABLE messages_y2025m01 PARTITION OF messages
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE messages_y2025m02 PARTITION OF messages
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

CREATE TABLE messages_y2025m03 PARTITION OF messages
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

CREATE TABLE messages_y2025m04 PARTITION OF messages
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');

CREATE TABLE messages_y2025m05 PARTITION OF messages
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');

CREATE TABLE messages_y2025m06 PARTITION OF messages
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

CREATE TABLE messages_y2025m07 PARTITION OF messages
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');

CREATE TABLE messages_y2025m08 PARTITION OF messages
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');

CREATE TABLE messages_y2025m09 PARTITION OF messages
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');

CREATE TABLE messages_y2025m10 PARTITION OF messages
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');

CREATE TABLE messages_y2025m11 PARTITION OF messages
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

CREATE TABLE messages_y2025m12 PARTITION OF messages
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- 创建分区管理函数
CREATE OR REPLACE FUNCTION create_monthly_partition(
    table_name TEXT,
    start_date DATE
) RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    end_date DATE;
BEGIN
    -- 计算分区名称和结束日期
    partition_name := table_name || '_y' || EXTRACT(YEAR FROM start_date) || 'm' || 
                     LPAD(EXTRACT(MONTH FROM start_date)::TEXT, 2, '0');
    end_date := start_date + INTERVAL '1 month';
    
    -- 创建分区
    EXECUTE format('CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
    
    RAISE NOTICE '创建分区: %', partition_name;
END;
$$ LANGUAGE plpgsql;

-- 创建自动分区管理函数
CREATE OR REPLACE FUNCTION maintain_partitions() RETURNS VOID AS $$
DECLARE
    current_date DATE := CURRENT_DATE;
    future_date DATE;
    partition_tables TEXT[] := ARRAY['user_activity', 'messages'];
    table_name TEXT;
BEGIN
    -- 为未来3个月创建分区
    FOR i IN 1..3 LOOP
        future_date := date_trunc('month', current_date) + (i || ' months')::INTERVAL;
        
        FOREACH table_name IN ARRAY partition_tables LOOP
            -- 检查分区是否已存在
            IF NOT EXISTS (
                SELECT 1 FROM pg_tables 
                WHERE tablename = table_name || '_y' || EXTRACT(YEAR FROM future_date) || 'm' || 
                      LPAD(EXTRACT(MONTH FROM future_date)::TEXT, 2, '0')
            ) THEN
                PERFORM create_monthly_partition(table_name, future_date);
            END IF;
        END LOOP;
    END LOOP;
    
    RAISE NOTICE '分区维护完成';
END;
$$ LANGUAGE plpgsql;

-- 创建清理旧分区函数
CREATE OR REPLACE FUNCTION cleanup_old_partitions(
    table_name TEXT,
    retention_months INTEGER DEFAULT 12
) RETURNS VOID AS $$
DECLARE
    cutoff_date DATE;
    partition_name TEXT;
    partition_record RECORD;
BEGIN
    -- 计算保留截止日期
    cutoff_date := date_trunc('month', CURRENT_DATE) - (retention_months || ' months')::INTERVAL;
    
    -- 查找需要删除的分区
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE table_name || '_y%'
        AND schemaname = 'public'
    LOOP
        -- 从分区名称提取日期
        partition_name := partition_record.tablename;
        
        -- 简单的日期提取（假设格式为 table_yYYYYmMM）
        DECLARE
            year_part INTEGER;
            month_part INTEGER;
            partition_date DATE;
        BEGIN
            year_part := substring(partition_name from '_y(\d{4})m')::INTEGER;
            month_part := substring(partition_name from 'm(\d{2})$')::INTEGER;
            partition_date := make_date(year_part, month_part, 1);
            
            -- 如果分区日期早于截止日期，则删除
            IF partition_date < cutoff_date THEN
                EXECUTE format('DROP TABLE IF EXISTS %I', partition_name);
                RAISE NOTICE '删除旧分区: %', partition_name;
            END IF;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE '跳过无效分区名称: %', partition_name;
        END;
    END LOOP;
    
    RAISE NOTICE '旧分区清理完成';
END;
$$ LANGUAGE plpgsql;

-- 创建分区统计函数
CREATE OR REPLACE FUNCTION get_partition_statistics(table_name TEXT)
RETURNS TABLE(
    partition_name TEXT,
    row_count BIGINT,
    size_bytes BIGINT,
    size_pretty TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.tablename::TEXT,
        COALESCE(s.n_tup_ins - s.n_tup_del, 0) as row_count,
        pg_total_relation_size(t.tablename::regclass) as size_bytes,
        pg_size_pretty(pg_total_relation_size(t.tablename::regclass)) as size_pretty
    FROM pg_tables t
    LEFT JOIN pg_stat_user_tables s ON t.tablename = s.relname
    WHERE t.tablename LIKE table_name || '_y%'
    AND t.schemaname = 'public'
    ORDER BY t.tablename;
END;
$$ LANGUAGE plpgsql;

-- 添加函数注释
COMMENT ON FUNCTION create_monthly_partition(TEXT, DATE) IS '为指定表创建月度分区';
COMMENT ON FUNCTION maintain_partitions() IS '自动维护分区，为未来月份创建分区';
COMMENT ON FUNCTION cleanup_old_partitions(TEXT, INTEGER) IS '清理超过保留期的旧分区';
COMMENT ON FUNCTION get_partition_statistics(TEXT) IS '获取指定表的分区统计信息';

-- 创建定时任务（需要pg_cron扩展，可选）
-- SELECT cron.schedule('partition-maintenance', '0 2 1 * *', 'SELECT maintain_partitions();');
-- SELECT cron.schedule('partition-cleanup', '0 3 1 * *', 'SELECT cleanup_old_partitions(''user_activity'', 12); SELECT cleanup_old_partitions(''messages'', 6);');

COMMENT ON SCHEMA public IS '包含分区管理的主要模式，支持按时间自动分区和清理';
