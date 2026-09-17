-- 053: auth_refresh_tokens 哈希化 + 宽限托管列
--
-- refresh_token / rotated_to 从明文改为 SHA-256 hex：DB 或备份泄漏不再
-- 等价于全体会话可劫持（此前 refresh token 为可登录长效凭证，明文落库）。
-- 明文与哈希同为 64 位 hex，无法按格式判幂等（迁移每次启动重跑），故用
-- data_migrations 旗标表保证数据迁移只执行一次。
--
-- 并发刷新的宽限期语义（spec api_spec_auth.md「并发刷新策略」）要求把
-- 新链 token 明文回给并发败者，哈希不可逆——新增 grace_token_enc
-- （AES-256-GCM 密文，密钥取自 AUTH_TOKEN_SECRET 派生、不在 DB），
-- 行随既有 sweeper 在宽限期后清除。存量宽限行（迁移瞬间正在宽限期的）
-- 无托管密文，按无效处理，最坏情况是该 30s 窗口内的并发刷新方需重新登录。

CREATE TABLE IF NOT EXISTS data_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM data_migrations WHERE name = 'refresh_token_hash_v1') THEN
        UPDATE auth_refresh_tokens
        SET refresh_token = encode(sha256(refresh_token::bytea), 'hex');
        UPDATE auth_refresh_tokens
        SET rotated_to = encode(sha256(rotated_to::bytea), 'hex')
        WHERE rotated_to IS NOT NULL;
        ALTER TABLE auth_refresh_tokens ADD COLUMN IF NOT EXISTS grace_token_enc BYTEA NULL;
        INSERT INTO data_migrations(name) VALUES ('refresh_token_hash_v1');
    END IF;
END $$;
