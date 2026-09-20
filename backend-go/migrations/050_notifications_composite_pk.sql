-- 系统公告 WS notify_id 与落库主键一致性修复（审计 F-06）。
--
-- 契约（spec api_spec_websocket.md → 「消息可靠性与去重」；api_spec_notify.md
-- → 「通知与 WebSocket 事件的关系」）：携带 notify_id 的 WS 事件，其
-- notify_id 必须与 HTTP notify/fetch 返回的 NotifyItem.notify_id 为**同一主键**，
-- 客户端按 notify_id 去重并调 notify/mark_read(scope='ids') 标记已读。
--
-- 原实现：notifications 的主键只有 notification_id，而「一条公告 fanout 给
-- N 个用户」必须让每行主键不同，于是批量插入用 `$2 || '_' || row_number()` 派生
-- 逐行 id；服务层却只把**基础 id**（不含后缀）作为 WS notify_id 下发。结果：
--   * 客户端拿 WS 的 notify_id 去 mark_read，库里没有该主键 → 已读更新落空；
--   * 按 notify_id 去重对公告失效（同一公告每行 id 都不同）。
--
-- 治法：把主键改为复合主键 (notification_id, user_id)。notification_id 表达
-- 「这一条逻辑通知」，user_id 表达「发给谁」，同一逻辑通知的所有收件人共用
-- 同一个 notification_id——这既满足契约的「同一主键」，也让 mark_read 与去重
-- 天然成立。单收件人通知（关注/评论/@/reaction 等）行为不变。
--
-- 幂等：重复执行时先判断主键是否已为复合主键。

DO $$
DECLARE
  pk_cols text[];
BEGIN
  SELECT array_agg(a.attname ORDER BY array_position(i.indkey, a.attnum))
  INTO pk_cols
  FROM pg_index i
  JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
  WHERE i.indrelid = 'notifications'::regclass AND i.indisprimary;

  -- 已是复合主键（notification_id, user_id）则跳过。
  IF pk_cols = ARRAY['notification_id', 'user_id'] THEN
    RETURN;
  END IF;

  ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_pkey;

  -- 历史数据里公告行带有 '_<row_number>' 后缀，先把同一公告的重复行收敛：
  -- 同 (user_id, type, payload) 且 created_at 相同的行只保留一行，并把
  -- notification_id 截断为去掉后缀的基础 id，保证复合主键不冲突。
  UPDATE notifications
  SET notification_id = regexp_replace(notification_id, '_[0-9]+$', '')
  WHERE type = 'system_announcement' AND notification_id ~ '_[0-9]+$';

  DELETE FROM notifications a
  USING notifications b
  WHERE a.type = 'system_announcement'
    AND b.type = 'system_announcement'
    AND a.notification_id = b.notification_id
    AND a.user_id = b.user_id
    AND a.ctid < b.ctid;

  ALTER TABLE notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (notification_id, user_id);
END $$;

-- notify/mark_read 与列表查询都按 (user_id, notification_id) 过滤，
-- 复合主键的前缀是 notification_id，需为 user_id 侧补索引（列表分页已由
-- 005 的 idx_notifications_user_created_at 承担，这里补按 id 定位已读）。
CREATE INDEX IF NOT EXISTS idx_notifications_user_notification
  ON notifications(user_id, notification_id);
