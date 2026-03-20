package users

const (
	queryGetUserProfile = `
SELECT user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, presence_status, last_online, created_at
FROM users
WHERE user_id = $1`

	queryGetUserSettings = `
SELECT privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, liked_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable
FROM user_settings
WHERE user_id = $1`
)
