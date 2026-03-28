package private_messages

const (
	queryInsertPrivateChat = `
INSERT INTO private_chats (chat_key, user_a_id, user_b_id, created_at)
VALUES ($1,$2,$3,$4)
ON CONFLICT (chat_key) DO NOTHING`

	queryInsertPrivateMessage = `
INSERT INTO private_messages (message_id, chat_key, sender_user_id, type, content, share_code, sent_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)`

	queryPrivateMessagesHistory = `
SELECT message_id, type, content, share_code, sender_user_id, sent_at
FROM private_messages
WHERE chat_key = $1
ORDER BY sent_at DESC, message_id DESC`

	queryUpdatePrivateMessage = `
UPDATE private_messages
SET content = $2, content_prev = content, edited_at = NOW()
WHERE message_id = $1 AND is_recalled = FALSE`

	queryRecallPrivateMessage = `
UPDATE private_messages
SET is_recalled = TRUE, recalled_at = NOW()
WHERE message_id = $1`
)
