package feed

const (
	queryFeedUsers = `
SELECT u.user_id,
       u.nickname,
       COALESCE(u.avatar_url, ''),
       COALESCE(u.bio, ''),
       u.location_name,
       u.latitude,
       u.longitude,
       EXISTS(SELECT 1 FROM user_relations mine WHERE mine.user_id = $5 AND mine.target_user_id = u.user_id AND mine.is_following = TRUE),
       EXISTS(SELECT 1 FROM user_relations rev WHERE rev.user_id = u.user_id AND rev.target_user_id = $5 AND rev.is_following = TRUE),
       COALESCE((
         SELECT COUNT(*)
         FROM user_relations mine
         JOIN user_relations theirs ON theirs.user_id = u.user_id AND theirs.target_user_id = mine.target_user_id AND theirs.is_following = TRUE
         WHERE mine.user_id = $5 AND mine.is_following = TRUE
       ), 0),
       COALESCE((
         SELECT COUNT(*)
         FROM group_members gm1
         JOIN group_members gm2 ON gm2.group_id = gm1.group_id AND gm2.user_id = u.user_id
         JOIN groups g ON g.group_id = gm1.group_id
         WHERE gm1.user_id = $5 AND g.deleted_at IS NULL
       ), 0)
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND us.location_visible = TRUE
  AND us.privacy_level = 'public'
  AND u.latitude BETWEEN $1 AND $2
  AND u.longitude BETWEEN $3 AND $4`

	queryFeedGroups = `
SELECT g.group_id,
       g.name,
       COALESCE(g.avatar_url, ''),
       COALESCE(g.description, ''),
       g.has_password,
       g.location_name,
       g.latitude,
       g.longitude,
       COUNT(gm.user_id) AS members_count,
       EXISTS(SELECT 1 FROM group_members viewer_gm WHERE viewer_gm.group_id = g.group_id AND viewer_gm.user_id = $5),
       COALESCE((
         SELECT COUNT(*)
         FROM group_members gm_friend
         JOIN user_relations ur ON ur.target_user_id = gm_friend.user_id AND ur.user_id = $5 AND ur.is_following = TRUE
         WHERE gm_friend.group_id = g.group_id
       ), 0)
FROM groups g
LEFT JOIN group_members gm ON gm.group_id = g.group_id
WHERE g.deleted_at IS NULL
  AND g.is_searchable = TRUE
  AND g.latitude BETWEEN $1 AND $2
  AND g.longitude BETWEEN $3 AND $4
GROUP BY g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude`

	queryFeedMomentsFollowing = `
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
       EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.moment_id AND ml.user_id = $1),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)
FROM moments m
JOIN users u ON u.user_id = m.user_id
JOIN user_relations ur ON ur.target_user_id = m.user_id AND ur.user_id = $1 AND ur.is_following = TRUE
WHERE m.privacy_level = 'public'`

	queryFeedMomentsNearby = `
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
       EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.moment_id AND ml.user_id = $5),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $5),
       EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $5 AND ur.is_following = TRUE)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND m.latitude BETWEEN $1 AND $2
  AND m.longitude BETWEEN $3 AND $4`

	queryFeedMomentsAll = `
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
       EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.moment_id AND ml.user_id = $5),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $5),
       EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $5 AND ur.is_following = TRUE)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND (
    (m.latitude BETWEEN $1 AND $2 AND m.longitude BETWEEN $3 AND $4)
    OR EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $5 AND ur.is_following = TRUE)
  )`

	queryFeedMomentTags = `
SELECT tag_name
FROM moment_tags
WHERE moment_id = $1
ORDER BY tag_name ASC`

	queryFeedTrendingTags = `
SELECT mt.tag_name, COUNT(*) AS usage_count
FROM moment_tags mt
JOIN moments m ON m.moment_id = mt.moment_id
WHERE m.privacy_level = 'public'
GROUP BY mt.tag_name`

	queryFeedTrendingHotMoments = `
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
       EXISTS(SELECT 1 FROM moment_likes ml WHERE ml.moment_id = m.moment_id AND ml.user_id = $1),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'`
)
