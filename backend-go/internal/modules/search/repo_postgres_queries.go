package search

// maxSuggestRows bounds the result set of each suggest query. The suggest
// queries use leading-wildcard ILIKE (unanchored, non-indexable), so a short
// or common keyword would otherwise scan and return the whole table. This cap
// keeps autocomplete fast and memory bounded.
const maxSuggestRows = 20

const (
	querySearchSuggestTags = `
SELECT mt.tag_name, COUNT(*) AS usage_count
FROM moment_tags mt
JOIN moments m ON m.moment_id = mt.moment_id
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  AND mt.tag_name ILIKE '%' || $1 || '%'
GROUP BY mt.tag_name
ORDER BY usage_count DESC
LIMIT $2`

	querySearchSuggestUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.bio, ''), u.location_name, u.latitude, u.longitude,
       u.presence_status, u.last_online,
       COALESCE((SELECT ARRAY_AGG(tag_name ORDER BY tag_name) FROM user_profile_tags t WHERE t.user_id = u.user_id), '{}'),
       COALESCE((SELECT COUNT(*) FROM moments m WHERE m.user_id = u.user_id AND m.is_deleted = FALSE), 0),
       COALESCE((SELECT COUNT(*) FROM user_relations ur JOIN user_relations rev ON rev.user_id = ur.target_user_id AND rev.target_user_id = ur.user_id AND rev.is_following = TRUE WHERE ur.user_id = u.user_id AND ur.is_following = TRUE), 0),
       COALESCE((SELECT COUNT(*) FROM user_relations ur WHERE ur.target_user_id = u.user_id AND ur.is_following = TRUE), 0),
       COALESCE((SELECT COUNT(*) FROM user_relations ur WHERE ur.user_id = u.user_id AND ur.is_following = TRUE), 0),
       COALESCE((SELECT COUNT(*) FROM group_members gm JOIN groups g ON g.group_id = gm.group_id WHERE gm.user_id = u.user_id AND g.deleted_at IS NULL), 0)
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND u.is_deleted = FALSE
  AND us.privacy_level = 'public'
  AND (
    u.nickname ILIKE '%' || $1 || '%'
    OR (us.nickname_searchable = TRUE AND u.bio ILIKE '%' || $1 || '%')
  )
LIMIT $2`

	querySearchSuggestGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.location_name, g.latitude, g.longitude, COUNT(gm.user_id) AS members_count,
       g.join_method, g.requires_approval, g.tags, g.next_activity_at, g.is_searchable, g.max_members, g.created_at, g.owner_id,
       COALESCE((SELECT COUNT(*) FROM group_members om JOIN users ou ON ou.user_id = om.user_id AND ou.presence_status = 'online' WHERE om.group_id = g.group_id), 0)
FROM groups g
LEFT JOIN group_members gm ON gm.group_id = g.group_id
WHERE g.deleted_at IS NULL
  AND g.is_searchable = TRUE
  AND g.name ILIKE '%' || $1 || '%'
GROUP BY g.group_id, g.name, g.avatar_url, g.description, g.location_name, g.latitude, g.longitude,
         g.join_method, g.requires_approval, g.tags, g.next_activity_at, g.is_searchable, g.max_members, g.created_at, g.owner_id
LIMIT $2`

	querySearchSuggestMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.location_name, m.latitude, m.longitude,
       (m.likes_count + m.comments_count + m.shares_count) AS popularity
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  AND m.description ILIKE '%' || $1 || '%'
ORDER BY popularity DESC
LIMIT $2`
)
