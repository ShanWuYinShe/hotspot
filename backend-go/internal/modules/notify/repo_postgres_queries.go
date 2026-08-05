package notify

const (
	queryListNotifications = `
SELECT notification_id, type, is_read, created_at, payload::text
FROM notifications
WHERE user_id = $1
  AND ($2 = '' OR type = $2)
  AND ($3 IS NULL OR is_read = $3)
  AND ($4 IS NULL OR created_at < $4 OR (created_at = $4 AND notification_id < $5))
ORDER BY created_at DESC, notification_id DESC
LIMIT $6`

	queryInsertNotification = `
INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)`

	// queryInsertNotificationsBatch fans out one notification to many users in
	// a single round trip. Each row gets its own notification_id (the table's
	// PK is notification_id alone, so a shared id would collide on the 2nd
	// user); the prefix is concatenated with the row ordinal to keep ids unique
	// while remaining traceable to the same logical event.
	queryInsertNotificationsBatch = `
WITH target AS (
  SELECT uid, row_number() OVER () AS rn
  FROM unnest($1::text[]) AS uid
)
INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload)
SELECT $2 || '_' || rn, uid, $3, $4, $5, $6::jsonb
FROM target`

	queryUnreadNotificationCounts = `
SELECT type, COUNT(*)::text
FROM notifications
WHERE user_id = $1 AND is_read = FALSE
GROUP BY type`

	queryMarkNotificationsRead = `
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND notification_id = ANY($2)`

	queryMarkAllNotificationsRead = `
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND is_read = FALSE`
)
