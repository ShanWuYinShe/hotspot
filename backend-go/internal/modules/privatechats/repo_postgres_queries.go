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
)
