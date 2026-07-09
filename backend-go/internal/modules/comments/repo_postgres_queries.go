package comments

const (
	queryInsertComment = `
	INSERT INTO comments (comment_id, moment_id, user_id, parent_id, content, created_at, likes_count, replies_count)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	querySoftDeleteComment = `
	UPDATE comments SET is_deleted = TRUE, content = ''
	WHERE comment_id = $1`

	queryListComments = `
	SELECT comment_id, user_id, COALESCE(parent_id, ''), content, created_at, likes_count, replies_count, is_deleted
	FROM comments
	WHERE moment_id = $1
	  AND parent_id IS NULL
	  AND ($2::timestamptz IS NULL OR (created_at, comment_id) < ($2::timestamptz, $3))
	ORDER BY created_at DESC, comment_id DESC
	LIMIT $4`

	queryGetComment = `
	SELECT comment_id, moment_id, user_id, COALESCE(parent_id, ''), content, created_at, likes_count, replies_count, is_deleted
	FROM comments
	WHERE comment_id = $1`

	queryListCommentReplies = `
	SELECT comment_id, user_id, COALESCE(parent_id, ''), content, created_at, likes_count, replies_count, is_deleted
	FROM comments
	WHERE parent_id = $1
	  AND ($2::timestamptz IS NULL OR (created_at, comment_id) < ($2::timestamptz, $3))
	ORDER BY created_at DESC, comment_id DESC
	LIMIT $4`

	// queryListTopRepliesByParentIDs fetches up to $2 newest replies per parent
	// in a single query using row_number(), eliminating the N+1 of calling
	// ListReplies once per top-level comment.
	queryListTopRepliesByParentIDs = `
	SELECT comment_id, user_id, COALESCE(parent_id, ''), content, created_at, likes_count, replies_count, is_deleted
	FROM (
	  SELECT c.*, ROW_NUMBER() OVER (PARTITION BY parent_id ORDER BY created_at DESC, comment_id DESC) AS rn
	  FROM comments c
	  WHERE parent_id = ANY($1::text[])
	) ranked
	WHERE rn <= $2`

	queryInsertCommentLike = `
	INSERT INTO comment_likes (comment_id, user_id, liked_at)
	VALUES ($1, $2, $3)
	ON CONFLICT (comment_id, user_id) DO NOTHING`

	queryDeleteCommentLike = `
	DELETE FROM comment_likes
	WHERE comment_id = $1 AND user_id = $2`

	queryCheckCommentLike = `
	SELECT 1 FROM comment_likes WHERE comment_id = $1 AND user_id = $2`

	queryIncrementCommentLikeCount = `
UPDATE comments SET likes_count = likes_count + 1 WHERE comment_id = $1`

	queryDecrementCommentLikeCount = `
UPDATE comments SET likes_count = GREATEST(likes_count - 1, 0) WHERE comment_id = $1`

	queryInsertCommentLikeAndCount = `
WITH ins AS (
  INSERT INTO comment_likes (comment_id, user_id, liked_at)
  VALUES ($1, $2, $3)
  ON CONFLICT (comment_id, user_id) DO NOTHING
  RETURNING comment_id
)
UPDATE comments
SET likes_count = likes_count + 1
WHERE comment_id = $1
  AND EXISTS (SELECT 1 FROM ins)
RETURNING EXISTS (SELECT 1 FROM ins) AS inserted`

	queryDeleteCommentLikeAndCount = `
WITH del AS (
  DELETE FROM comment_likes
  WHERE comment_id = $1 AND user_id = $2
  RETURNING comment_id
)
UPDATE comments
SET likes_count = GREATEST(likes_count - 1, 0)
WHERE comment_id = $1
  AND EXISTS (SELECT 1 FROM del)
RETURNING EXISTS (SELECT 1 FROM del) AS deleted`

	queryListCommentLikes = `
	SELECT cl.user_id, u.nickname, cl.liked_at
	FROM comment_likes cl
	LEFT JOIN users u ON u.user_id = cl.user_id
	WHERE cl.comment_id = $1
	  AND ($2::timestamptz IS NULL OR (cl.liked_at, cl.user_id) < ($2::timestamptz, $3))
	ORDER BY cl.liked_at DESC, cl.user_id DESC
	LIMIT $4`
)
