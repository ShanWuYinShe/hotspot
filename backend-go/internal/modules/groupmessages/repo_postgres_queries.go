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

	queryGroupMessageReadStateByMessage = `
SELECT gm.sent_at, gm.message_id
FROM group_messages gm
JOIN group_members members ON members.group_id = gm.group_id AND members.user_id = $1
WHERE gm.group_id = $2 AND gm.message_id = $3`

	queryUpsertGroupMessageReadState = `
INSERT INTO group_message_read_state (group_id, user_id, last_read_message_id, last_read_sent_at, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (group_id, user_id) DO UPDATE
SET last_read_message_id = EXCLUDED.last_read_message_id,
    last_read_sent_at = EXCLUDED.last_read_sent_at,
    updated_at = NOW()
WHERE group_message_read_state.last_read_sent_at IS NULL
   OR group_message_read_state.last_read_sent_at <= EXCLUDED.last_read_sent_at`

	queryMarkGroupMessagesReadToLatest = `
INSERT INTO group_message_read_state (group_id, user_id, last_read_message_id, last_read_sent_at, updated_at)
SELECT members.group_id, members.user_id, latest.message_id, latest.sent_at, NOW()
FROM group_members members
JOIN LATERAL (
    SELECT gm.message_id, gm.sent_at
    FROM group_messages gm
    WHERE gm.group_id = members.group_id
    ORDER BY gm.sent_at DESC, gm.message_id DESC
    LIMIT 1
) latest ON TRUE
WHERE members.user_id = $1 AND members.group_id = $2
ON CONFLICT (group_id, user_id) DO UPDATE
SET last_read_message_id = EXCLUDED.last_read_message_id,
    last_read_sent_at = EXCLUDED.last_read_sent_at,
    updated_at = NOW()
WHERE group_message_read_state.last_read_sent_at IS NULL
   OR group_message_read_state.last_read_sent_at <= EXCLUDED.last_read_sent_at`

	queryMarkAllGroupMessagesRead = `
INSERT INTO group_message_read_state (group_id, user_id, last_read_message_id, last_read_sent_at, updated_at)
SELECT members.group_id, members.user_id, latest.message_id, latest.sent_at, NOW()
FROM group_members members
JOIN LATERAL (
    SELECT gm.message_id, gm.sent_at
    FROM group_messages gm
    WHERE gm.group_id = members.group_id
    ORDER BY gm.sent_at DESC, gm.message_id DESC
    LIMIT 1
) latest ON TRUE
WHERE members.user_id = $1
ON CONFLICT (group_id, user_id) DO UPDATE
SET last_read_message_id = EXCLUDED.last_read_message_id,
    last_read_sent_at = EXCLUDED.last_read_sent_at,
    updated_at = NOW()
WHERE group_message_read_state.last_read_sent_at IS NULL
   OR group_message_read_state.last_read_sent_at <= EXCLUDED.last_read_sent_at`
)
