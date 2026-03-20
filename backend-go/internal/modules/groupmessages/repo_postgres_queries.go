package groupmessages

const (
	queryGroupMessagesHistory = `
SELECT message_id, type, content, sent_at
FROM group_messages
WHERE group_id = $1
ORDER BY sent_at DESC, message_id DESC`
)
