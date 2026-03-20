package notifications

const (
	queryListNotifications = `
SELECT notification_id, type, is_read, created_at
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC, notification_id DESC`
)
