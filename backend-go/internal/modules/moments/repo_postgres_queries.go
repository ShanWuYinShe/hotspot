package moments

const (
	queryGetUserNickname = `
SELECT COALESCE(nickname, '')
FROM users
WHERE user_id = $1`

	queryGetMoment = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted
FROM moments m
LEFT JOIN users u ON u.user_id = m.user_id
WHERE m.moment_id = $1`

	queryListMyMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted
FROM moments m
LEFT JOIN users u ON u.user_id = m.user_id
WHERE m.user_id = $1
  AND ($2::timestamptz IS NULL
       OR (m.created_at, m.moment_id) < ($2::timestamptz, $3))
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $4`

	queryInsertMoment = `
INSERT INTO moments (moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level, s2_cell_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`

	queryInsertMomentTagAssignment = `
INSERT INTO moment_tags (moment_id, tag_name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (moment_id, tag_name) DO NOTHING`

	queryFilterAllowedMomentTags = `
SELECT candidate.tag_name
FROM unnest($1::text[]) AS candidate(tag_name)
LEFT JOIN admin_blocked_tags bt ON bt.name = candidate.tag_name
WHERE bt.name IS NULL
ORDER BY candidate.tag_name ASC`

	queryListMomentTagsByMomentID = `
SELECT tag_name
FROM moment_tags
WHERE moment_id = $1
ORDER BY tag_name ASC`

	queryListMomentTagsByMomentIDs = `
SELECT moment_id, tag_name
FROM moment_tags
WHERE moment_id = ANY($1)
ORDER BY moment_id, tag_name ASC`

	queryDeleteMomentTagsByMomentID = `
DELETE FROM moment_tags
WHERE moment_id = $1`

	queryInsertMomentMention = `
INSERT INTO moment_mentions (moment_id, mentioned_user_id, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (moment_id, mentioned_user_id) DO NOTHING`

	queryListMomentMentionsByMomentID = `
SELECT mentioned_user_id
FROM moment_mentions
WHERE moment_id = $1
ORDER BY mentioned_user_id ASC`

	queryListMomentMentionsByMomentIDs = `
SELECT moment_id, mentioned_user_id
FROM moment_mentions
WHERE moment_id = ANY($1)
ORDER BY moment_id, mentioned_user_id ASC`

	queryDeleteMomentMentionsByMomentID = `
DELETE FROM moment_mentions
WHERE moment_id = $1`

	queryIncrementMomentCommentCount = `
UPDATE moments
SET comments_count = comments_count + 1
	WHERE moment_id = $1`

	queryDecrementMomentCommentCount = `
UPDATE moments
SET comments_count = GREATEST(comments_count - 1, 0)
WHERE moment_id = $1`

	queryIncrementMomentShareCount = `
UPDATE moments
SET shares_count = shares_count + 1
WHERE moment_id = $1`

	queryUpdateMoment = `
UPDATE moments
SET description = COALESCE($2, description),
    privacy_level = COALESCE($3, privacy_level),
    location_name = COALESCE($4, location_name),
    latitude      = COALESCE($5, latitude),
    longitude     = COALESCE($6, longitude),
    s2_cell_id    = COALESCE($7, s2_cell_id)
WHERE moment_id = $1`

	queryDeleteMoment = `
UPDATE moments
SET is_deleted = TRUE, description = ''
WHERE moment_id = $1`

	queryCheckMomentFavorite = `
SELECT 1
FROM moment_favorites
WHERE user_id = $1 AND moment_id = $2`

	queryInsertMomentFavorite = `
INSERT INTO moment_favorites (user_id, moment_id, favorited_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, moment_id) DO NOTHING`

	queryDeleteMomentFavorite = `
DELETE FROM moment_favorites
WHERE user_id = $1 AND moment_id = $2`

	queryListMomentFavoriteUserIDs = `
SELECT user_id
FROM moment_favorites
WHERE moment_id = $1`

	queryListFavoriteMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted, mf.favorited_at
FROM moment_favorites mf
JOIN moments m ON m.moment_id = mf.moment_id
LEFT JOIN users u ON u.user_id = m.user_id
WHERE mf.user_id = $1
  AND (
    $2 IS NULL
    OR mf.favorited_at < $2
    OR (mf.favorited_at = $2 AND m.moment_id < $3)
  )
ORDER BY mf.favorited_at DESC, m.moment_id DESC
LIMIT $4`

	// queryFollowingMoments 关注动态（spec moment.md → moments/fetch_following，
	// 原 feed/following 迁移）：keyset 谓词与 LIMIT 直接下推 SQL。
	queryFollowingMoments = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = $1),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)
FROM moments m
JOIN users u ON u.user_id = m.user_id
JOIN user_relations ur ON ur.target_user_id = m.user_id AND ur.user_id = $1 AND ur.is_following = TRUE
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  -- 双向拉黑过滤（对齐私聊/评论口径）：任何一方拉黑即互不可见
  AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = $1 AND blk.target_user_id = m.user_id AND blk.is_blocked = TRUE)
  AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = m.user_id AND blk.target_user_id = $1 AND blk.is_blocked = TRUE)
  AND ($2::timestamptz IS NULL OR m.created_at < $2::timestamptz OR (m.created_at = $2::timestamptz AND m.moment_id < $3))
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $4`

	// queryNearbyMoments 附近瞬间（spec moment.md → moments/fetch_nearby，
	// 原 feed/nearby_moments 迁移）：S2 覆盖区间粗筛，精筛在 Go 层（geo.SearchPlan）。
	// 终序由 Go 层 compareNearbyCandidates（全序，momentID 兜底）确定，SQL 行序
	// 不影响排序结果；ORDER BY + LIMIT 仅作为确定性防御截断——区间行数无界时
	// 避免 DoS/内存无界（截断表现为"没有更多"，分页不错乱；正常规模不触达）。
	queryNearbyMoments = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       (SELECT COUNT(*)::int FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id),
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = $3),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $3),
       EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $3 AND ur.is_following = TRUE)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  -- 双向拉黑过滤（对齐私聊/评论口径）：任何一方拉黑即互不可见
  AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = $3 AND blk.target_user_id = m.user_id AND blk.is_blocked = TRUE)
  AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = m.user_id AND blk.target_user_id = $3 AND blk.is_blocked = TRUE)
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)
ORDER BY m.created_at DESC
LIMIT 2000`
)

// ── users 模块「按作者聚合的瞬间列表」读能力（原实现位于 users repo，直查
// moments/reactions 表；现已归位本模块，users 经 Service 接口消费）──
const (
	// 点赞数走 reactions 实时 COUNT（审计 F-08 收口）：likes_count 列已无任何
	// 写路径（点赞改走 reactions 表），死列恒为 0，与 nearby/search 口径对齐。
	queryListMomentsByAuthor = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, (SELECT COUNT(*)::int FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id), m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level
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

	queryListReactedByUser = `
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
)
