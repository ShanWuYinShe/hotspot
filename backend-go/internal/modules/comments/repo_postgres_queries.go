package comments

const (
	queryInsertComment = `
	INSERT INTO comments (comment_id, moment_id, user_id, parent_id, content, created_at, likes_count, replies_count)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	queryGetNickname = `
	SELECT COALESCE(nickname, user_id) FROM users WHERE user_id = $1`

	querySoftDeleteComment = `
	UPDATE comments SET is_deleted = TRUE, content = ''
	WHERE comment_id = $1`

	// 软删 + 父评论 replies_count 递减 + 命中标志，单条 SQL 原子完成。
	// deleted CTE 带 is_deleted = FALSE 守卫：并发重复 Delete 时第二次 0 行
	// RETURNING → deleted=false、replies_count 不再递减（此前守卫缺失，service
	// 层「Get 检查→软删」的 TOCTOU 窗口内双发执行会双重递减，且该计数无自愈
	// 机制）。调用方以 deleted=false 为准跳过 moments.comments_count 递减。
	querySoftDeleteReplyAndDecrementParent = `
	WITH deleted AS (
		UPDATE comments SET is_deleted = TRUE, content = ''
		WHERE comment_id = $1 AND is_deleted = FALSE
		RETURNING parent_id
	), decremented AS (
		UPDATE comments SET replies_count = GREATEST(replies_count - 1, 0)
		WHERE comment_id IN (SELECT parent_id FROM deleted WHERE parent_id IS NOT NULL AND parent_id != '')
	)
	SELECT EXISTS(SELECT 1 FROM deleted) AS deleted`

	queryListComments = `
		SELECT c.comment_id, c.user_id, COALESCE(u.nickname, c.user_id), COALESCE(c.parent_id, ''), c.content, c.created_at, c.likes_count, c.replies_count, c.is_deleted
		FROM comments c
		LEFT JOIN users u ON u.user_id = c.user_id
		WHERE c.moment_id = $1
		  AND c.parent_id IS NULL
		  AND ($2::timestamptz IS NULL OR (c.created_at, c.comment_id) < ($2::timestamptz, $3))
		ORDER BY c.created_at DESC, c.comment_id DESC
		LIMIT $4`

	queryGetComment = `
		SELECT c.comment_id, c.moment_id, c.user_id, COALESCE(u.nickname, c.user_id), COALESCE(c.parent_id, ''), c.content, c.created_at, c.likes_count, c.replies_count, c.is_deleted
		FROM comments c
		LEFT JOIN users u ON u.user_id = c.user_id
		WHERE c.comment_id = $1`

	queryListCommentReplies = `
		SELECT c.comment_id, c.user_id, COALESCE(u.nickname, c.user_id), COALESCE(c.parent_id, ''), c.content, c.created_at, c.likes_count, c.replies_count, c.is_deleted
		FROM comments c
		LEFT JOIN users u ON u.user_id = c.user_id
		WHERE c.parent_id = $1
		  AND ($2::timestamptz IS NULL OR (c.created_at, c.comment_id) > ($2::timestamptz, $3))
		ORDER BY c.created_at ASC, c.comment_id ASC
		LIMIT $4`

	// queryListTopRepliesByParentIDs fetches up to $2 newest replies per parent
	// in a single query using row_number(), eliminating the N+1 of calling
	// ListReplies once per top-level comment.
	queryListTopRepliesByParentIDs = `
		SELECT comment_id, user_id, COALESCE(nickname, user_id), COALESCE(parent_id, ''), content, created_at, likes_count, replies_count, is_deleted
		FROM (
		  SELECT c.*, u.nickname, ROW_NUMBER() OVER (PARTITION BY c.parent_id ORDER BY c.created_at DESC, c.comment_id DESC) AS rn
		  FROM comments c
		  LEFT JOIN users u ON u.user_id = c.user_id
		  WHERE c.parent_id = ANY($1::text[])
		) ranked
		WHERE rn <= $2`

	queryIncrementCommentRepliesCount = `
	UPDATE comments SET replies_count = replies_count + 1 WHERE comment_id = $1`

	queryDecrementCommentRepliesCount = `
	UPDATE comments SET replies_count = GREATEST(replies_count - 1, 0) WHERE comment_id = $1`
)
