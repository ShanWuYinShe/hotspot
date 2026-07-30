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

	queryRotateRefreshToken = `
WITH rotated AS (
    UPDATE auth_refresh_tokens
    SET rotated_to = $2, grace_until = $3
    WHERE refresh_token = $1
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
    expires_at = EXCLUDED.expires_at`

	queryGetRefreshToken = `
SELECT user_id, session_id, refresh_token, rotated_to, grace_until, expires_at
FROM auth_refresh_tokens
WHERE refresh_token = $1`
)
