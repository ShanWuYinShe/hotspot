package mapmodule

// maxMapScanRows bounds the number of candidates each map-canvas query loads
// before the Go-side distance sort + pagination. Without it a wide viewport
// would pull every matching row into memory. 500 covers a dense cluster while
// keeping the worst case predictable.
const maxMapScanRows = 500

const (
	queryMapCanvasUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.location_name, ''), u.latitude, u.longitude, u.created_at
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)
LIMIT $3`

	queryMapCanvasGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.location_name, ''), g.latitude, g.longitude
FROM groups g
WHERE g.is_searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)
LIMIT $3`

	queryMapCanvasMoments = `
SELECT m.moment_id, m.description, COALESCE(m.location_name, ''), m.latitude, m.longitude, m.created_at
FROM moments m
WHERE m.privacy_level = 'public'
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)
LIMIT $3`
)
