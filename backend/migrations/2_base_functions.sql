-- ============================================================================
-- 03. 基础函数定义
-- ============================================================================
-- 
-- 功能职责：
-- - 定义系统核心的基础函数
-- - 提供唯一ID生成功能
-- - 实现用户ID编码规范
-- - 支持其他模块的基础功能需求
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql
--
-- ============================================================================

-- 用户ID序列管理表
CREATE TABLE user_id_sequences (
    user_type INTEGER PRIMARY KEY,                  -- 用户类型（1=临时用户，2=正式用户，3=系统用户）
    sequence_name TEXT NOT NULL,                   -- 对应的序列名
    description TEXT NOT NULL,                     -- 描述
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 插入用户类型序列配置
INSERT INTO user_id_sequences (user_type, sequence_name, description) VALUES
(1, 'temp_user_seq', '临时用户序列'),
(2, 'regular_user_seq', '正式用户序列'),
(3, 'system_user_seq', '系统用户序列');

-- 创建用户ID序列
CREATE SEQUENCE temp_user_seq START 1;
CREATE SEQUENCE regular_user_seq START 1;
CREATE SEQUENCE system_user_seq START 1;

-- 生成唯一标识符函数（UUID）
CREATE OR REPLACE FUNCTION generate_unique_id() RETURNS TEXT AS $$
BEGIN
    RETURN uuid_generate_v4()::TEXT;
END;
$$ LANGUAGE plpgsql;

-- 生成用户ID函数（支持用户类型编码）
CREATE OR REPLACE FUNCTION generate_user_id(user_type INTEGER) RETURNS BIGINT AS $$
DECLARE
    sequence_id BIGINT;
    type_prefix BIGINT;
    sequence_name TEXT;
BEGIN
    -- 获取对应的序列名
    SELECT us.sequence_name INTO sequence_name
    FROM user_id_sequences us
    WHERE us.user_type = generate_user_id.user_type;
    
    IF sequence_name IS NULL THEN
        RAISE EXCEPTION '无效的用户类型: %', user_type;
    END IF;
    
    -- 获取序列号（56位）
    EXECUTE format('SELECT nextval(%L)', sequence_name) INTO sequence_id;
    
    -- 用户类型左移56位作为前缀（高8位）
    type_prefix := user_type::BIGINT << 56;
    
    -- 合并类型前缀和序列号
    RETURN type_prefix | sequence_id;
END;
$$ LANGUAGE plpgsql;

-- 获取用户类型函数
CREATE OR REPLACE FUNCTION get_user_type(user_id BIGINT) RETURNS INTEGER AS $$
BEGIN
    RETURN (user_id >> 56)::INTEGER;
END;
$$ LANGUAGE plpgsql;

-- 验证用户ID有效性
CREATE OR REPLACE FUNCTION is_valid_user_id(user_id BIGINT) RETURNS BOOLEAN AS $$
DECLARE
    user_type INTEGER;
BEGIN
    user_type := get_user_type(user_id);
    RETURN user_type IN (1, 2, 3);
END;
$$ LANGUAGE plpgsql;

-- 提取用户ID基础序号（去除类型编码）
CREATE OR REPLACE FUNCTION extract_user_base_id(user_id BIGINT) RETURNS BIGINT AS $$
BEGIN
    -- 清除高8位，保留低56位
    RETURN user_id & ((1::BIGINT << 56) - 1);
END;
$$ LANGUAGE plpgsql;

-- 生成分区表名函数
CREATE OR REPLACE FUNCTION generate_partition_name(base_table TEXT, partition_key TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN base_table || '_' || partition_key;
END;
$$ LANGUAGE plpgsql;

-- 获取当前时间戳（毫秒）
CREATE OR REPLACE FUNCTION current_timestamp_ms() RETURNS BIGINT AS $$
BEGIN
    RETURN EXTRACT(EPOCH FROM NOW()) * 1000;
END;
$$ LANGUAGE plpgsql;

-- 软删除函数（通用）
CREATE OR REPLACE FUNCTION soft_delete_record(table_name TEXT, record_id TEXT) RETURNS BOOLEAN AS $$
DECLARE
    sql_query TEXT;
    rows_affected INTEGER;
BEGIN
    -- 构造动态SQL
    sql_query := format('UPDATE %I SET is_deleted = true, deleted_at = NOW() WHERE id = %L AND is_deleted = false', 
                       table_name, record_id);
    
    -- 执行更新
    EXECUTE sql_query;
    
    -- 获取影响的行数
    GET DIAGNOSTICS rows_affected = ROW_COUNT;
    
    RETURN rows_affected > 0;
END;
$$ LANGUAGE plpgsql;

-- 恢复软删除记录函数
CREATE OR REPLACE FUNCTION restore_soft_deleted_record(table_name TEXT, record_id TEXT) RETURNS BOOLEAN AS $$
DECLARE
    sql_query TEXT;
    rows_affected INTEGER;
BEGIN
    -- 构造动态SQL
    sql_query := format('UPDATE %I SET is_deleted = false, deleted_at = NULL WHERE id = %L AND is_deleted = true', 
                       table_name, record_id);
    
    -- 执行更新
    EXECUTE sql_query;
    
    -- 获取影响的行数
    GET DIAGNOSTICS rows_affected = ROW_COUNT;
    
    RETURN rows_affected > 0;
END;
$$ LANGUAGE plpgsql;

-- 计算两点间距离函数（使用PostGIS）
CREATE OR REPLACE FUNCTION calculate_distance(lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION, 
                                            lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION) 
RETURNS DOUBLE PRECISION AS $$
BEGIN
    RETURN ST_Distance(
        ST_GeogFromText('POINT(' || lon1 || ' ' || lat1 || ')'),
        ST_GeogFromText('POINT(' || lon2 || ' ' || lat2 || ')')
    );
END;
$$ LANGUAGE plpgsql;

-- 添加函数注释
COMMENT ON FUNCTION generate_unique_id() IS '生成UUID格式的唯一标识符';
COMMENT ON FUNCTION generate_user_id(INTEGER) IS '生成带类型编码的用户ID，高8位表示用户类型，低56位为序列号';
COMMENT ON FUNCTION get_user_type(BIGINT) IS '从用户ID中提取用户类型（1=临时，2=正式，3=系统）';
COMMENT ON FUNCTION is_valid_user_id(BIGINT) IS '验证用户ID是否有效';
COMMENT ON FUNCTION extract_user_base_id(BIGINT) IS '提取用户ID的基础序号（去除类型编码）';
COMMENT ON FUNCTION soft_delete_record(TEXT, TEXT) IS '软删除指定表的记录';
COMMENT ON FUNCTION restore_soft_deleted_record(TEXT, TEXT) IS '恢复软删除的记录';
COMMENT ON FUNCTION calculate_distance(DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION) IS '计算两个地理坐标点之间的距离（米）';
