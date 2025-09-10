-- 手动修复迁移校验和问题
-- 添加pg_trgm扩展
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 更新迁移记录的校验和
UPDATE _sqlx_migrations 
SET checksum = decode('66d6b131feb8479cd15d2ab4227152168ac7089fbf79d8b9a6420cc9ed19cdd7ed167902e56ab6881ccba0b080de9556', 'hex')
WHERE version = 0;

