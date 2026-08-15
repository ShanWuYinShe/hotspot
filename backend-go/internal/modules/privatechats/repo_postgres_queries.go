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
	WHERE pc.user_a_id = $1 OR pc.user_b_id = $1`

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
