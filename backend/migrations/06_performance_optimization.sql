-- 数据库性能优化
-- 包含: 复合索引、查询优化、统计信息更新

-- 1. 用户相关复合索引优化
-- 优化附近用户查询（按距离和活跃度）
CREATE INDEX IF NOT EXISTS idx_user_locations_geom 
ON user_locations USING gist (geom);
CREATE INDEX IF NOT EXISTS idx_user_locations_updated_at 
ON user_locations (updated_at DESC);

-- 优化用户搜索（按昵称和活跃度）
CREATE INDEX IF NOT EXISTS idx_users_nickname_last_active 
ON users(nickname, last_active_at DESC);

-- 优化用户ID范围查询（分页）
CREATE INDEX IF NOT EXISTS idx_users_user_id_created_at 
ON users(user_id DESC, created_at DESC);

-- 2. 群组相关复合索引优化
-- 优化附近群组查询（按距离和活跃度）
CREATE INDEX IF NOT EXISTS idx_groups_geom 
ON groups USING gist (geom);
CREATE INDEX IF NOT EXISTS idx_groups_last_active 
ON groups (last_active_at DESC);

-- 优化群组成员查询
CREATE INDEX IF NOT EXISTS idx_group_members_user_role_joined 
ON group_members(user_id, role, joined_at DESC);

-- 优化群组消息查询（按群组和时间）
CREATE INDEX IF NOT EXISTS idx_group_messages_group_created 
ON group_messages(group_id, created_at DESC);

-- 3. 生活瞬间相关复合索引优化
-- 优化附近瞬间查询（按距离和时间）
CREATE INDEX IF NOT EXISTS idx_moments_geom 
ON moments USING gist (geom);
CREATE INDEX IF NOT EXISTS idx_moments_created_at 
ON moments (created_at DESC);

-- 优化用户瞬间查询
CREATE INDEX IF NOT EXISTS idx_moments_user_created 
ON moments(user_id, created_at DESC);

-- 优化瞬间点赞统计
CREATE INDEX IF NOT EXISTS idx_moment_likes_moment_created 
ON moment_likes(moment_id, created_at DESC);

-- 4. 消息相关复合索引优化
-- 优化会话消息查询
CREATE INDEX IF NOT EXISTS idx_direct_messages_conversation_created 
ON direct_messages(conversation_id, created_at DESC);

-- 优化未读消息统计
CREATE INDEX IF NOT EXISTS idx_direct_messages_recipient_created 
ON direct_messages(recipient_id, created_at DESC);

-- 5. 分区表优化（为大数据量准备）
-- 为消息表创建按月分区（如果数据量大）
-- 注意：这需要在生产环境中谨慎执行

-- 创建分区函数
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name text, start_date date)
RETURNS void AS $$
DECLARE
    partition_name text;
    end_date date;
BEGIN
    partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM');
    end_date := start_date + interval '1 month';
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I 
                    FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;

-- 6. 查询性能优化设置
-- 注意：以下设置需要在postgresql.conf中手动配置
-- work_mem = 256MB
-- maintenance_work_mem = 1GB
-- random_page_cost = 1.1
-- effective_cache_size = 4GB

-- 7. 统计信息更新
-- 更新表统计信息以优化查询计划
ANALYZE users;
ANALYZE user_locations;
ANALYZE groups;
ANALYZE group_members;
ANALYZE group_messages;
ANALYZE moments;
ANALYZE moment_likes;
ANALYZE direct_messages;

-- 8. 自动清理配置优化
-- 优化自动清理设置以保持性能
ALTER TABLE users SET (autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE user_locations SET (autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE groups SET (autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE group_messages SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE direct_messages SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE moments SET (autovacuum_vacuum_scale_factor = 0.1);

-- 9. 创建性能监控视图
CREATE OR REPLACE VIEW performance_stats AS
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats 
WHERE schemaname = 'public' 
  AND tablename IN ('users', 'user_locations', 'groups', 'moments', 'group_messages', 'direct_messages')
ORDER BY tablename, attname;

-- 10. 地理位置查询优化函数
CREATE OR REPLACE FUNCTION optimize_geo_query(
    center_lat double precision,
    center_lng double precision,
    radius_km double precision
) RETURNS TABLE (
    bbox_min_lat double precision,
    bbox_max_lat double precision,
    bbox_min_lng double precision,
    bbox_max_lng double precision
) AS $$
DECLARE
    lat_delta double precision;
    lng_delta double precision;
BEGIN
    -- 计算边界框以优化地理查询
    -- 1度纬度约等于111公里
    lat_delta := radius_km / 111.0;
    -- 经度随纬度变化，使用余弦修正
    lng_delta := radius_km / (111.0 * cos(radians(center_lat)));
    
    RETURN QUERY SELECT
        center_lat - lat_delta,
        center_lat + lat_delta,
        center_lng - lng_delta,
        center_lng + lng_delta;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 11. 批量操作优化
-- 创建批量插入函数以提高性能
CREATE OR REPLACE FUNCTION batch_insert_user_locations(
    user_ids bigint[],
    latitudes double precision[],
    longitudes double precision[],
    location_names text[]
) RETURNS integer AS $$
DECLARE
    i integer;
    inserted_count integer := 0;
BEGIN
    FOR i IN 1..array_length(user_ids, 1) LOOP
        INSERT INTO user_locations (user_id, latitude, longitude, location_name)
        VALUES (user_ids[i], latitudes[i], longitudes[i], location_names[i])
        ON CONFLICT (user_id) DO UPDATE SET
            latitude = EXCLUDED.latitude,
            longitude = EXCLUDED.longitude,
            location_name = EXCLUDED.location_name,
            updated_at = NOW();
        
        inserted_count := inserted_count + 1;
    END LOOP;
    
    RETURN inserted_count;
END;
$$ LANGUAGE plpgsql;

-- 12. 缓存预热查询
-- 创建常用查询的物化视图
CREATE MATERIALIZED VIEW IF NOT EXISTS popular_locations AS
SELECT 
    location_name,
    COUNT(*) as user_count,
    AVG(latitude) as avg_latitude,
    AVG(longitude) as avg_longitude
FROM user_locations 
WHERE updated_at > NOW() - INTERVAL '7 days'
GROUP BY location_name
HAVING COUNT(*) >= 5
ORDER BY user_count DESC
LIMIT 100;

-- 创建刷新物化视图的函数
CREATE OR REPLACE FUNCTION refresh_popular_locations()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW popular_locations;
END;
$$ LANGUAGE plpgsql;

-- 13. 性能监控查询
-- 注意：慢查询监控需要启用pg_stat_statements扩展
-- CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- 注释：应用这些优化后的注意事项
-- 1. 需要重启PostgreSQL以应用系统参数更改
-- 2. 建议在低峰期执行索引创建
-- 3. 定期运行ANALYZE更新统计信息
-- 4. 监控磁盘空间，索引会占用额外空间
-- 5. 根据实际查询模式调整索引策略
