package private_messages

const (
	queryPrivateMessagesHistory = `
SELECT message_id, type, content, share_code, sender_user_id, sent_at
FROM private_messages
WHERE chat_key = $1
ORDER BY sent_at DESC, message_id DESC`
)
