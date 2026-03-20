package users

const (
	queryGetUserProfile = `
SELECT user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, presence_status, last_online, created_at
FROM users
WHERE user_id = $1`

	queryUpsertUserProfile = `
INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, presence_status, last_online, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (user_id) DO UPDATE SET
nickname = EXCLUDED.nickname,
avatar_url = EXCLUDED.avatar_url,
user_type = EXCLUDED.user_type,
bio = EXCLUDED.bio,
gender = EXCLUDED.gender,
email = EXCLUDED.email,
phone = EXCLUDED.phone,
location_name = EXCLUDED.location_name,
latitude = EXCLUDED.latitude,
longitude = EXCLUDED.longitude,
presence_status = EXCLUDED.presence_status,
last_online = EXCLUDED.last_online`

	queryDeleteUserProfile = `DELETE FROM users WHERE user_id = $1`

	queryGetUserSettings = `
SELECT privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, liked_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable
FROM user_settings
WHERE user_id = $1`

	queryUpsertUserSettings = `
INSERT INTO user_settings (user_id, privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, liked_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (user_id) DO UPDATE SET
privacy_level = EXCLUDED.privacy_level,
location_visible = EXCLUDED.location_visible,
presence_visible = EXCLUDED.presence_visible,
social_lists_visible = EXCLUDED.social_lists_visible,
groups_visible = EXCLUDED.groups_visible,
liked_moments_visible = EXCLUDED.liked_moments_visible,
stats_visible = EXCLUDED.stats_visible,
allow_follow = EXCLUDED.allow_follow,
allow_private_chat = EXCLUDED.allow_private_chat,
searchable = EXCLUDED.searchable,
phone_searchable = EXCLUDED.phone_searchable,
nickname_searchable = EXCLUDED.nickname_searchable,
location_searchable = EXCLUDED.location_searchable`

	queryUpsertUserRelation = `
INSERT INTO user_relations (user_id, target_user_id, is_following, is_blocked, remark)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
is_following = EXCLUDED.is_following,
is_blocked = EXCLUDED.is_blocked,
remark = EXCLUDED.remark`

	queryGetUserRelation = `
SELECT is_following, is_blocked, COALESCE(remark, '')
FROM user_relations
WHERE user_id = $1 AND target_user_id = $2`

	queryListBlockedUsers = `
SELECT target_user_id, COALESCE(remark, '')
FROM user_relations
WHERE user_id = $1 AND is_blocked = TRUE
ORDER BY target_user_id ASC`
)
