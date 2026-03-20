package privatechats

const (
	queryListPrivateChats = `
SELECT chat_key, created_at
FROM private_chats
WHERE user_a_id = $1 OR user_b_id = $1
ORDER BY created_at DESC, chat_key ASC`
)
