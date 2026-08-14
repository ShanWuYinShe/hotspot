package notify

const (
	queryListNotifys = `
SELECT notify_id, type, is_read, created_at, payload::text
FROM notifys
WHERE user_id = $1
  AND ($2 = '' OR type = $2)
  AND ($3 IS NULL OR is_read = $3)
  AND ($4 IS NULL OR created_at < $4 OR (created_at = $4 AND notify_id < $5))
ORDER BY created_at DESC, notify_id DESC
LIMIT $6`

	queryInsertNotify = `
INSERT INTO notifys (notify_id, user_id, type, is_read, created_at, payload)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)`

	// queryInsertNotifysBatch fans out one notify to many users in
	// a single round trip. Each row gets its own notify_id (the table's
	// PK is notify_id alone, so a shared id would collide on the 2nd
	// user); the prefix is concatenated with the row ordinal to keep ids unique
	// while remaining traceable to the same logical event.
	queryInsertNotifysBatch = `
WITH target AS (
  SELECT uid, row_number() OVER () AS rn
  FROM unnest($1::text[]) AS uid
)
INSERT INTO notifys (notify_id, user_id, type, is_read, created_at, payload)
SELECT $2 || '_' || rn, uid, $3, $4, $5, $6::jsonb
FROM target`

	queryUnreadNotifyCounts = `
SELECT type, COUNT(*)::text
FROM notifys
WHERE user_id = $1 AND is_read = FALSE
GROUP BY type`

	queryMarkNotifysRead = `
UPDATE notifys
SET is_read = TRUE
WHERE user_id = $1 AND notify_id = ANY($2)`

	queryMarkAllNotifysRead = `
UPDATE notifys
SET is_read = TRUE
WHERE user_id = $1 AND is_read = FALSE`
)
