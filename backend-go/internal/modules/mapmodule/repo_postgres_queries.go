package mapmodule

// maxMapScanRows bounds the number of candidates each map-canvas query loads
// before the Go-side distance sort + pagination. Without it a wide viewport
// would pull every matching row into memory. 500 covers a dense cluster while
// keeping the worst case predictable.
const maxMapScanRows = 500

const (
	// 确定性排序（主键 + 时间 tie-breaker）：LIMIT 截断的候选子集必须稳定，
	// 否则热力图聚合结果抖动、hotspots 跨页出现重复/丢失。
	// 隐私过滤与 feed 的 queryFeedUsers 对齐（038 迁移：注销用户不再可被
	// 搜索/访问；不公开位置/非 public 用户不得出现在匿名地图上）。
	queryMapCanvasUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.location_name, ''), u.latitude, u.longitude, u.created_at
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND u.is_deleted = FALSE
  AND us.location_visible = TRUE
  AND us.privacy_level = 'public'
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

	/* ── 热力聚合（SQL 直聚合，真实计数）────────────────────────────
	   大尺度视口下候选行远超 LIMIT 截断上限：若把截断后的候选样本拉到 Go 再
	   分桶，桶计数是采样的（每区域只有个位数），用户看到「区域统计不合并」。
	   改为在 SQL 里按 S2 祖先 cell GROUP BY 真实计数——结果集天然有界
	   （≤ 覆盖 cell 数），且 COUNT 是全量真值。
	   $1/$2 覆盖范围对、$3 = ParentShift(level) 移位数；moments 另带时间窗。 */

	queryHeatmapUsers = `
SELECT ((u.s2_cell_id >> $3) << $3) AS bucket_cell,
       count(*) AS total,
       sum(u.latitude) AS lat_sum,
       sum(u.longitude) AS lng_sum
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND u.is_deleted = FALSE
  AND us.location_visible = TRUE
  AND us.privacy_level = 'public'
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)
GROUP BY 1`

	queryHeatmapGroups = `
SELECT ((g.s2_cell_id >> $3) << $3) AS bucket_cell,
       count(*) AS total,
       sum(g.latitude) AS lat_sum,
       sum(g.longitude) AS lng_sum
FROM groups g
WHERE g.is_searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)
GROUP BY 1`

	queryHeatmapMoments = `
SELECT ((m.s2_cell_id >> $4) << $4) AS bucket_cell,
       count(*) AS total,
       sum(m.latitude) AS lat_sum,
       sum(m.longitude) AS lng_sum
FROM moments m
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)
  AND ($5::timestamptz IS NULL OR m.created_at >= $5)
  AND ($6::timestamptz IS NULL OR m.created_at <= $6)
GROUP BY 1`
)
