package groupmessages

const (
	queryGetGroupName = `
	SELECT name
	FROM groups
	WHERE group_id = $1`

	queryInsertGroupMessage = `
	INSERT INTO group_messages (message_id, group_id, sender_user_id, type, content, share_code, file_id, reply_to_message_id, mentions, sent_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`

	queryGroupMessagesHistory = `
		SELECT gm.message_id,
		       gm.type,
		       COALESCE(gm.content, ''),
		       TO_CHAR(gm.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
		       COALESCE(TO_CHAR(gm.edited_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
		       COALESCE(TO_CHAR(gm.recalled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
		       COALESCE(gm.file_id, ''),
		       COALESCE(gm.share_code, ''),
		       COALESCE(reply.message_id, ''),
		       COALESCE(reply.type, ''),
		       COALESCE(reply.content, ''),
		       COALESCE(reply.share_code, ''),
		       COALESCE(reply.sender_user_id, ''),
		       COALESCE(reply_sender.nickname, ''),
		       COALESCE(reply_sender.avatar_url, ''),
		       COALESCE(TO_CHAR(reply.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''),
		       COALESCE(gm.sender_user_id, ''),
		       COALESCE(sender.nickname, ''),
		       COALESCE(sender.avatar_url, ''),
		       COALESCE(gm.recalled_by, ''),
		       COALESCE(recall_operator.nickname, ''),
		       COALESCE(recall_operator.avatar_url, ''),
		       gm.mentions
		FROM group_messages gm
		LEFT JOIN users sender ON sender.user_id = gm.sender_user_id
		LEFT JOIN users recall_operator ON recall_operator.user_id = gm.recalled_by
		LEFT JOIN group_messages reply ON reply.message_id = gm.reply_to_message_id
		LEFT JOIN users reply_sender ON reply_sender.user_id = reply.sender_user_id
		-- read_count 不在本查询逐行 LATERAL 计数：万人大群每行扫一遍
		-- read_state×members（审计 perf P2-13）。History 由 repo 在主查询后用
		-- queryGroupMessageReadCounts 按当前页 message_id ANY 一次聚合回填，
		-- 计数口径不变（last_read_sent_at >= sent_at 且 joined_at <= sent_at，
		-- 即只统计「消息发出时已在群」的成员，F-07 语义）。
		WHERE gm.group_id = $1
		  AND ($2::timestamptz IS NULL OR gm.sent_at < $2::timestamptz OR (gm.sent_at = $2::timestamptz AND gm.message_id < $3))
		ORDER BY gm.sent_at DESC, gm.message_id DESC
		LIMIT $4`

	// RETURNING：0 行命中（lookup 与 UPDATE 之间被并发撤回/编辑）可检测，
	// repo 据此返回冲突，防止「DB 未变但广播已编辑/已撤回事件」。
	queryUpdateGroupMessage = `
	UPDATE group_messages
	SET content = $3, content_prev = content, edited_at = NOW()
	WHERE message_id = $1 AND sender_user_id = $2 AND is_recalled = FALSE
	RETURNING message_id`

	queryRecallGroupMessage = `
	UPDATE group_messages
	SET is_recalled = TRUE, recalled_at = NOW(), recalled_by = $2
	WHERE message_id = $1 AND sender_user_id = $2 AND is_recalled = FALSE
	RETURNING message_id`

	// queryRecallGroupMessageAsAdmin recalls a message regardless of who sent
	// it, with no recall-window enforcement（groups/admin/messages/recall）。
	queryRecallGroupMessageAsAdminReturning = `
	UPDATE group_messages
	SET is_recalled = TRUE, recalled_at = NOW(), recalled_by = $2
	WHERE message_id = $1 AND is_recalled = FALSE
	RETURNING message_id`

	// queryLookupMessageOwner fetches only what the authorization layer needs:
	// the group a message belongs to and who sent it. NULL sender becomes ""
	// so the service can refuse to mutate legacy/anonymous messages.
	queryLookupMessageOwner = `
	SELECT group_id, sender_user_id FROM group_messages WHERE message_id = $1`

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
	   OR (group_message_read_state.last_read_sent_at, COALESCE(group_message_read_state.last_read_message_id, ''))
	      < (EXCLUDED.last_read_sent_at, COALESCE(EXCLUDED.last_read_message_id, ''))`

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
	   OR (group_message_read_state.last_read_sent_at, COALESCE(group_message_read_state.last_read_message_id, ''))
	      < (EXCLUDED.last_read_sent_at, COALESCE(EXCLUDED.last_read_message_id, ''))`

	queryGroupMessageReadCount = `
	SELECT COUNT(*)::int
	FROM group_message_read_state grs
	JOIN group_members member
	  ON member.group_id = grs.group_id AND member.user_id = grs.user_id
	WHERE grs.group_id = $1
	  AND grs.last_read_sent_at >= (SELECT gm.sent_at FROM group_messages gm WHERE gm.group_id = $1 AND gm.message_id = $2)
	  AND member.joined_at <= (SELECT gm.sent_at FROM group_messages gm WHERE gm.group_id = $1 AND gm.message_id = $2)`

	// queryGroupMessageReadCounts 批量版已读计数：一次往返取多条消息的已读
	// 成员数（flush 节流窗口尾使用），计数口径与 queryGroupMessageReadCount
	// 一致（last_read_sent_at >= 消息 sent_at 且 joined_at <= sent_at 的成员）。
	queryGroupMessageReadCounts = `
SELECT gm.message_id, COUNT(member.user_id)::int
FROM group_messages gm
LEFT JOIN group_message_read_state grs
  ON grs.group_id = gm.group_id AND grs.last_read_sent_at >= gm.sent_at
LEFT JOIN group_members member
  ON member.group_id = grs.group_id AND member.user_id = grs.user_id
 AND member.joined_at <= gm.sent_at
WHERE gm.group_id = $1 AND gm.message_id = ANY($2)
GROUP BY gm.message_id`

	queryLatestGroupMessageID = `
	SELECT gm.message_id
	FROM group_messages gm
	WHERE gm.group_id = $1
	ORDER BY gm.sent_at DESC, gm.message_id DESC
	LIMIT 1`

	queryUserGroupIDs = `
	SELECT DISTINCT group_id FROM group_members WHERE user_id = $1`

	queryCheckGroupMembership = `
SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2`

	queryFilterGroupMemberIDs = `
SELECT user_id FROM group_members WHERE group_id = $1 AND user_id = ANY($2)`

	queryMemberMuteStatus = `
SELECT is_muted, COALESCE(TO_CHAR(muted_until AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), '')
FROM group_members WHERE group_id = $1 AND user_id = $2`

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
	   OR (group_message_read_state.last_read_sent_at, COALESCE(group_message_read_state.last_read_message_id, ''))
	      < (EXCLUDED.last_read_sent_at, COALESCE(EXCLUDED.last_read_message_id, ''))`

	queryGetGroupMessageForReply = `
		SELECT gm.sender_user_id,
		       COALESCE(u.nickname, gm.sender_user_id),
		       COALESCE(u.avatar_url, ''),
		       gm.type,
		       COALESCE(gm.content, ''),
		       COALESCE(gm.share_code, ''),
		       TO_CHAR(gm.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
		FROM group_messages gm
		LEFT JOIN users u ON u.user_id = gm.sender_user_id
		WHERE gm.message_id = $1`

	// 仅被 IsReactionTargetVisible 消费：撤回消息按不存在处理（不再接受
	// 新的 reaction，与软删评论/软删瞬间同口径，审计 F-11）。
	queryGetGroupMessageGroupID = `
		SELECT group_id
		FROM group_messages
		WHERE message_id = $1 AND is_recalled = FALSE`

	// queryGetGroupMessageSnapshot 取群消息真值（举报快照生成依据，spec
	// api_spec_types.md ReportSnapshot：服务端受理举报时生成并冻结）。
	queryGetGroupMessageSnapshot = `
		SELECT gm.group_id,
		       gm.type,
		       COALESCE(gm.content, ''),
		       COALESCE(gm.file_id, ''),
		       TO_CHAR(gm.sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
		FROM group_messages gm
		WHERE gm.message_id = $1`

	// queryGetGroupMediaFileForMessage 取聊天媒体元数据（WS media 负载）。owner
	// 过滤与私聊侧 queryGetMediaFileForMessage 一致：只能引用自己上传的文件。
	queryGetGroupMediaFileForMessage = `
		SELECT file_id,
		       storage_key,
		       COALESCE(file_name, ''),
		       declared_file_size,
		       declared_mime_type,
		       COALESCE(thumbnail_url, ''),
		       width,
		       height,
		       duration
		FROM media_upload_sessions
		WHERE file_id = $1 AND status = 'completed' AND user_id = $2`

	// queryGroupMediaFilesByIDs 批量取消息媒体元数据（History media 回填）。
	// 历史行面向全体群成员可见，不做 owner 过滤；仅回填 completed 文件。
	queryGroupMediaFilesByIDs = `
		SELECT file_id,
		       storage_key,
		       COALESCE(file_name, ''),
		       declared_file_size,
		       declared_mime_type,
		       COALESCE(thumbnail_url, ''),
		       width,
		       height,
		       duration
		FROM media_upload_sessions
		WHERE file_id = ANY($1) AND status = 'completed'`
)
