package search

const (
	querySearchSuggestTags = `
SELECT mt.tag_name, COUNT(*) AS usage_count
FROM moment_tags mt
WHERE mt.tag_name ILIKE '%' || $1 || '%'
GROUP BY mt.tag_name`

	querySearchSuggestUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.bio, ''), u.location_name, u.latitude, u.longitude
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND (us.nickname_searchable = TRUE OR u.bio <> '')
  AND (u.nickname ILIKE '%' || $1 || '%' OR u.bio ILIKE '%' || $1 || '%')`

	querySearchSuggestGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.location_name, g.latitude, g.longitude, COUNT(gm.user_id) AS members_count
FROM groups g
LEFT JOIN group_members gm ON gm.group_id = g.group_id
WHERE g.is_searchable = TRUE
  AND g.name ILIKE '%' || $1 || '%'
GROUP BY g.group_id, g.name, g.avatar_url, g.description, g.location_name, g.latitude, g.longitude`

	querySearchSuggestMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.location_name, m.latitude, m.longitude,
       (m.likes_count + m.comments_count + m.shares_count) AS popularity
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND m.description ILIKE '%' || $1 || '%'`
)
