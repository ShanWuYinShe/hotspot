package privatechats

const (
	queryInsertPrivateChat = `
INSERT INTO private_chats (chat_key, user_a_id, user_b_id, created_at)
VALUES ($1,$2,$3,$4)
ON CONFLICT (chat_key) DO NOTHING`

	queryGetPrivateChatCreatedAt = `
SELECT created_at
FROM private_chats
WHERE chat_key = $1`

	queryListPrivateChats = `
SELECT chat_key, created_at
FROM private_chats
WHERE user_a_id = $1 OR user_b_id = $1
ORDER BY created_at DESC, chat_key ASC`
)
