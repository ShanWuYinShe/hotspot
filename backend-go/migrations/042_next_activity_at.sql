-- 契约改名（spec ce5d42ed）：GroupItem.next_activity → next_activity_at。
-- 存量库旧列 next_activity 改名保数据；036 重放已按新列名建列时，丢弃旧列
-- （仅出现在旧列已被 042 改名后的二次重放场景，此时旧列必为空列）。
-- 迁移为无状态重放设计（ApplyFS 每次 startup 全量执行），必须幂等。
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'groups' AND column_name = 'next_activity'
  ) THEN
    IF EXISTS (
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = 'groups' AND column_name = 'next_activity_at'
    ) THEN
      ALTER TABLE groups DROP COLUMN next_activity;
    ELSE
      ALTER TABLE groups RENAME COLUMN next_activity TO next_activity_at;
    END IF;
  END IF;
END $$;
