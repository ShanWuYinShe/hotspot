package auth

const (
	queryUpsertSession = `
INSERT INTO auth_sessions (session_id, user_id, created_at, last_active_at, revoked_at, device_type, device_name, ip_address, ip_location, device_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (session_id) DO UPDATE
SET user_id = EXCLUDED.user_id,
    created_at = EXCLUDED.created_at,
    last_active_at = EXCLUDED.last_active_at,
    revoked_at = EXCLUDED.revoked_at,
    device_type = EXCLUDED.device_type,
    device_name = EXCLUDED.device_name,
    ip_address = EXCLUDED.ip_address,
    ip_location = EXCLUDED.ip_location,
    device_id = EXCLUDED.device_id`

	queryTouchSession = `
UPDATE auth_sessions
SET last_active_at = NOW()
WHERE session_id = $1 AND revoked_at IS NULL`

	queryRevokeSession = `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE session_id = $1 AND revoked_at IS NULL`

	queryRevokeOtherSessions = `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND session_id <> $2 AND revoked_at IS NULL`

	queryGetSession = `
SELECT session_id, user_id, created_at, last_active_at, revoked_at, COALESCE(device_type, ''), COALESCE(device_name, ''), COALESCE(ip_address, ''), COALESCE(ip_location, ''), COALESCE(device_id, '')
FROM auth_sessions
WHERE session_id = $1`

	queryListActiveSessionsByUser = `
SELECT session_id, user_id, created_at, last_active_at, revoked_at, COALESCE(device_type, ''), COALESCE(device_name, ''), COALESCE(ip_address, ''), COALESCE(ip_location, ''), COALESCE(device_id, '')
FROM auth_sessions
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY last_active_at DESC, session_id ASC`

	queryUpsertRefreshToken = `
INSERT INTO auth_refresh_tokens (refresh_token, user_id, session_id, rotated_to, grace_until, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (refresh_token) DO UPDATE
SET user_id = EXCLUDED.user_id,
    session_id = EXCLUDED.session_id,
    rotated_to = EXCLUDED.rotated_to,
    grace_until = EXCLUDED.grace_until,
    expires_at = EXCLUDED.expires_at`

	// 轮换原子性：UPDATE 带 rotated_to IS NULL 守卫——并发携带同一旧 token 的
	// 两个请求只有一个能命中 rotated CTE（另一个 0 行 → INSERT 不执行 →
	// RETURNING 无行），repo 据此返回轮换冲突，防止分叉出两个互不可见的
	// 有效 token（绕过被盗会话的复用检测）。
	// $7 = grace_token_enc：新链 token 的 AES-GCM 托管密文，写在被轮换的
	// 旧行上——并发刷新败者在宽限期内凭此换回明文，行随 sweeper 清除。
	queryRotateRefreshToken = `
WITH rotated AS (
    UPDATE auth_refresh_tokens
    SET rotated_to = $2, grace_until = $3, grace_token_enc = $7
    WHERE refresh_token = $1 AND rotated_to IS NULL
    RETURNING 1
)
INSERT INTO auth_refresh_tokens (refresh_token, user_id, session_id, rotated_to, grace_until, expires_at)
SELECT $2, $4, $5, NULL, NULL, $6
WHERE EXISTS (SELECT 1 FROM rotated)
ON CONFLICT (refresh_token) DO UPDATE
SET user_id = EXCLUDED.user_id,
    session_id = EXCLUDED.session_id,
    rotated_to = EXCLUDED.rotated_to,
    grace_until = EXCLUDED.grace_until,
    expires_at = EXCLUDED.expires_at
RETURNING refresh_token`

	queryGetRefreshToken = `
SELECT user_id, session_id, refresh_token, rotated_to, grace_until, expires_at, grace_token_enc
FROM auth_refresh_tokens
WHERE refresh_token = $1`

	// 清理永不可用的 token：已过期，或轮换淘汰且 30s 宽限期早已结束的行。
	querySweepRefreshTokens = `
DELETE FROM auth_refresh_tokens
WHERE expires_at < $1
   OR (rotated_to IS NOT NULL AND (grace_until IS NULL OR grace_until < $1))`
)
