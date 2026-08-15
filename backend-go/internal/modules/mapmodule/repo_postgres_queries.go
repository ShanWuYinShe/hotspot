package mapmodule

// maxMapScanRows bounds the number of candidates each map-canvas query loads
// before the Go-side distance sort + pagination. Without it a wide viewport
// would pull every matching row into memory. 500 covers a dense cluster while
// keeping the worst case predictable.
const maxMapScanRows = 500

const (
	// 确定性排序（主键 + 时间 tie-breaker）：LIMIT 截断的候选子集必须稳定，
	// 否则热力图聚合结果抖动、hotspots 跨页出现重复/丢失。
	queryMapCanvasUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.location_name, ''), u.latitude, u.longitude, u.created_at
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)
ORDER BY u.user_id ASC
LIMIT $3`

	queryMapCanvasGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.location_name, ''), g.latitude, g.longitude
FROM groups g
WHERE g.is_searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)
ORDER BY g.group_id ASC
LIMIT $3`

	// 时间窗在 SQL 内过滤（LIMIT 之后于 Go 侧过滤会先截断后过滤，历史时间窗丢数据）；
	// created_at DESC 让截断优先保留更新的瞬间。
	queryMapCanvasMoments = `
SELECT m.moment_id, m.description, COALESCE(m.location_name, ''), m.latitude, m.longitude, m.created_at
FROM moments m
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)
  AND ($4::timestamptz IS NULL OR m.created_at >= $4)
  AND ($5::timestamptz IS NULL OR m.created_at <= $5)
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $3`
)
