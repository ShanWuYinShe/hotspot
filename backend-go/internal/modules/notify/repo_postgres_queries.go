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
  AND ($3 IS NULL OR is_read = $3)
  AND ($4 IS NULL OR created_at < $4 OR (created_at = $4 AND notification_id < $5))
ORDER BY created_at DESC, notification_id DESC
LIMIT $6`

	queryInsertNotify = `
INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)`

	// queryInsertNotifysBatch fans out one notify to many users in
	// a single round trip. Each row gets its own notification_id (the table's
	// PK is notification_id alone, so a shared id would collide on the 2nd
	// user); the prefix is concatenated with the row ordinal to keep ids unique
	// while remaining traceable to the same logical event.
	queryInsertNotifysBatch = `
WITH target AS (
  SELECT uid, row_number() OVER () AS rn
  FROM unnest($1::text[]) AS uid
)
INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload)
SELECT $2 || '_' || rn, uid, $3, $4, $5, $6::jsonb
FROM target`

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
)
