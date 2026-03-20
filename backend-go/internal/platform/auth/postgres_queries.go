package auth

const (
	queryGetSession = `
SELECT session_id, user_id, created_at, last_active_at, revoked_at
FROM auth_sessions
WHERE session_id = $1`

	queryListActiveSessionsByUser = `
SELECT session_id, user_id, created_at, last_active_at, revoked_at
FROM auth_sessions
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY last_active_at DESC, session_id ASC`

	queryGetRefreshToken = `
SELECT user_id, session_id, refresh_token, rotated_to, grace_until, expires_at
FROM auth_refresh_tokens
WHERE refresh_token = $1`
)
