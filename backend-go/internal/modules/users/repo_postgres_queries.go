package users

const (
	queryGetUserProfile = `
SELECT user_id, nickname, COALESCE(avatar_url, ''), user_type, COALESCE(bio, ''), COALESCE(gender, ''), COALESCE(email, ''), COALESCE(phone, ''), location_name, latitude, longitude, presence_status, last_online, created_at, is_deleted
FROM users
WHERE user_id = $1`

	// queryGetUserProfiles 与 queryGetUserProfile 同列，供 GetProfiles 一次取多行。
	queryGetUserProfiles = `
SELECT user_id, nickname, COALESCE(avatar_url, ''), user_type, COALESCE(bio, ''), COALESCE(gender, ''), COALESCE(email, ''), COALESCE(phone, ''), location_name, latitude, longitude, presence_status, last_online, created_at, is_deleted
FROM users
WHERE user_id = ANY($1)`

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

	// queryUpdatePresenceStatus 是 presence 的运行时写路径（WS 首条连接建立置
	// online、最后一条连接断开置 offline）。此前 presence_status 只有注册 INSERT、
	// 资料 upsert 覆写、注销置 offline 三个非运行时写点，groups/search 等处的
	// presence_status='online' 统计恒错。last_online 与 presence_status 是一对，
	// 写 presence 时一并落到 NOW()（「最后在线」随之有了真实来源）。
	queryUpdatePresenceStatus = `
UPDATE users
SET presence_status = $2,
    last_online = NOW()
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
	//
	// DO UPDATE 带 `WHERE is_following = FALSE` 守卫 + RETURNING：已关注时命中 0 行
	// （pgx.ErrNoRows），服务层据此跳过重复通知——此前 upsert 恒成功，反复
	// follow/unfollow 每轮都会再落一条 new_follower（并可能重复 mutual_follow）
	// 通知（审计 F-12）。返回行即表示确实发生了「未关注 → 关注」的状态迁移
	// （新插入或取消关注后重新关注，两者都应通知）。
	queryUpsertUserRelationFollow = `
INSERT INTO user_relations (user_id, target_user_id, is_following, followed_at)
VALUES ($1,$2,TRUE,$3)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET
is_following = TRUE, followed_at = EXCLUDED.followed_at
WHERE user_relations.is_following = FALSE
RETURNING is_following`

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

	// queryGetBlockedEitherWay 一次往返解析 a、b 间的双向拉黑状态。
	// 无关系行时聚合返回单行 NULL，经 COALESCE 归一为 false。
	queryGetBlockedEitherWay = `
SELECT
  COALESCE(bool_or(is_blocked) FILTER (WHERE user_id = $1), false),
  COALESCE(bool_or(is_blocked) FILTER (WHERE user_id = $2), false)
FROM user_relations
WHERE (user_id = $1 AND target_user_id = $2) OR (user_id = $2 AND target_user_id = $1)`

	// queryListUserRelationsForward fetches the forward-edge rows
	// (user_id = $1 → target_user_id = ANY($2)) for a batched
	// relationship lookup. Used by RelationshipBatch to avoid an N+1.
	queryListUserRelationsForward = `
SELECT target_user_id, is_following, is_blocked
FROM user_relations
WHERE user_id = $1 AND target_user_id = ANY($2)`

	// queryListUserRelationsReverse fetches the reverse-edge rows
	// (target_user_id = $1 → user_id = ANY($2)) for a batched
	// relationship lookup so RelationshipBatch can resolve IsFollowedBy
	// and TargetBlocked (对方是否拉黑了查看者).
	queryListUserRelationsReverse = `
SELECT user_id, is_following, is_blocked
FROM user_relations
WHERE target_user_id = $1 AND user_id = ANY($2)`

	queryCountUserStats = `
SELECT
  (SELECT COUNT(*) FROM moments m WHERE m.user_id = $1 AND m.is_deleted = FALSE),
  (SELECT COUNT(*) FROM user_relations ur JOIN user_relations rev ON rev.user_id = ur.target_user_id AND rev.target_user_id = ur.user_id AND rev.is_following = TRUE WHERE ur.user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM user_relations ur WHERE ur.target_user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM user_relations ur WHERE ur.user_id = $1 AND ur.is_following = TRUE),
  (SELECT COUNT(*) FROM group_members gm JOIN groups g ON g.group_id = gm.group_id WHERE gm.user_id = $1 AND g.deleted_at IS NULL)`

	// queryMutualFriends 双向共同好友一次查全：总数经 COUNT(*) OVER 随行返回
	//（LIMIT 只截预览，不截总数），预览按最近互关优先。
	queryMutualFriends = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COUNT(*) OVER() AS total
FROM user_relations a1
JOIN user_relations a2 ON a2.user_id = a1.target_user_id AND a2.target_user_id = a1.user_id AND a2.is_following = TRUE
JOIN user_relations b1 ON b1.target_user_id = a1.target_user_id
JOIN user_relations b2 ON b2.user_id = b1.target_user_id AND b2.target_user_id = b1.user_id AND b2.is_following = TRUE
JOIN users u ON u.user_id = a1.target_user_id
WHERE a1.user_id = $1 AND a1.is_following = TRUE
  AND b1.user_id = $2 AND b1.is_following = TRUE
ORDER BY GREATEST(a1.followed_at, a2.followed_at) DESC, u.user_id ASC
LIMIT $3`

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
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), me.followed_at, u.presence_status, u.last_online, COALESCE(u.last_online, NOW())
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
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), me.followed_at, u.presence_status, u.last_online, COALESCE(u.last_online, NOW())
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

	queryUserExists = `
SELECT 1 FROM users WHERE user_id = $1`

	// queryListBlockedUserTags 返回 tags 中命中平台屏蔽标签（admin_blocked_tags）的标签名（update_account 10400）。
	queryListBlockedUserTags = `
SELECT bt.name
FROM unnest($1::text[]) AS candidate(tag_name)
JOIN admin_blocked_tags bt ON bt.name = candidate.tag_name
ORDER BY bt.name ASC`
)
