package notifications

const (
	queryListNotifications = `
SELECT notification_id, type, is_read, created_at, payload::text
FROM notifications
WHERE user_id = $1
  AND ($2 = '' OR type = $2)
  AND ($3 = '' OR created_at < $3 OR (created_at = $3 AND notification_id < $4))
ORDER BY created_at DESC, notification_id DESC
LIMIT $5`

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
