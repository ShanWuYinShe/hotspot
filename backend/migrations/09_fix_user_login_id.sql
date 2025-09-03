-- 修复用户表login_id字段，允许临时用户为NULL
-- 临时用户不需要login_id，只有注册用户才需要

-- 移除login_id的NOT NULL约束
ALTER TABLE users ALTER COLUMN login_id DROP NOT NULL;

-- 添加用户类型字段来区分临时用户和注册用户
ALTER TABLE users ADD COLUMN IF NOT EXISTS user_type VARCHAR(20) NOT NULL DEFAULT 'temporary';

-- 更新现有数据：有login_id的设为registered，没有的设为temporary
UPDATE users SET user_type = 'registered' WHERE login_id IS NOT NULL;
UPDATE users SET user_type = 'temporary' WHERE login_id IS NULL;

-- 添加约束：注册用户必须有login_id，临时用户不能有login_id
ALTER TABLE users ADD CONSTRAINT check_user_type_login_id 
CHECK (
    (user_type = 'temporary' AND login_id IS NULL) OR 
    (user_type = 'registered' AND login_id IS NOT NULL)
);

-- 更新索引，login_id可能为NULL
-- 先删除约束，再重新创建索引
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_login_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_login_id_unique ON users(login_id) WHERE login_id IS NOT NULL;

-- 添加注释
COMMENT ON COLUMN users.user_type IS '用户类型：temporary-临时用户，registered-注册用户';
COMMENT ON COLUMN users.login_id IS '登录ID，仅注册用户需要，临时用户为NULL';
