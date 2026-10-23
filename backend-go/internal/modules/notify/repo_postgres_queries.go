package notify

// 表/列名对齐 migrations/005_notifications.sql：
// notifications(notification_id, user_id, type, is_read, created_at, payload)。
// 此前查询误写为 notifys/notify_id，接 Postgres 后全部报
// relation "notifys" does not exist 且被 service 吞错，通知功能静默失效。
const (
	queryListNotifys = `
SELECT notification_id, type, is_read, created_at, payload::text
FROM notifications
WHERE user_id = $1
  AND ($2 = '' OR type = $2)
  AND ($3::boolean IS NULL OR is_read = $3::boolean)
  AND ($4::timestamptz IS NULL OR created_at < $4::timestamptz OR (created_at = $4::timestamptz AND notification_id < $5))
ORDER BY created_at DESC, notification_id DESC
LIMIT $6`

	queryInsertNotify = `
INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)`

	// queryInsertNotifysBatch fans out one notify to many users in a single
	// round trip。主键为 (notification_id, user_id)（050 迁移），因此同一逻辑
	// 通知的所有收件人**共用同一个 notification_id**——这是契约要求
	// （spec api_spec_websocket.md「消息可靠性与去重」：WS 的 notify_id 与
	// notify/fetch 的 NotifyItem.notify_id 为同一主键，客户端据此去重与
	// mark_read）。此前每行用 `$2 || '_' || row_number()` 造独立主键，服务层
	// 却只下发不含后缀的基础 id，导致 mark_read 落空、公告去重失效（审计 F-06）。
	queryInsertNotifysBatch = `
INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload)
SELECT $2, uid, $3, $4, $5, $6::jsonb
FROM unnest($1::text[]) AS uid
ON CONFLICT (notification_id, user_id) DO NOTHING`

	// queryInsertNotifysBatchPerUser 与 queryInsertNotifysBatch 的区别：每行
	// 携带**各自独立**的 notify_id 与 payload（@ 提及场景：payload 内嵌各自
	// 的 notify_id，语义与逐人 Append 一致），六个同长数组按位 unnest 成行、
	// 单次往返插入。created_at 以 RFC3339 文本传入、显式 cast 为 timestamptz
	// （unnest 产出的 text 不会隐式转换）。主键 (notification_id, user_id)
	// 冲突时跳过（与批量公告一致）。
	queryInsertNotifysBatchPerUser = `
INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload)
SELECT unnest($1::text[]), unnest($2::text[]), unnest($3::text[]), unnest($4::bool[]), unnest($5::text[])::timestamptz, unnest($6::jsonb[])
ON CONFLICT (notification_id, user_id) DO NOTHING`

	queryUnreadNotifyCounts = `
SELECT type, COUNT(*)::text
FROM notifications
WHERE user_id = $1 AND is_read = FALSE
GROUP BY type`

	queryMarkNotifysRead = `
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND notification_id = ANY($2)`

	queryMarkAllNotifysRead = `
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND is_read = FALSE`

	// queryPruneReadNotifysBatch 分批物理删除已读且创建早于截止时间的通知
	// （后台 prune sweeper 专用，见 Service.StartPruneSweeper）。notifications
	// 无 updated_at 列，以 created_at 近似已读年龄（保守：只会少删）。
	// 子查询 LIMIT 封顶单事务体量，防存量积压时长事务持锁 + WAL 尖刺（审查 M-4）。
	// IN 谓词必须用复合主键 (notification_id, user_id)：主键已非 notification_id
	// 单列（050 迁移），单列 IN 会因子查询返回重复 id 使一条 fanout >LIMIT 的
	// 公告一次性全删（N 行），分批封顶失效（P2-7）。
	queryPruneReadNotifysBatch = `
DELETE FROM notifications
WHERE (notification_id, user_id) IN (
  SELECT notification_id, user_id FROM notifications
  WHERE is_read = TRUE AND created_at < $1
  LIMIT $2
)
RETURNING notification_id`
)
