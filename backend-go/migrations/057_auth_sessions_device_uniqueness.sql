-- 057_auth_sessions_device_uniqueness.sql
-- 设备会话唯一性（TOCTOU 修复，审计「未修限制 1」）：
-- 原 Login/Guest 是「先撤销旧会话再建新会话」两步操作，同一设备两个登录请求
-- 严格并发时可双双通过撤销检查、各自插入新会话——同一设备留下两个活跃会话，
-- 旧会话的 refresh token 仍可轮转，与「单设备登录即逐出」的产品语义冲突。
--
-- 修法：部分唯一索引约束「同一 (user_id, device_id) 至多一个活跃会话」
-- （device_id 为空的历史/匿名会话不参与约束）；应用层改为「先插新会话，冲突则
-- 撤销旧会话后重试」，由数据库保证最终只有一个活跃会话。
--
-- 幂等性：先做一次性去重（保留每个用户+设备最新会话，只加不删——仅置
-- revoked_at，不删行），重复执行时已无重复可改；索引 IF NOT EXISTS。
-- 向后兼容：旧版二进制在竞态窗口内插入重复会话会收到 23505 报错（此前该竞态
-- 会静默留下双会话），部署切换期属可接受；索引创建用普通 CREATE INDEX（迁移
-- 在隐式事务内执行，禁止 CONCURRENTLY，SHARE 锁仅在启动迁移瞬间持有）。

-- 1) 存量去重：同一 (user_id, device_id) 多个活跃会话时，仅保留
--    (created_at, session_id) 最大者，其余置为已撤销。
UPDATE auth_sessions AS s
SET revoked_at = NOW()
WHERE s.revoked_at IS NULL
  AND s.device_id <> ''
  AND EXISTS (
    SELECT 1
    FROM auth_sessions AS keep
    WHERE keep.user_id = s.user_id
      AND keep.device_id = s.device_id
      AND keep.revoked_at IS NULL
      AND (keep.created_at, keep.session_id) > (s.created_at, s.session_id)
  );

-- 2) 活跃会话的设备唯一约束（空 device_id 不参与：未知设备/历史数据）。
CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_sessions_active_device
  ON auth_sessions (user_id, device_id)
  WHERE revoked_at IS NULL AND device_id <> '';
