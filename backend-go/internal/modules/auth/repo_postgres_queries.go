package auth

const (
	queryInsertAuthAccount = `INSERT INTO auth_accounts (login_id, user_id, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)`

	queryDeleteAuthAccountByLoginID = `DELETE FROM auth_accounts
WHERE login_id = $1`

	queryInsertUserProfile = `INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, s2_cell_id, presence_status, last_online, created_at, register_ip, register_location, last_login_ip, last_login_location)
VALUES ($1, $2, $3, $4, '', '', $5, '', $6, $7, $8, $9, 'online', $10, $11, $12, $13, $14, $15)`

	queryDeleteUserProfileByUserID = `DELETE FROM users
WHERE user_id = $1`

	queryUpsertUserSettings = `INSERT INTO user_settings (user_id, privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, liked_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable)
VALUES ($1, 'public', TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, 'everyone', TRUE, FALSE, TRUE, TRUE)
ON CONFLICT (user_id) DO NOTHING`

	queryGetAccountByLoginID = `SELECT login_id, user_id, password_hash
FROM auth_accounts
WHERE login_id = $1`

	queryGetAccountByUserID = `SELECT login_id, user_id, password_hash
FROM auth_accounts
WHERE user_id = $1`

	queryInsertPasswordReset = `INSERT INTO auth_password_reset_requests (request_id, login_id, verification_code, expires_at, created_at, used_at)
VALUES ($1, $2, $3, $4, $5, NULL)`

	queryGetPasswordReset = `SELECT request_id, login_id, verification_code, expires_at, created_at, used_at
FROM auth_password_reset_requests
WHERE login_id = $1 AND verification_code = $2
ORDER BY created_at DESC
LIMIT 1`

	queryLatestPasswordResetByLoginID = `SELECT request_id, login_id, verification_code, expires_at, created_at, used_at
FROM auth_password_reset_requests
WHERE login_id = $1
ORDER BY created_at DESC
LIMIT 1`

	queryUsePasswordReset = `UPDATE auth_password_reset_requests
SET used_at = $2
WHERE request_id = $1`

	queryUpdateAccountPassword = `UPDATE auth_accounts
SET password_hash = $2, updated_at = $3
WHERE user_id = $1`

	queryUpdateLoginMeta = `UPDATE users
SET last_login_ip = $2,
    last_login_location = $3
WHERE user_id = $1`
)
