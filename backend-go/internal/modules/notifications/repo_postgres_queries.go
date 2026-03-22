package notifications

const (
	queryListNotifications = `
SELECT notification_id, type, is_read, created_at, payload::text
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC, notification_id DESC`

	queryMarkNotificationsRead = `
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND notification_id = ANY($2)`

	queryMarkAllNotificationsRead = `
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1 AND is_read = FALSE`
)
