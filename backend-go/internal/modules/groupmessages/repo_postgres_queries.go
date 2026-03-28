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

	queryUpdateGroupMessage = `
UPDATE group_messages
SET content = $2, content_prev = content, edited_at = NOW()
WHERE message_id = $1 AND is_recalled = FALSE`

	queryRecallGroupMessage = `
UPDATE group_messages
SET is_recalled = TRUE, recalled_at = NOW()
WHERE message_id = $1`
)
