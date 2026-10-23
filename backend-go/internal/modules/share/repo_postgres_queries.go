package share

const (
	queryInsertShare = `
	INSERT INTO shares (share_id, share_code, creator_user_id, share_type, payload, created_at, expires_at, is_revoked)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	queryGetShare = `
	SELECT s.share_id,
	       s.share_code,
	       s.share_type,
	       s.creator_user_id,
	       u.nickname,
	       COALESCE(u.avatar_url, ''),
	       s.payload::text,
	       to_char(s.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	       COALESCE(to_char(s.expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
	       s.is_revoked
	FROM shares s
	JOIN users u ON u.user_id = s.creator_user_id
	WHERE s.share_code = $1`

	queryGetShareByID = `
	SELECT share_id, creator_user_id, is_revoked
	FROM shares
	WHERE share_id = $1`

	// queryResolveShareIDByCode 只取 share_id，不过滤 is_revoked / expires_at：
	// 撤销幂等需要定位到「已不可用」的分享（见 Repository.ResolveShareIDByCode）。
	queryResolveShareIDByCode = `SELECT share_id FROM shares WHERE share_code = $1`

	queryListShares = `
	SELECT share_id,
	       share_code,
	       share_type,
	       to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	       COALESCE(to_char(expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
	       is_revoked,
	       payload::text
	FROM shares
	WHERE creator_user_id = $1
	  AND ($2 IS NULL OR share_type = $2)
	  AND ($3 IS NULL OR (created_at, share_id) < ($3, $4))
	ORDER BY created_at DESC, share_id DESC
	LIMIT $5`

	queryRevokeShare = `
	UPDATE shares
	SET is_revoked = TRUE
	WHERE share_id = $1 AND creator_user_id = $2 AND is_revoked = FALSE`

	// 目标可见性（spec api_spec_share.md「校验目标资源存在且可访问」）：
	// 已注销（is_deleted）用户按不可访问处理，创建分享返回 10300 防枚举。
	queryShareUserExists = `
	SELECT user_id, nickname, COALESCE(avatar_url, '')
	FROM users
	WHERE user_id = $1
	  AND is_deleted = FALSE`

	queryShareGroupExists = `
	SELECT group_id, name, COALESCE(description, ''), COALESCE(avatar_url, '')
	FROM groups
	WHERE group_id = $1
	  AND deleted_at IS NULL`

	queryShareMomentExists = `
	SELECT m.moment_id,
	       m.description,
	       m.location_name,
	       u.nickname,
	       COALESCE(u.avatar_url, '')
	FROM moments m
	JOIN users u ON u.user_id = m.user_id
	WHERE m.moment_id = $1
	  AND m.privacy_level = 'public'
	  AND m.is_deleted = FALSE`
)
