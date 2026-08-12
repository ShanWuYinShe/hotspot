package users

const (
	queryGetUserProfile = `
SELECT user_id, nickname, COALESCE(avatar_url, ''), user_type, COALESCE(bio, ''), COALESCE(gender, ''), COALESCE(email, ''), COALESCE(phone, ''), location_name, latitude, longitude, presence_status, last_online, created_at, is_deleted
FROM users
WHERE user_id = $1`

	queryListUserProfileTags = `
SELECT tag_name
FROM user_profile_tags
WHERE user_id = $1
ORDER BY tag_name ASC`

	// queryListUserProfileTagsBatch fetches profile tags for many user_ids in
	// one query so SearchProfiles can avoid an N+1 of loadUserProfileTags
	// calls inside the rows scan loop.
	queryListUserProfileTagsBatch = `
SELECT user_id, tag_name
FROM user_profile_tags
WHERE user_id = ANY($1)
ORDER BY user_id, tag_name ASC`

	queryReplaceUserProfileTags = `
DELETE FROM user_profile_tags WHERE user_id = $1`

	queryInsertUserProfileTag = `
INSERT INTO user_profile_tags (user_id, tag_name)
VALUES ($1, $2)
ON CONFLICT (user_id, tag_name) DO NOTHING`

	// queryInsertUserProfileTagsBatch inserts all tags for a user in one round
	// trip via unnest, replacing the previous per-tag loop. Used inside the
	// PutProfile transaction.
	queryInsertUserProfileTagsBatch = `
INSERT INTO user_profile_tags (user_id, tag_name)
SELECT $1, tag FROM unnest($2::text[]) AS tag
ON CONFLICT (user_id, tag_name) DO NOTHING`

	queryUpsertUserProfile = `
INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, presence_status, last_online, created_at, s2_cell_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::timestamptz,$14::timestamptz,$15)
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
last_online = EXCLUDED.last_online,
s2_cell_id = EXCLUDED.s2_cell_id`

	// 账户删除匿名化（spec api_spec_user.md「数据保留策略」）：users 行保留
	// （user_id 关联不断、内容因不触发级联而保留），昵称/头像等替换为占位，
	// is_deleted 标记使已注销用户不再可被搜索/访问。
	queryAnonymizeUserProfile = `
UPDATE users
SET nickname = '已注销用户',
    avatar_url = NULL,
    bio = NULL,
    gender = NULL,
    email = NULL,
    phone = NULL,
    location_name = '',
    presence_status = 'offline',
    is_deleted = TRUE
WHERE user_id = $1`

	queryDeleteAuthAccountByUserID = `DELETE FROM auth_accounts WHERE user_id = $1`

	queryRevokeAllUserSessions = `DELETE FROM auth_sessions WHERE user_id = $1`

	queryDeleteUserSettingsByUserID = `DELETE FROM user_settings WHERE user_id = $1`

	queryDeleteUserRelations = `DELETE FROM user_relations WHERE user_id = $1 OR target_user_id = $1`

	queryDeleteUserProfileTagsByUserID = `DELETE FROM user_profile_tags WHERE user_id = $1`

	queryDeleteUserFavorites = `DELETE FROM moment_favorites WHERE user_id = $1`

	queryDeleteUserNotifications = `DELETE FROM notifications WHERE user_id = $1`

	queryDeleteUserPrivateChats = `
DELETE FROM private_chats
WHERE user_a_id = $1 OR user_b_id = $1`

	queryDeleteUserUploadSessions = `DELETE FROM media_upload_sessions WHERE user_id = $1`

	queryGetUserSettings = `
SELECT privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, reacted_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable, COALESCE(nearby_moment_notify, FALSE)
FROM user_settings
WHERE user_id = $1`

	queryGetUserSettingsBatch = `
SELECT user_id, privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, reacted_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable, COALESCE(nearby_moment_notify, FALSE)
FROM user_settings
WHERE user_id = ANY($1)`

	queryUpsertUserSettings = `
INSERT INTO user_settings (user_id, privacy_level, location_visible, presence_visible, social_lists_visible, groups_visible, reacted_moments_visible, stats_visible, allow_follow, allow_private_chat, searchable, phone_searchable, nickname_searchable, location_searchable, nearby_moment_notify)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (user_id) DO UPDATE SET
privacy_level = EXCLUDED.privacy_level,
location_visible = EXCLUDED.location_visible,
presence_visible = EXCLUDED.presence_visible,
social_lists_visible = EXCLUDED.social_lists_visible,
groups_visible = EXCLUDED.groups_visible,
reacted_moments_visible = EXCLUDED.reacted_moments_visible,
stats_visible = EXCLUDED.stats_visible,
allow_follow = EXCLUDED.allow_follow,
allow_private_chat = EXCLUDED.allow_private_chat,
searchable = EXCLUDED.searchable,
phone_searchable = EXCLUDED.phone_searchable,
nickname_searchable = EXCLUDED.nickname_searchable,
location_searchable = EXCLUDED.location_searchable,
nearby_moment_notify = EXCLUDED.nearby_moment_notify`

	// 关系操作的条件更新：每个操作只写自己的字段，避免 read-modify-write
	// 的 lost update（并发 Follow+Block 互相覆盖对方字段）。
	queryUpsertUserRelationFollow = `
INSERT INTO user_relations (user_id, target_user_id, is_following, followed_at)
VALUES ($1,$2,TRUE,$3)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
is_following = TRUE, followed_at = EXCLUDED.followed_at`

	queryUpsertUserRelationUnfollow = `
INSERT INTO user_relations (user_id, target_user_id, is_following)
VALUES ($1,$2,FALSE)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
is_following = FALSE, followed_at = NULL`

	queryUpsertUserRelationBlock = `
INSERT INTO user_relations (user_id, target_user_id, is_blocked, blocked_at, block_reason)
VALUES ($1,$2,TRUE,$3,$4)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
is_blocked = TRUE, blocked_at = EXCLUDED.blocked_at, block_reason = EXCLUDED.block_reason`

	queryUpsertUserRelationUnblock = `
INSERT INTO user_relations (user_id, target_user_id, is_blocked)
VALUES ($1,$2,FALSE)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
is_blocked = FALSE, blocked_at = NULL, block_reason = NULL`

	queryUpsertUserRelationRemark = `
INSERT INTO user_relations (user_id, target_user_id, remark)
VALUES ($1,$2,$3)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
remark = EXCLUDED.remark`

	queryGetUserRelation = `
SELECT is_following, is_blocked, COALESCE(remark, ''), followed_at, blocked_at, COALESCE(block_reason, '')
FROM user_relations
WHERE user_id = $1 AND target_user_id = $2`

	// queryListUserRelationsForward fetches the forward-edge rows
	// (user_id = $1 → target_user_id = ANY($2)) for a batched
	// relationship lookup. Used by RelationshipBatch to avoid an N+1.
	queryListUserRelationsForward = `
SELECT target_user_id, is_following, is_blocked
FROM user_relations
WHERE user_id = $1 AND target_user_id = ANY($2)`

	// queryListUserRelationsReverse fetches the reverse-edge rows
	// (target_user_id = $1 → user_id = ANY($2)) for a batched
	// relationship lookup so RelationshipBatch can resolve IsFollowedBy.
	queryListUserRelationsReverse = `
SELECT user_id, is_following
FROM user_relations
WHERE target_user_id = $1 AND user_id = ANY($2)`

	queryCountUserStats = `
SELECT
  (SELECT COUNT(*) FROM moments m WHERE m.user_id = $1 AND m.is_deleted = FALSE),
  (SELECT COUNT(*) FROM user_relations ur JOIN user_relations rev ON rev.user_id = ur.target_user_id AND rev.target_user_id = ur.user_id AND rev.is_following = TRUE WHERE ur.user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM user_relations ur WHERE ur.target_user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM user_relations ur WHERE ur.user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM group_members gm JOIN groups g ON g.group_id = gm.group_id WHERE gm.user_id = $1 AND g.deleted_at IS NULL)`

	queryListBlockedUsers = `
SELECT target_user_id, blocked_at, COALESCE(block_reason, '')
FROM user_relations
WHERE user_id = $1 AND is_blocked = TRUE
  AND (
    $2::timestamptz IS NULL
    OR blocked_at < $2::timestamptz
    OR (blocked_at = $2::timestamptz AND target_user_id > $3)
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
    $2::timestamptz IS NULL
    OR GREATEST(me.followed_at, rev.followed_at) < $2::timestamptz
    OR (GREATEST(me.followed_at, rev.followed_at) = $2::timestamptz AND u.user_id > $3)
  )
ORDER BY GREATEST(me.followed_at, rev.followed_at) DESC, u.user_id ASC
LIMIT $4`

	queryListFollowing = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), me.followed_at, EXISTS(SELECT 1 FROM user_relations rev WHERE rev.user_id = me.target_user_id AND rev.target_user_id = me.user_id AND rev.is_following = TRUE), u.presence_status, u.last_online, COALESCE(u.last_online, NOW())
