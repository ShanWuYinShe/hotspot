package comments

const (
	queryInsertComment = `
	INSERT INTO comments (comment_id, moment_id, user_id, parent_id, content, created_at, mentions)
	VALUES ($1,$2,$3,$4,$5,$6,$7)`

	queryGetNickname = `
	SELECT COALESCE(nickname, user_id) FROM users WHERE user_id = $1`

	// 软删 + 命中标志，单条 SQL 原子完成。deleted CTE 带 is_deleted = FALSE
	// 守卫：并发重复 Delete 第二次 0 行 → deleted=false，调用方据此跳过瞬间
	// 互动计数递减（幂等 no-op）。
	querySoftDeleteComment = `
	WITH deleted AS (
		UPDATE comments SET is_deleted = TRUE, content = ''
		WHERE comment_id = $1 AND is_deleted = FALSE
		RETURNING comment_id
	)
	SELECT EXISTS(SELECT 1 FROM deleted) AS deleted`

	queryListComments = `
		SELECT c.comment_id, c.user_id, COALESCE(u.nickname, c.user_id), COALESCE(c.parent_id, ''), c.content, c.created_at, (SELECT COUNT(*)::int FROM comments rc WHERE rc.parent_id = c.comment_id AND rc.is_deleted = FALSE) AS replies_count, c.is_deleted, c.mentions
		FROM comments c
		LEFT JOIN users u ON u.user_id = c.user_id
		WHERE c.moment_id = $1
		  AND c.parent_id IS NULL
		  AND ($2::timestamptz IS NULL OR (c.created_at, c.comment_id) < ($2::timestamptz, $3))
		ORDER BY c.created_at DESC, c.comment_id DESC
		LIMIT $4`

	queryGetComment = `
		SELECT c.comment_id, c.moment_id, c.user_id, COALESCE(u.nickname, c.user_id), COALESCE(c.parent_id, ''), c.content, c.created_at, (SELECT COUNT(*)::int FROM comments rc WHERE rc.parent_id = c.comment_id AND rc.is_deleted = FALSE) AS replies_count, c.is_deleted, c.mentions
		FROM comments c
		LEFT JOIN users u ON u.user_id = c.user_id
		WHERE c.comment_id = $1`

	queryListCommentReplies = `
		SELECT c.comment_id, c.user_id, COALESCE(u.nickname, c.user_id), COALESCE(c.parent_id, ''), c.content, c.created_at, (SELECT COUNT(*)::int FROM comments rc WHERE rc.parent_id = c.comment_id AND rc.is_deleted = FALSE) AS replies_count, c.is_deleted, c.mentions
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
		SELECT comment_id, user_id, COALESCE(nickname, user_id), COALESCE(parent_id, ''), content, created_at,
		       (SELECT COUNT(*)::int FROM comments rc WHERE rc.parent_id = ranked.comment_id AND rc.is_deleted = FALSE) AS replies_count,
		       is_deleted, mentions
		FROM (
		  SELECT c.comment_id, c.user_id, c.parent_id, c.content, c.created_at, c.is_deleted, c.mentions, u.nickname,
		         ROW_NUMBER() OVER (PARTITION BY c.parent_id ORDER BY c.created_at DESC, c.comment_id DESC) AS rn
		  FROM comments c
		  LEFT JOIN users u ON u.user_id = c.user_id
		  WHERE c.parent_id = ANY($1::text[])
		) ranked
		WHERE rn <= $2`
)
