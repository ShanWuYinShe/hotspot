package comments

const (
	queryInsertComment = `
INSERT INTO comments (comment_id, moment_id, user_id, parent_id, content, created_at, likes_count, replies_count)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	queryDeleteComment = `
DELETE FROM comments
WHERE comment_id = $1`

	queryListComments = `
SELECT comment_id, user_id, COALESCE(parent_id, ''), content, created_at, likes_count, replies_count
FROM comments
WHERE moment_id = $1
ORDER BY created_at DESC, comment_id DESC`
)
