-- allow-destructive: 系统公告通知改虚拟模型：旧 fanout 行迁移为已读状态后删除，避免与新模型重复。
--
-- 系统公告原实现为「发布时向每位用户写一行 notifications」（100k 用户 =
-- 100k 行/公告），且 fanout 非原子、读取侧无法区分已读与公告本体。改为：
--   * system_announcements 作为公告本体（已存在）；
--   * announcement_reads 记录 (announcement_id, user_id) 已读状态（稀疏）；
--   * notify/fetch 读取时把公告虚拟成 system_announcement 通知行
--     （notify_id = 'sysann_' || announcement_id）。
-- 可见性口径与旧 fanout 对齐：仅公告创建时已注册的用户可见（读取侧 join
-- users.created_at），新用户不会被历史公告「补发」。
CREATE TABLE IF NOT EXISTS announcement_reads (
  announcement_id TEXT NOT NULL REFERENCES system_announcements(announcement_id) ON DELETE CASCADE,
  user_id         TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  read_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (announcement_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_user
  ON announcement_reads (user_id, announcement_id);

-- 旧 fanout 行中已读的状态迁入新表（未读无需迁移：虚拟模型默认未读）；随后删除
-- 全部旧 system_announcement 通知行，避免与新虚拟行重复。
INSERT INTO announcement_reads (announcement_id, user_id, read_at)
SELECT n.payload->>'announcement_id', n.user_id, n.created_at
FROM notifications n
WHERE n.type = 'system_announcement'
  AND n.is_read = TRUE
  AND n.payload ? 'announcement_id'
  AND n.payload->>'announcement_id' <> ''
ON CONFLICT DO NOTHING;

DELETE FROM notifications WHERE type = 'system_announcement';
