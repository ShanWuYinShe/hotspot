-- ============================================================================
-- HotSpot 用户ID编码规范修复迁移
-- ============================================================================
-- 
-- 功能职责：
-- - 修复用户ID编码规范，使用64位整数，高8位表示用户类型
-- - 添加用户ID编码相关函数
-- - 更新用户表结构以符合需求文档
--
-- 依赖关系：
-- - 依赖现有的用户表结构
--
-- ============================================================================

-- 1. 创建用户ID编码函数
-- 生成带用户类型编码的用户ID
CREATE OR REPLACE FUNCTION generate_user_id(user_type INTEGER) RETURNS BIGINT AS $$
DECLARE
    sequence_id BIGINT;
    type_prefix BIGINT;
BEGIN
    -- 获取序列号（56位）
    sequence_id := nextval('users_id_seq');
    
    -- 用户类型左移56位作为前缀（高8位）
    type_prefix := user_type::BIGINT << 56;
    
    -- 合并类型前缀和序列号
    RETURN type_prefix | sequence_id;
END;
$$ LANGUAGE plpgsql;

-- 2. 获取用户类型函数
CREATE OR REPLACE FUNCTION get_user_type(user_id BIGINT) RETURNS INTEGER AS $$
BEGIN
    -- 右移56位获取高8位的用户类型
    RETURN (user_id >> 56)::INTEGER;
END;
$$ LANGUAGE plpgsql;

-- 3. 验证用户ID格式函数
CREATE OR REPLACE FUNCTION is_valid_user_id(user_id BIGINT) RETURNS BOOLEAN AS $$
DECLARE
    user_type INTEGER;
BEGIN
    user_type := get_user_type(user_id);
    -- 验证用户类型是否在有效范围内（1-3）
    RETURN user_type >= 1 AND user_type <= 3;
END;
$$ LANGUAGE plpgsql;

-- 4. 添加用户ID编码相关的约束和索引
-- 为现有用户表添加用户ID编码支持
ALTER TABLE users ADD COLUMN IF NOT EXISTS encoded_user_id BIGINT;

-- 创建索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_encoded_user_id ON users(encoded_user_id) WHERE encoded_user_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_user_type ON users((get_user_type(encoded_user_id))) WHERE encoded_user_id IS NOT NULL;

-- 5. 创建触发器函数，自动生成编码用户ID
CREATE OR REPLACE FUNCTION auto_generate_encoded_user_id() RETURNS TRIGGER AS $$
DECLARE
    user_type_code INTEGER;
BEGIN
    -- 根据用户类型枚举映射到编码
    CASE NEW.user_type
        WHEN 'temporary' THEN user_type_code := 1;
        WHEN 'regular' THEN user_type_code := 2;
        WHEN 'system' THEN user_type_code := 3;
        ELSE user_type_code := 2; -- 默认为普通用户
    END CASE;
    
    -- 如果没有编码用户ID，则生成一个
    IF NEW.encoded_user_id IS NULL THEN
        NEW.encoded_user_id := generate_user_id(user_type_code);
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 6. 创建触发器
DROP TRIGGER IF EXISTS trigger_auto_generate_encoded_user_id ON users;
CREATE TRIGGER trigger_auto_generate_encoded_user_id
    BEFORE INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION auto_generate_encoded_user_id();

-- 7. 为现有用户生成编码用户ID
UPDATE users 
SET encoded_user_id = generate_user_id(
    CASE user_type
        WHEN 'temporary' THEN 1
        WHEN 'regular' THEN 2
        WHEN 'system' THEN 3
        ELSE 2
    END
)
WHERE encoded_user_id IS NULL;

-- 8. 添加约束确保编码用户ID的唯一性和有效性
ALTER TABLE users ADD CONSTRAINT users_encoded_user_id_unique UNIQUE (encoded_user_id);
ALTER TABLE users ADD CONSTRAINT users_encoded_user_id_valid CHECK (is_valid_user_id(encoded_user_id));

-- 9. 创建视图，方便查询用户类型
CREATE OR REPLACE VIEW users_with_type AS
SELECT 
    *,
    get_user_type(encoded_user_id) as user_type_code,
    CASE get_user_type(encoded_user_id)
        WHEN 1 THEN 'temporary'
        WHEN 2 THEN 'regular'
        WHEN 3 THEN 'system'
        ELSE 'unknown'
    END as user_type_name
FROM users
WHERE encoded_user_id IS NOT NULL;

-- 10. 创建用于API的辅助函数
-- 将编码用户ID转换为字符串（前端使用）
CREATE OR REPLACE FUNCTION user_id_to_string(user_id BIGINT) RETURNS TEXT AS $$
BEGIN
    RETURN user_id::TEXT;
END;
$$ LANGUAGE plpgsql;

-- 将字符串转换为编码用户ID（后端使用）
CREATE OR REPLACE FUNCTION string_to_user_id(user_id_str TEXT) RETURNS BIGINT AS $$
BEGIN
    RETURN user_id_str::BIGINT;
EXCEPTION
    WHEN invalid_text_representation THEN
        RAISE EXCEPTION 'Invalid user ID format: %', user_id_str;
END;
$$ LANGUAGE plpgsql;

-- 11. 添加注释
COMMENT ON FUNCTION generate_user_id(INTEGER) IS '生成带用户类型编码的64位用户ID，高8位为用户类型';
COMMENT ON FUNCTION get_user_type(BIGINT) IS '从64位用户ID中提取用户类型（高8位）';
COMMENT ON FUNCTION is_valid_user_id(BIGINT) IS '验证用户ID格式是否有效';
COMMENT ON COLUMN users.encoded_user_id IS '64位编码用户ID，高8位表示用户类型，低56位表示序号';
COMMENT ON VIEW users_with_type IS '用户表视图，包含解析后的用户类型信息';

-- 12. 创建示例数据验证
DO $$
DECLARE
    temp_user_id BIGINT;
    regular_user_id BIGINT;
    system_user_id BIGINT;
BEGIN
    -- 生成示例用户ID
    temp_user_id := generate_user_id(1);
    regular_user_id := generate_user_id(2);
    system_user_id := generate_user_id(3);
    
    -- 验证用户类型提取
    ASSERT get_user_type(temp_user_id) = 1, '临时用户ID类型提取失败';
    ASSERT get_user_type(regular_user_id) = 2, '普通用户ID类型提取失败';
    ASSERT get_user_type(system_user_id) = 3, '系统用户ID类型提取失败';
    
    -- 验证ID有效性
    ASSERT is_valid_user_id(temp_user_id), '临时用户ID验证失败';
    ASSERT is_valid_user_id(regular_user_id), '普通用户ID验证失败';
    ASSERT is_valid_user_id(system_user_id), '系统用户ID验证失败';
    
    RAISE NOTICE '用户ID编码规范验证通过';
END $$;
