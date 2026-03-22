package share

const (
	queryInsertShare = `
INSERT INTO shares (share_id, share_code, creator_user_id, share_type, payload, created_at, expires_at, is_revoked)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	queryGetShare = `
SELECT share_id, share_code, share_type, creator_user_id, payload::text, created_at, COALESCE(to_char(expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''), is_revoked
FROM shares
WHERE share_code = $1`

	queryListShares = `
SELECT share_id, share_code, share_type, created_at, COALESCE(to_char(expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''), is_revoked
FROM shares
WHERE creator_user_id = $1
ORDER BY created_at DESC, share_id DESC`

	queryRevokeShare = `
UPDATE shares
SET is_revoked = TRUE
WHERE share_id = $1 AND creator_user_id = $2 AND is_revoked = FALSE`
)
