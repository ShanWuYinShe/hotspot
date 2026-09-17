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

	queryUpsertUserSettings = `INSERT INTO user_settings (user_id, privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, reacted_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable)
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

	// 验码失败计数：给 login_id 最新有效码 failed_attempts + 1（048 迁移）。
	// 子查询在执行瞬间重判有效（未用、未过期），并发下至多多放行一两次，
	// 对 5 次烧码阈值无实质影响。
	queryBumpResetCodeFailure = `UPDATE auth_password_reset_requests
SET failed_attempts = failed_attempts + 1
WHERE request_id = (
  SELECT request_id FROM auth_password_reset_requests
  WHERE login_id = $1 AND used_at IS NULL AND expires_at > $2
  ORDER BY created_at DESC LIMIT 1)
RETURNING request_id, failed_attempts`

	queryBurnResetCode = `UPDATE auth_password_reset_requests
SET used_at = $2
WHERE request_id = $1`

	queryUpdateAccountPassword = `UPDATE auth_accounts
SET password_hash = $2, updated_at = $3
WHERE user_id = $1`

	queryUpdateLoginMeta = `UPDATE users
SET last_login_ip = $2,
    last_login_location = $3
WHERE user_id = $1`

	queryInsertRegisterCode = `INSERT INTO auth_register_verification_codes (request_id, login_id, verification_code, expires_at, created_at, consumed_at)
VALUES ($1, $2, $3, $4, $5, NULL)`

	queryLatestRegisterCodeCreatedAt = `SELECT created_at
FROM auth_register_verification_codes
WHERE login_id = $1
ORDER BY created_at DESC
LIMIT 1`

	queryValidRegisterCode = `SELECT request_id
FROM auth_register_verification_codes
WHERE login_id = $1 AND verification_code = $2 AND consumed_at IS NULL AND expires_at > $3
ORDER BY created_at DESC
LIMIT 1`

	queryConsumeRegisterCode = `UPDATE auth_register_verification_codes
SET consumed_at = $3
WHERE login_id = $1 AND verification_code = $2 AND consumed_at IS NULL`

	queryBumpRegisterCodeFailure = `UPDATE auth_register_verification_codes
SET failed_attempts = failed_attempts + 1
WHERE request_id = (
  SELECT request_id FROM auth_register_verification_codes
  WHERE login_id = $1 AND consumed_at IS NULL AND expires_at > $2
  ORDER BY created_at DESC LIMIT 1)
RETURNING request_id, failed_attempts`

	queryBurnRegisterCode = `UPDATE auth_register_verification_codes
SET consumed_at = $2
WHERE request_id = $1`

	// 数据保留期清理（retention.go）：auth_sessions 行含 IP/设备 PII，
	// 已撤销 30 天后删除；验证码/重置码行内含验证码明文，过期 7 天后删除。
	// 此前两类行永不删除（唯一出口是注销级联），表随注册量无限增长。
	querySweepRevokedSessions = `
WITH del AS (
    DELETE FROM auth_sessions
    WHERE revoked_at IS NOT NULL AND revoked_at < $1
    RETURNING 1
)
SELECT COUNT(*) FROM del`

	querySweepRegisterCodes = `
WITH del AS (
    DELETE FROM auth_register_verification_codes
    WHERE expires_at < $1
    RETURNING 1
)
SELECT COUNT(*) FROM del`

	querySweepPasswordResets = `
WITH del AS (
    DELETE FROM auth_password_reset_requests
    WHERE expires_at < $1
    RETURNING 1
)
SELECT COUNT(*) FROM del`
)
