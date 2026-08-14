-- 036: 群组加入方式重设计
-- join_method 从 'open' | 'approval' | 'invite_only' 拆为：
--   准入方式 join_method：'direct' | 'password' | 'invite_only'
--   + 审核开关 requires_approval（审核通过后才算真正加入）
-- 另新增 next_activity_at（下次活动预告，群主通过 update 设置；契约改名对齐列名）
ALTER TABLE groups ADD COLUMN IF NOT EXISTS requires_approval BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS next_activity_at TEXT;

-- 存量迁移：
--   'open' + has_password=true  → 'password'（需密码才能加入）
--   'open' + has_password=false → 'direct'（直接加入）
--   'approval'                  → 'direct' + requires_approval=true（申请制）
UPDATE groups SET join_method = 'password' WHERE join_method = 'open' AND has_password = TRUE;
UPDATE groups SET join_method = 'direct' WHERE join_method = 'open' AND has_password = FALSE;
UPDATE groups SET join_method = 'direct', requires_approval = TRUE WHERE join_method = 'approval';

ALTER TABLE groups ALTER COLUMN join_method SET DEFAULT 'direct';
