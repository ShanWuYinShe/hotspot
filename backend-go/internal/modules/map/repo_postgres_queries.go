package mapmodule

// maxMapScanRows bounds the number of candidates each map-canvas query loads.
// Distance is computed and ordered in SQL now, so the Go-side merge only
// combines the per-type top-N (already sorted by distance_meters ASC).
const maxMapScanRows = 500

const (
	// queryMapCanvasUsers computes distance_meters in SQL and orders by it so
	// the Go layer no longer recomputes distance for sorting. Previously
	// newMapCandidate called geo.DistanceMeters for every row, violating the
	// "distance/sort must happen in PostgreSQL" rule.
	queryMapCanvasUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.location_name, ''), u.latitude, u.longitude, u.created_at,
       CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(u.latitude - $3) / 2), 2) + COS(RADIANS($3)) * COS(RADIANS(u.latitude)) * POWER(SIN(RADIANS(u.longitude - $4) / 2), 2))) AS INTEGER) AS distance_meters
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)
ORDER BY distance_meters ASC
LIMIT $5`

	queryMapCanvasGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.location_name, ''), g.latitude, g.longitude,
       CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(g.latitude - $3) / 2), 2) + COS(RADIANS($3)) * COS(RADIANS(g.latitude)) * POWER(SIN(RADIANS(g.longitude - $4) / 2), 2))) AS INTEGER) AS distance_meters
FROM groups g
WHERE g.is_searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)
ORDER BY distance_meters ASC
LIMIT $5`

	queryMapCanvasMoments = `
SELECT m.moment_id, m.description, COALESCE(m.location_name, ''), m.latitude, m.longitude, m.created_at,
       CAST(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(m.latitude - $3) / 2), 2) + COS(RADIANS($3)) * COS(RADIANS(m.latitude)) * POWER(SIN(RADIANS(m.longitude - $4) / 2), 2))) AS INTEGER) AS distance_meters
FROM moments m
WHERE m.privacy_level = 'public'
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)
ORDER BY distance_meters ASC
LIMIT $5`
)
