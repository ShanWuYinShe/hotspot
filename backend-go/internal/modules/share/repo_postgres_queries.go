package share

const (
	queryGetShare = `
SELECT share_id, share_code, share_type, creator_user_id, created_at, is_revoked
FROM shares
WHERE share_code = $1`

	queryListShares = `
SELECT share_id, share_code, share_type, created_at, is_revoked
FROM shares
WHERE creator_user_id = $1
ORDER BY created_at DESC, share_id DESC`
)
