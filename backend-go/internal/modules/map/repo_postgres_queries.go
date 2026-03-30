package mapmodule

const (
	queryMapCanvasUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.location_name, ''), u.latitude, u.longitude, u.created_at
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND u.latitude BETWEEN $1 AND $2
  AND u.longitude BETWEEN $3 AND $4`

	queryMapCanvasGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.location_name, ''), g.latitude, g.longitude
FROM groups g
WHERE g.is_searchable = TRUE
  AND g.latitude BETWEEN $1 AND $2
  AND g.longitude BETWEEN $3 AND $4`

	queryMapCanvasMoments = `
SELECT m.moment_id, m.description, COALESCE(m.location_name, ''), m.latitude, m.longitude, m.created_at
FROM moments m
WHERE m.privacy_level = 'public'
  AND m.latitude BETWEEN $1 AND $2
  AND m.longitude BETWEEN $3 AND $4`
)