FROM user_relations me
JOIN users u ON u.user_id = me.target_user_id
WHERE me.user_id = $1 AND me.is_following = TRUE
  AND (
    $2::timestamptz IS NULL
    OR me.followed_at < $2::timestamptz
    OR (me.followed_at = $2::timestamptz AND u.user_id > $3)
  )
ORDER BY me.followed_at DESC, u.user_id ASC
LIMIT $4`

	queryListFollowers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), me.followed_at, EXISTS(SELECT 1 FROM user_relations mine WHERE mine.user_id = $1 AND mine.target_user_id = me.user_id AND mine.is_following = TRUE), u.presence_status, u.last_online, COALESCE(u.last_online, NOW())
FROM user_relations me
JOIN users u ON u.user_id = me.user_id
WHERE me.target_user_id = $1 AND me.is_following = TRUE
  AND (
    $2::timestamptz IS NULL
    OR me.followed_at < $2::timestamptz
    OR (me.followed_at = $2::timestamptz AND u.user_id > $3)
  )
ORDER BY me.followed_at DESC, u.user_id ASC
LIMIT $4`

	queryListUserGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.is_searchable, g.max_members, gm.joined_at
FROM group_members gm
JOIN groups g ON g.group_id = gm.group_id
WHERE gm.user_id = $1 AND g.deleted_at IS NULL AND g.is_searchable = TRUE
  AND (
    $2::timestamptz IS NULL
    OR gm.joined_at < $2::timestamptz
    OR (gm.joined_at = $2::timestamptz AND g.group_id > $3)
  )
ORDER BY gm.joined_at DESC, g.group_id ASC
LIMIT $4`

	queryListUserMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level
FROM moments m
LEFT JOIN users u ON u.user_id = m.user_id
WHERE m.user_id = $1
  AND m.is_deleted = FALSE
  AND ($2 OR m.privacy_level <> 'private')
  AND (
    $3::timestamptz IS NULL
    OR m.created_at < $3::timestamptz
    OR (m.created_at = $3::timestamptz AND m.moment_id < $4)
  )
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $5`

	queryListReactedMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, r.reacted_at
FROM reactions r
JOIN moments m ON m.moment_id = r.target_id
LEFT JOIN users u ON u.user_id = m.user_id
WHERE r.target_type = 'moment' AND r.user_id = $1
  AND m.is_deleted = FALSE
  AND ($2 = '' OR m.user_id = $2 OR m.privacy_level <> 'private')
  AND (
    $3::timestamptz IS NULL
    OR r.reacted_at < $3::timestamptz
    OR (r.reacted_at = $3::timestamptz AND m.moment_id < $4)
  )
ORDER BY r.reacted_at DESC, m.moment_id DESC
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
