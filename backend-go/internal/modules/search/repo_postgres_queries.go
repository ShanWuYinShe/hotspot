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
  AND mt.tag_name ILIKE '%' || $1 || '%'
GROUP BY mt.tag_name
ORDER BY usage_count DESC
LIMIT $2`

	// querySearchSuggestUsers computes the Haversine distance in SQL and
	// filters by radius there ($4 = 0 disables the radius filter). Previously
	// every row was pulled into Go and filtered with distanceFrom, which
	// violated the "distance/range must happen in PostgreSQL" rule.
	querySearchSuggestUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.bio, ''), u.location_name, u.latitude, u.longitude,
       CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(u.latitude - $2) / 2), 2) + COS(RADIANS($2)) * COS(RADIANS(u.latitude)) * POWER(SIN(RADIANS(u.longitude - $3) / 2), 2))) AS INTEGER) AS distance_meters
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND us.privacy_level = 'public'
  AND ($4 = 0 OR CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(u.latitude - $2) / 2), 2) + COS(RADIANS($2)) * COS(RADIANS(u.latitude)) * POWER(SIN(RADIANS(u.longitude - $3) / 2), 2))) AS INTEGER) <= $4)
  AND (
    u.nickname ILIKE '%' || $1 || '%'
    OR (us.nickname_searchable = TRUE AND u.bio ILIKE '%' || $1 || '%')
  )
ORDER BY distance_meters ASC
LIMIT $5`

	querySearchSuggestGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.location_name, g.latitude, g.longitude, COUNT(gm.user_id) AS members_count,
       CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(g.latitude - $2) / 2), 2) + COS(RADIANS($2)) * COS(RADIANS(g.latitude)) * POWER(SIN(RADIANS(g.longitude - $3) / 2), 2))) AS INTEGER) AS distance_meters
FROM groups g
LEFT JOIN group_members gm ON gm.group_id = g.group_id
WHERE g.deleted_at IS NULL
  AND g.is_searchable = TRUE
  AND g.name ILIKE '%' || $1 || '%'
  AND ($4 = 0 OR CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(g.latitude - $2) / 2), 2) + COS(RADIANS($2)) * COS(RADIANS(g.latitude)) * POWER(SIN(RADIANS(g.longitude - $3) / 2), 2))) AS INTEGER) <= $4)
GROUP BY g.group_id, g.name, g.avatar_url, g.description, g.location_name, g.latitude, g.longitude
ORDER BY distance_meters ASC
LIMIT $5`

	querySearchSuggestMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.location_name, m.latitude, m.longitude,
       (m.likes_count + m.comments_count + m.shares_count) AS popularity,
       CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(m.latitude - $2) / 2), 2) + COS(RADIANS($2)) * COS(RADIANS(m.latitude)) * POWER(SIN(RADIANS(m.longitude - $3) / 2), 2))) AS INTEGER) AS distance_meters
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND m.description ILIKE '%' || $1 || '%'
  AND ($4 = 0 OR CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(m.latitude - $2) / 2), 2) + COS(RADIANS($2)) * COS(RADIANS(m.latitude)) * POWER(SIN(RADIANS(m.longitude - $3) / 2), 2))) AS INTEGER) <= $4)
ORDER BY popularity DESC
LIMIT $5`
)
