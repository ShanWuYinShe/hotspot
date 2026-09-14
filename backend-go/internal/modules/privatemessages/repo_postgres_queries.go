package privatemessages

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
		       COALESCE(sender.nickname, ''),
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
		-- LEFT JOIN：sender 行被物理删除时（FK CASCADE）会话历史不应整段消失，
		-- 与群聊侧一致（nickname/avatar 由 COALESCE 兜底为空）。
		LEFT JOIN users sender ON sender.user_id = pm.sender_user_id
		LEFT JOIN private_messages reply ON reply.message_id = pm.reply_to_message_id
		LEFT JOIN users reply_sender ON reply_sender.user_id = reply.sender_user_id
		LEFT JOIN private_chat_read_state prs
		  ON prs.chat_key = pm.chat_key AND prs.user_id = $3
		WHERE pm.chat_key = $1
		  AND ($4::timestamptz IS NULL OR pm.sent_at < $4::timestamptz OR (pm.sent_at = $4::timestamptz AND pm.message_id < $5))
		  AND ($7::timestamptz IS NULL OR pm.sent_at > $7::timestamptz)
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

	// RETURNING：0 行命中（lookup 与 UPDATE 之间被并发撤回/编辑）可检测，
	// repo 据此返回冲突，防止「DB 未变但广播已编辑/已撤回事件」。
	queryUpdatePrivateMessage = `
		UPDATE private_messages
		SET content = $2, content_prev = content, edited_at = NOW()
		WHERE message_id = $1 AND is_recalled = FALSE
		RETURNING message_id`

	queryRecallPrivateMessage = `
		UPDATE private_messages
		SET is_recalled = TRUE, recalled_at = NOW()
		WHERE message_id = $1 AND is_recalled = FALSE
		RETURNING message_id`

	queryGetPrivateMessageMeta = `
		SELECT chat_key,
		       sender_user_id,
		       type,
		       COALESCE(is_recalled, FALSE),
		       TO_CHAR(sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
		FROM private_messages
		WHERE message_id = $1`
	queryGetPrivateMessageForReply = `
		SELECT pm.sender_user_id,
		       COALESCE(u.nickname, pm.sender_user_id),
		       COALESCE(pm.content, ''),
		       COALESCE(pm.share_code, '')
		FROM private_messages pm
		JOIN private_chats pc ON pc.chat_key = pm.chat_key
		LEFT JOIN users u ON u.user_id = pm.sender_user_id
		WHERE pm.message_id = $1
		  AND (pc.user_a_id = $2 OR pc.user_b_id = $2)`

	queryGetMediaFileForMessage = `
		SELECT file_id,
		       storage_key,
		       COALESCE(file_name, ''),
		       declared_file_size,
		       declared_mime_type
		FROM media_upload_sessions
		WHERE file_id = $1 AND status = 'completed' AND user_id = $2`

	queryPrivateMessageExists = `
		SELECT sender_user_id
		FROM private_messages
		WHERE message_id = $1`

	// 仅被 IsReactionTargetVisible 消费：已撤回消息按不存在处理（不再接受
	// 新的 reaction，与软删评论/软删瞬间同口径，审计 F-11）。
	queryPrivateMessageParticipants = `
SELECT pm.sender_user_id,
       CASE WHEN pc.user_a_id = pm.sender_user_id THEN pc.user_b_id ELSE pc.user_a_id END
FROM private_messages pm
JOIN private_chats pc ON pc.chat_key = pm.chat_key
WHERE pm.message_id = $1 AND pm.is_recalled = FALSE`

	// queryGetPrivateMessageSnapshot 取私聊消息真值（举报快照生成依据，spec
	// api_spec_types.md ReportSnapshot：服务端受理举报时生成并冻结）。
	// target_chat_id = 相对发送者的会话对方 user_id。
	queryGetPrivateMessageSnapshot = `
SELECT pm.sender_user_id,
       CASE WHEN pc.user_a_id = pm.sender_user_id THEN pc.user_b_id ELSE pc.user_a_id END,
       pm.type,
       COALESCE(pm.content, ''),
       COALESCE(pm.file_id, ''),
       TO_CHAR(pm.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
FROM private_messages pm
JOIN private_chats pc ON pc.chat_key = pm.chat_key
WHERE pm.message_id = $1`

	// 未读数独立 COUNT（口径与 History 的 is_read CASE 一致：对方发来的、
	// sent_at 晚于已读游标的消息）。
	queryUnreadPrivateMessageCount = `
SELECT COUNT(*)
FROM private_messages pm
LEFT JOIN private_chat_read_state prs
  ON prs.chat_key = pm.chat_key AND prs.user_id = $2
WHERE pm.chat_key = $1
  AND pm.sender_user_id <> $2
  AND pm.is_recalled = FALSE
  AND ($3::timestamptz IS NULL OR pm.sent_at > $3::timestamptz)
  AND (prs.last_read_sent_at IS NULL OR pm.sent_at > prs.last_read_sent_at
       OR (pm.sent_at = prs.last_read_sent_at AND prs.last_read_message_id IS NOT NULL AND pm.message_id <> prs.last_read_message_id))`

	queryCountPrivateMessagesByTarget = `
		SELECT CASE
		         WHEN pc.user_a_id = $1 THEN pc.user_b_id
		         ELSE pc.user_a_id
		       END AS target_user_id,
		       COUNT(*)
		FROM private_messages pm
		JOIN private_chats pc ON pc.chat_key = pm.chat_key
		WHERE pm.sender_user_id = $1 AND pm.sent_at >= $2
		GROUP BY target_user_id`

	// queryLockStrangerQuota 事务级咨询锁（事务结束自动释放，同 key 可重入）：
	// 以「发送者 + UTC 日期」为 key 串行化同发送者的配额检查与消息写入，
	// 消除 check-then-act 竞态（并发 Send 各自 COUNT 看不到对方未提交行会
	// 双双超额放行）；先到者提交后，后到者的事务内 COUNT 能看到其消息。
	queryLockStrangerQuota = `SELECT pg_advisory_xact_lock(hashtext($1)::bigint) IS NULL`
)
