package groupmessages

const (
	queryInsertGroupMessage = `
	INSERT INTO group_messages (message_id, group_id, sender_user_id, type, content, sent_at)
	VALUES ($1,$2,$3,$4,$5,$6)`

	queryGroupMessagesHistory = `
	SELECT gm.message_id,
	       gm.type,
	       COALESCE(gm.content, ''),
	       TO_CHAR(gm.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
	       COALESCE(TO_CHAR(gm.edited_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
	       COALESCE(TO_CHAR(gm.recalled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
	       COALESCE(gm.sender_user_id, ''),
	       COALESCE(sender.nickname, ''),
	       COALESCE(sender.avatar_url, ''),
	       COALESCE((
	         SELECT COUNT(*)::int
	         FROM group_message_read_state grs
	         WHERE grs.group_id = gm.group_id AND grs.last_read_sent_at >= gm.sent_at
	       ), 0) AS read_count
	FROM group_messages gm
	LEFT JOIN users sender ON sender.user_id = gm.sender_user_id
	WHERE gm.group_id = $1
	  AND ($2 IS NULL OR gm.sent_at < $2 OR (gm.sent_at = $2 AND gm.message_id < $3))
	ORDER BY gm.sent_at DESC, gm.message_id DESC
	LIMIT $4`

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

	queryCheckGroupMembership = `
SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2`

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

	queryGetGroupMessageForReply = `
		SELECT gm.sender_user_id,
		       COALESCE(u.nickname, gm.sender_user_id),
		       COALESCE(gm.content, ''),
		       COALESCE(gm.share_code, '')
		FROM group_messages gm
		LEFT JOIN users u ON u.user_id = gm.sender_user_id
		WHERE gm.message_id = $1`

)