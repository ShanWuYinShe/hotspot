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

	// queryListPrivateChats 会话列表 keyset 分页（审查 P2-7：此前每次请求全量
	// 拉取用户全部会话 + 全量批量聚合，翻页成本恒为 O(全量)）。
	//
	// 排序键与契约一致：is_pinned DESC, last_activity_at DESC, target_user_id ASC。
	// last_activity 取「最近可见消息 sent_at」（LATERAL 走
	// idx_private_messages_chat_key_sent_at 倒序取 1 行），无消息时回退 chat
	// created_at——与 buildChats 用 LastByChatKeys(boundaries=visible_since) 的口径
	// 一致。已删除且窗口内无消息的会话在 SQL 里直接排除（等价 buildChats 的跳过
	// 逻辑），并 JOIN users 过滤已注销目标，避免分页出现空页。
	queryListPrivateChats = `
	SELECT pc.chat_key,
	       CASE WHEN pc.user_a_id = $1 THEN pc.user_b_id ELSE pc.user_a_id END AS target_user_id,
	       pc.created_at,
	       COALESCE(pcs.is_muted, FALSE),
	       COALESCE(pcs.is_pinned, FALSE),
	       pcs.visible_since,
	       pcs.deleted_at,
	       COALESCE(lm.last_sent_at, pc.created_at) AS last_activity_at
	FROM private_chats pc
	LEFT JOIN private_chat_user_state pcs
	  ON pcs.chat_key = pc.chat_key AND pcs.user_id = $1
	LEFT JOIN LATERAL (
	    SELECT m.sent_at AS last_sent_at
	    FROM private_messages m
	    WHERE m.chat_key = pc.chat_key
	      AND (pcs.visible_since IS NULL OR m.sent_at >= pcs.visible_since)
	    ORDER BY m.sent_at DESC
	    LIMIT 1
	) lm ON TRUE
	JOIN users u
	  ON u.user_id = (CASE WHEN pc.user_a_id = $1 THEN pc.user_b_id ELSE pc.user_a_id END)
	 AND u.is_deleted = FALSE
	WHERE (pc.user_a_id = $1 OR pc.user_b_id = $1)
	  AND NOT (pcs.deleted_at IS NOT NULL AND lm.last_sent_at IS NULL)
	  AND ($2::boolean IS NULL
	       OR COALESCE(pcs.is_pinned, FALSE) < $2
	       OR (COALESCE(pcs.is_pinned, FALSE) = $2
	           AND (COALESCE(lm.last_sent_at, pc.created_at) < $3::timestamptz
	                OR (COALESCE(lm.last_sent_at, pc.created_at) = $3::timestamptz
	                    AND (CASE WHEN pc.user_a_id = $1 THEN pc.user_b_id ELSE pc.user_a_id END) > $4))))
	ORDER BY COALESCE(pcs.is_pinned, FALSE) DESC,
	         last_activity_at DESC,
	         target_user_id ASC
	LIMIT $5`

	queryGetPrivateChatState = `
	SELECT pc.chat_key,
	       CASE WHEN pc.user_a_id = $1 THEN pc.user_b_id ELSE pc.user_a_id END AS target_user_id,
	       pc.created_at,
	       COALESCE(pcs.is_muted, FALSE),
	       COALESCE(pcs.is_pinned, FALSE),
	       pcs.visible_since,
	       pcs.deleted_at
	FROM private_chats pc
	LEFT JOIN private_chat_user_state pcs
	  ON pcs.chat_key = pc.chat_key AND pcs.user_id = $1
	WHERE pc.chat_key = $2`

	// 条件更新（指针语义）：仅覆盖调用方显式提供的列。此前 read-modify-write
	// 整行覆盖，并发 mute + pin 后到者会用旧快照抹掉前者的修改。
	queryUpdatePrivateChatUserSettings = `
INSERT INTO private_chat_user_state (chat_key, user_id, is_muted, is_pinned, visible_since, deleted_at, updated_at)
VALUES ($1, $2, COALESCE($3, FALSE), COALESCE($4, FALSE), NOW(), NULL, NOW())
ON CONFLICT (chat_key, user_id) DO UPDATE SET
is_muted = COALESCE($3, private_chat_user_state.is_muted),
is_pinned = COALESCE($4, private_chat_user_state.is_pinned),
updated_at = NOW()`

	queryMarkPrivateChatDeleted = `
INSERT INTO private_chat_user_state (chat_key, user_id, is_muted, is_pinned, visible_since, deleted_at, updated_at)
SELECT $1, $2, FALSE, FALSE, NOW(), $3, NOW()
WHERE EXISTS (SELECT 1 FROM private_chats WHERE chat_key = $1)
ON CONFLICT (chat_key, user_id) DO UPDATE SET
visible_since = EXCLUDED.visible_since,
deleted_at = EXCLUDED.deleted_at,
updated_at = NOW()`

	queryUpsertPrivateChatUserState = `
	INSERT INTO private_chat_user_state (chat_key, user_id, is_muted, is_pinned, visible_since, deleted_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, NOW())
	ON CONFLICT (chat_key, user_id) DO UPDATE SET
	is_muted = EXCLUDED.is_muted,
	is_pinned = EXCLUDED.is_pinned,
	visible_since = EXCLUDED.visible_since,
	deleted_at = EXCLUDED.deleted_at,
	updated_at = NOW()`

	// queryCountUnreadPrivateMessagesForUser 当前用户全部私聊会话的未读消息
	// 总数（冷启动 unread_summary 聚合，一条 COUNT 汇总）。逐会话口径与
	// privatemessages 的 queryUnreadPrivateMessageCount /
	// queryUnreadPrivateMessageCountsByChatKeys 逐字一致（已读游标含同一
	// sent_at 下 message_id 的边界处理、排除本人发送与已撤回消息），外层再叠加
	// 本模块的会话可见边界与列表可达性：
	//   - visible_since/deleted_at：删除会话后只统计可见窗口内的新消息（口径
	//     同 buildChats 的 visibleBoundary，SQL 用 COALESCE 兜底旧行）；
	//   - JOIN users peer ... is_deleted = FALSE：对方注销后会话在列表中不可达
	//     （queryListPrivateChats 同样过滤），此处计入会形成无法清除的角标。
	// 驱动表为 private_chats（idx_private_chats_user_a/b），消息侧走
	// idx_private_messages_chat_key_sent_at。
	queryCountUnreadPrivateMessagesForUser = `
	SELECT COUNT(*)
	FROM private_chats pc
	LEFT JOIN private_chat_user_state pcs
	  ON pcs.chat_key = pc.chat_key AND pcs.user_id = $1
	JOIN users peer
	  ON peer.user_id = (CASE WHEN pc.user_a_id = $1 THEN pc.user_b_id ELSE pc.user_a_id END)
	 AND peer.is_deleted = FALSE
	JOIN private_messages pm
	  ON pm.chat_key = pc.chat_key
	 AND pm.sender_user_id <> $1
	 AND pm.is_recalled = FALSE
	 AND (COALESCE(pcs.visible_since, pcs.deleted_at) IS NULL
	      OR pm.sent_at > COALESCE(pcs.visible_since, pcs.deleted_at))
	LEFT JOIN private_chat_read_state prs
	  ON prs.chat_key = pc.chat_key AND prs.user_id = $1
	WHERE (pc.user_a_id = $1 OR pc.user_b_id = $1)
	  AND (prs.last_read_sent_at IS NULL
	       OR pm.sent_at > prs.last_read_sent_at
	       OR (pm.sent_at = prs.last_read_sent_at
	           AND prs.last_read_message_id IS NOT NULL
	           AND pm.message_id <> prs.last_read_message_id))`
)
