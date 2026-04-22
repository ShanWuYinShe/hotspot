package users

const (
	queryGetUserProfile = `
SELECT user_id, nickname, COALESCE(avatar_url, ''), user_type, COALESCE(bio, ''), COALESCE(gender, ''), COALESCE(email, ''), COALESCE(phone, ''), location_name, latitude, longitude, presence_status, last_online, created_at
FROM users
WHERE user_id = $1`

	querySearchUserProfiles = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), u.user_type, COALESCE(u.bio, ''), COALESCE(u.gender, ''), COALESCE(u.email, ''), COALESCE(u.phone, ''), u.location_name, u.latitude, u.longitude, u.presence_status, u.last_online, u.created_at
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND ($1 = '' OR (us.nickname_searchable = TRUE AND u.nickname ILIKE '%' || $1 || '%') OR u.bio ILIKE '%' || $1 || '%')
ORDER BY u.created_at DESC, u.user_id ASC`

	queryListUserProfileTags = `
SELECT tag_name
FROM user_profile_tags
WHERE user_id = $1
ORDER BY tag_name ASC`

	queryReplaceUserProfileTags = `
DELETE FROM user_profile_tags WHERE user_id = $1`

	queryInsertUserProfileTag = `
INSERT INTO user_profile_tags (user_id, tag_name)
VALUES ($1, $2)
ON CONFLICT (user_id, tag_name) DO NOTHING`

	queryUpsertUserProfile = `
INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, presence_status, last_online, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::timestamptz,$14::timestamptz)
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
INSERT INTO user_relations (user_id, target_user_id, is_following, is_blocked, remark, followed_at, blocked_at, block_reason)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
is_following = EXCLUDED.is_following,
is_blocked = EXCLUDED.is_blocked,
remark = EXCLUDED.remark,
followed_at = EXCLUDED.followed_at,
blocked_at = EXCLUDED.blocked_at,
block_reason = EXCLUDED.block_reason`

	queryGetUserRelation = `
SELECT is_following, is_blocked, COALESCE(remark, ''), followed_at, blocked_at, COALESCE(block_reason, '')
FROM user_relations
WHERE user_id = $1 AND target_user_id = $2`

	queryCountUserStats = `
SELECT
  (SELECT COUNT(*) FROM moments m WHERE m.user_id = $1),
  (SELECT COUNT(*) FROM user_relations ur JOIN user_relations rev ON rev.user_id = ur.target_user_id AND rev.target_user_id = ur.user_id AND rev.is_following = TRUE WHERE ur.user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM user_relations ur WHERE ur.target_user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM user_relations ur WHERE ur.user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM group_members gm JOIN groups g ON g.group_id = gm.group_id WHERE gm.user_id = $1 AND g.deleted_at IS NULL)`

	queryListBlockedUsers = `
SELECT target_user_id, blocked_at, COALESCE(block_reason, '')
FROM user_relations
WHERE user_id = $1 AND is_blocked = TRUE
  AND (
    $2 IS NULL
    OR blocked_at < $2
    OR (blocked_at = $2 AND target_user_id > $3)
  )
ORDER BY blocked_at DESC, target_user_id ASC
LIMIT $4`

	queryListFriends = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), u.presence_status, u.last_online, COALESCE(u.last_online, NOW()), COALESCE(me.remark, ''), me.followed_at, GREATEST(me.followed_at, rev.followed_at)
FROM user_relations me
JOIN user_relations rev ON rev.user_id = me.target_user_id AND rev.target_user_id = me.user_id AND rev.is_following = TRUE
JOIN users u ON u.user_id = me.target_user_id
WHERE me.user_id = $1 AND me.is_following = TRUE
  AND (
    $2 IS NULL
    OR GREATEST(me.followed_at, rev.followed_at) < $2
    OR (GREATEST(me.followed_at, rev.followed_at) = $2 AND u.user_id > $3)
  )
ORDER BY GREATEST(me.followed_at, rev.followed_at) DESC, u.user_id ASC
LIMIT $4`

	queryListFollowing = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), me.followed_at, EXISTS(SELECT 1 FROM user_relations rev WHERE rev.user_id = me.target_user_id AND rev.target_user_id = me.user_id AND rev.is_following = TRUE), u.presence_status, u.last_online, COALESCE(u.last_online, NOW())
FROM user_relations me
JOIN users u ON u.user_id = me.target_user_id
WHERE me.user_id = $1 AND me.is_following = TRUE
  AND (
    $2 IS NULL
    OR me.followed_at < $2
    OR (me.followed_at = $2 AND u.user_id > $3)
  )
ORDER BY me.followed_at DESC, u.user_id ASC
LIMIT $4`

	queryListFollowers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), me.followed_at, EXISTS(SELECT 1 FROM user_relations mine WHERE mine.user_id = $1 AND mine.target_user_id = me.user_id AND mine.is_following = TRUE), u.presence_status, u.last_online, COALESCE(u.last_online, NOW())
FROM user_relations me
JOIN users u ON u.user_id = me.user_id
WHERE me.target_user_id = $1 AND me.is_following = TRUE
  AND (
    $2 IS NULL
    OR me.followed_at < $2
    OR (me.followed_at = $2 AND u.user_id > $3)
  )
ORDER BY me.followed_at DESC, u.user_id ASC
LIMIT $4`

	queryListUserGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.is_searchable, g.max_members, gm.joined_at
FROM group_members gm
JOIN groups g ON g.group_id = gm.group_id
WHERE gm.user_id = $1 AND g.deleted_at IS NULL AND g.is_searchable = TRUE
  AND (
    $2 IS NULL
    OR gm.joined_at < $2
    OR (gm.joined_at = $2 AND g.group_id > $3)
  )
ORDER BY gm.joined_at DESC, g.group_id ASC
LIMIT $4`

	queryListUserMoments = `
SELECT moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level
FROM moments
WHERE user_id = $1
  AND ($2 OR privacy_level <> 'private')
  AND (
    $3 IS NULL
    OR created_at < $3
    OR (created_at = $3 AND moment_id < $4)
  )
ORDER BY created_at DESC, moment_id DESC
LIMIT $5`

	queryListLikedMoments = `
SELECT m.moment_id, m.user_id, m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, ml.liked_at
FROM moment_likes ml
JOIN moments m ON m.moment_id = ml.moment_id
WHERE ml.user_id = $1
  AND ($2 = '' OR m.user_id = $2 OR m.privacy_level <> 'private')
  AND (
    $3 IS NULL
    OR ml.liked_at < $3
    OR (ml.liked_at = $3 AND m.moment_id < $4)
  )
ORDER BY ml.liked_at DESC, m.moment_id DESC
LIMIT $5`

	queryCountCommonGroups = `
SELECT COUNT(*)
FROM (
	SELECT gm.group_id
	FROM group_members gm
	JOIN groups g ON g.group_id = gm.group_id
	WHERE gm.user_id = $1 AND g.deleted_at IS NULL
	INTERSECT
	SELECT gm.group_id
	FROM group_members gm
	JOIN groups g ON g.group_id = gm.group_id
	WHERE gm.user_id = $2 AND g.deleted_at IS NULL
) common_groups`

	queryUserExists = `
SELECT 1 FROM users WHERE user_id = $1`

	momenttypesQueryListMomentTagsByMomentID = `
SELECT tag_name
FROM moment_tags
WHERE moment_id = $1
ORDER BY tag_name ASC`

	momenttypesQueryListMomentTagsByMomentIDs = `
SELECT moment_id, tag_name
FROM moment_tags
WHERE moment_id = ANY($1)
ORDER BY moment_id, tag_name ASC`
)
