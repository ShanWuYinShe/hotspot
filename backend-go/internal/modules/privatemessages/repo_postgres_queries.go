package private_messages

const (
	queryInsertPrivateChat = `
		INSERT INTO private_chats (chat_key, user_a_id, user_b_id, created_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (chat_key) DO NOTHING`

	queryInsertPrivateMessage = `
		INSERT INTO private_messages (message_id, chat_key, sender_user_id, type, content, file_id, share_code, reply_to_message_id, sent_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`

	queryPrivateMessagesHistory = `
		SELECT pm.message_id,
		       pm.type,
		       COALESCE(pm.content, ''),
		       COALESCE(pm.file_id, ''),
		       COALESCE(pm.share_code, ''),
		       pm.sender_user_id,
		       sender.nickname,
		       COALESCE(sender.avatar_url, ''),
		       COALESCE(reply.message_id, ''),
		       COALESCE(reply.type, ''),
		       COALESCE(reply.content, ''),
		       COALESCE(reply.share_code, ''),
		       COALESCE(reply.sender_user_id, ''),
		       COALESCE(reply_sender.nickname, ''),
		       COALESCE(reply_sender.avatar_url, ''),
		       COALESCE(TO_CHAR(reply.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
		       TO_CHAR(pm.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
		       COALESCE(TO_CHAR(pm.edited_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
		       COALESCE(TO_CHAR(pm.recalled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
		       CASE
		         WHEN pm.sender_user_id <> $2 THEN TRUE
		         WHEN prs.last_read_sent_at IS NULL THEN FALSE
		         WHEN pm.sent_at < prs.last_read_sent_at THEN TRUE
		         WHEN pm.sent_at > prs.last_read_sent_at THEN FALSE
		         WHEN prs.last_read_message_id IS NULL THEN FALSE
		         ELSE pm.message_id = prs.last_read_message_id
		       END AS is_read
		FROM private_messages pm
		JOIN users sender ON sender.user_id = pm.sender_user_id
		LEFT JOIN private_messages reply ON reply.message_id = pm.reply_to_message_id
		LEFT JOIN users reply_sender ON reply_sender.user_id = reply.sender_user_id
		LEFT JOIN private_chat_read_state prs
		  ON prs.chat_key = pm.chat_key AND prs.user_id = $3
		WHERE pm.chat_key = $1
		  AND ($4 IS NULL OR pm.sent_at < $4 OR (pm.sent_at = $4 AND pm.message_id < $5))
		ORDER BY pm.sent_at DESC, pm.message_id DESC
		LIMIT $6`

	queryPrivateMessageReadStateByMessage = `
		SELECT pm.sent_at, pm.message_id
		FROM private_messages pm
		WHERE pm.chat_key = $1 AND pm.message_id = $2`

	queryUpsertPrivateMessageReadState = `
		INSERT INTO private_chat_read_state (chat_key, user_id, last_read_message_id, last_read_sent_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (chat_key, user_id) DO UPDATE
		SET last_read_message_id = EXCLUDED.last_read_message_id,
		    last_read_sent_at = EXCLUDED.last_read_sent_at,
		    updated_at = NOW()
		WHERE private_chat_read_state.last_read_sent_at IS NULL
		   OR EXCLUDED.last_read_sent_at > private_chat_read_state.last_read_sent_at
		   OR (EXCLUDED.last_read_sent_at = private_chat_read_state.last_read_sent_at
		       AND EXCLUDED.last_read_message_id > private_chat_read_state.last_read_message_id)`

	queryMarkAllPrivateMessagesReadForChat = `
		WITH latest AS (
		  SELECT pm.message_id, pm.sent_at
		  FROM private_messages pm
		  WHERE pm.chat_key = $1
		  ORDER BY pm.sent_at DESC, pm.message_id DESC
		  LIMIT 1
		)
		INSERT INTO private_chat_read_state (chat_key, user_id, last_read_message_id, last_read_sent_at, updated_at)
		SELECT $1, $2, latest.message_id, latest.sent_at, NOW()
		FROM latest
		ON CONFLICT (chat_key, user_id) DO UPDATE
		SET last_read_message_id = EXCLUDED.last_read_message_id,
		    last_read_sent_at = EXCLUDED.last_read_sent_at,
		    updated_at = NOW()
		WHERE private_chat_read_state.last_read_sent_at IS NULL
		   OR EXCLUDED.last_read_sent_at > private_chat_read_state.last_read_sent_at
		   OR (EXCLUDED.last_read_sent_at = private_chat_read_state.last_read_sent_at
		       AND EXCLUDED.last_read_message_id > private_chat_read_state.last_read_message_id)`

	queryMarkAllPrivateMessagesRead = `
		WITH latest AS (
		  SELECT DISTINCT ON (pm.chat_key)
		         pm.chat_key,
		         pm.message_id,
		         pm.sent_at
		  FROM private_messages pm
		  JOIN private_chats pc ON pc.chat_key = pm.chat_key
		  WHERE pc.user_a_id = $1 OR pc.user_b_id = $1
		  ORDER BY pm.chat_key, pm.sent_at DESC, pm.message_id DESC
		)
		INSERT INTO private_chat_read_state (chat_key, user_id, last_read_message_id, last_read_sent_at, updated_at)
		SELECT latest.chat_key, $1, latest.message_id, latest.sent_at, NOW()
		FROM latest
		ON CONFLICT (chat_key, user_id) DO UPDATE
		SET last_read_message_id = EXCLUDED.last_read_message_id,
		    last_read_sent_at = EXCLUDED.last_read_sent_at,
		    updated_at = NOW()
		WHERE private_chat_read_state.last_read_sent_at IS NULL
		   OR EXCLUDED.last_read_sent_at > private_chat_read_state.last_read_sent_at
		   OR (EXCLUDED.last_read_sent_at = private_chat_read_state.last_read_sent_at
		       AND EXCLUDED.last_read_message_id > private_chat_read_state.last_read_message_id)`

	queryUpdatePrivateMessage = `
		UPDATE private_messages
		SET content = $2, content_prev = content, edited_at = NOW()
		WHERE message_id = $1 AND is_recalled = FALSE`

	queryRecallPrivateMessage = `
		UPDATE private_messages
		SET is_recalled = TRUE, recalled_at = NOW()
		WHERE message_id = $1 AND is_recalled = FALSE`

	queryGetPrivateMessageMeta = `
		SELECT chat_key,
		       sender_user_id,
		       type,
		       COALESCE(is_recalled, FALSE),
		       TO_CHAR(sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
		FROM private_messages
		WHERE message_id = $1`
)
