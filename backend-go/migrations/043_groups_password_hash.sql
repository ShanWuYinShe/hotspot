-- 群密码真实化：has_password 布尔不足以支撑 join_method='password' 的入群校验，
-- 增加 password_hash（bcrypt）。存量 password 群没有可用密码材料，统一退化为
-- direct（无历史兼容分支；管理员可重新设置密码恢复 password 语义）。
ALTER TABLE groups ADD COLUMN IF NOT EXISTS password_hash TEXT;

UPDATE groups
SET join_method = 'direct', has_password = FALSE
WHERE join_method = 'password' AND password_hash IS NULL;
