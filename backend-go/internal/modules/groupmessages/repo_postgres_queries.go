package groupmessages

const (
	queryInsertGroupMessage = `
INSERT INTO group_messages (message_id, group_id, sender_user_id, type, content, sent_at)
VALUES ($1,$2,$3,$4,$5,$6)`

	queryGroupMessagesHistory = `
SELECT message_id, type, content, sent_at
FROM group_messages
WHERE group_id = $1
ORDER BY sent_at DESC, message_id DESC`
)
