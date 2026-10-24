package admin

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

// StatsGeo 统计分析聚合（spec api_spec_admin.md「统计分析（stats）API」）：
// 一次往返取全站存量 + 区域 Top-N + 热门地点 Top-N。三条查询顺序执行
// （与 map 模块同款决策：并发会按请求占用多条池连接，得不偿失）。
func (r *PostgresRepository) StatsGeo(ctx context.Context, query geoStatsQuery) (AdminGeoStatsResponse, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return AdminGeoStatsResponse{}, persistence.ErrNotImplemented
	}

	resp := AdminGeoStatsResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Totals:      AdminGeoStatsTotals{},
		Regions:     []AdminRegionRankItem{},
		Places:      []AdminPlaceRankItem{},
		Trends:      []AdminGeoStatsTrendPoint{},
	}

	// 全站存量（不受时间窗影响）。
	var userTotal, groupTotal, momentTotal int64
	if err := queryable.QueryRow(ctx, queryAdminStatsTotals).Scan(&userTotal, &groupTotal, &momentTotal); err != nil {
		return AdminGeoStatsResponse{}, err
	}
	resp.Totals = AdminGeoStatsTotals{
		User:   int(userTotal),
		Group:  int(groupTotal),
		Moment: int(momentTotal),
		Total:  int(userTotal + groupTotal + momentTotal),
	}

	// 区域排名：SQL 产出 (bucket, total, user_n, group_n, moment_n)，桶→token/中心点在 Go 侧还原。
	if query.TrendDays > 0 {
		trends, err := r.statsTrends(ctx, queryable, query)
		if err != nil {
			return AdminGeoStatsResponse{}, err
		}
		resp.Trends = trends
	}

	start, end := nullableTimeRange(query.StartTime, query.EndTime)
	rows, err := queryable.Query(ctx, queryAdminStatsRegions, query.Limit, geo.CellLevelShift(query.Level), start, end)
	if err != nil {
		return AdminGeoStatsResponse{}, err
	}
	for rows.Next() {
		var bucket, total, userN, groupN, momentN int64
		if err := rows.Scan(&bucket, &total, &userN, &groupN, &momentN); err != nil {
			rows.Close()
			return AdminGeoStatsResponse{}, err
		}
		item := AdminRegionRankItem{
			CellToken:  geo.BucketCellToken(bucket, query.Level),
			Level:      query.Level,
			TotalCount: int(total),
			Breakdown: StatsEntityBreakdown{
				User:   int(userN),
				Group:  int(groupN),
				Moment: int(momentN),
			},
		}
		center := geo.CellCenter(item.CellToken)
		item.Center = GeoPoint{Latitude: center.Latitude, Longitude: center.Longitude}
		resp.Regions = append(resp.Regions, item)
	}
	if err := rows.Err(); err != nil {
		return AdminGeoStatsResponse{}, err
	}
	rows.Close()

	// 热门地点排名：location_name 精确文本聚合。
	rows, err = queryable.Query(ctx, queryAdminStatsPlaces, query.Limit, start, end)
	if err != nil {
		return AdminGeoStatsResponse{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var total, userN, groupN, momentN int64
		if err := rows.Scan(&name, &total, &userN, &groupN, &momentN); err != nil {
			return AdminGeoStatsResponse{}, err
		}
		resp.Places = append(resp.Places, AdminPlaceRankItem{
			LocationName: name,
			TotalCount:   int(total),
			Breakdown: StatsEntityBreakdown{
				User:   int(userN),
				Group:  int(groupN),
				Moment: int(momentN),
			},
		})
	}
	return resp, rows.Err()
}

// statsTrendsCursorNamespace 是下钻 keyset 游标的编码命名空间（AES-GCM 隔离）。
const statsTrendsCursorNamespace = "admin.stats"

func encodeGeoDrilldownCursor(c geoDrilldownCursor) string {
	return encodeCursor(statsTrendsCursorNamespace, c)
}

func decodeGeoDrilldownCursor(raw string) (geoDrilldownCursor, *httpx.APIError) {
	return decodeCursor[geoDrilldownCursor](statsTrendsCursorNamespace, raw)
}

// fillTrendDays 把查询结果补零展开为连续日序列（TrendStart 起、UTC+8 日界）。
// statsTrends 查询每日新增分布：窗口天数与日期序列全部在 SQL 内基于数据库时钟
// （now()，UTC+8 自然日）生成，避免 Go/DB 时钟偏移在本地午夜附近错位；返回行
// 已按日升序且空日补零，无需 Go 侧再展开。
func (r *PostgresRepository) statsTrends(ctx context.Context, queryable persistence.Queryable, query geoStatsQuery) ([]AdminGeoStatsTrendPoint, error) {
	days := query.TrendDays
	if days <= 0 {
		days = 7
	}
	rows, err := queryable.Query(ctx, queryAdminStatsTrends, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AdminGeoStatsTrendPoint, 0, days)
	for rows.Next() {
		var day time.Time
		var userN, groupN, momentN int64
		if err := rows.Scan(&day, &userN, &groupN, &momentN); err != nil {
			return nil, err
		}
		out = append(out, AdminGeoStatsTrendPoint{
			Date:   day.Format("2006-01-02"),
			User:   int(userN),
			Group:  int(groupN),
			Moment: int(momentN),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PostgresRepository) StatsGeoDrilldown(ctx context.Context, query geoDrilldownQuery) (geoDrilldownPage, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return geoDrilldownPage{}, persistence.ErrNotImplemented
	}
	sqlByType := map[string]string{
		"user":   queryAdminStatsDrilldownUsers,
		"group":  queryAdminStatsDrilldownGroups,
		"moment": queryAdminStatsDrilldownMoments,
	}
	sql, ok := sqlByType[query.EntityType]
	if !ok {
		return geoDrilldownPage{}, persistence.ErrNotImplemented
	}
	var cursorTime *time.Time
	var cursorID *string
	if query.Cursor != "" {
		c, apiErr := decodeGeoDrilldownCursor(query.Cursor)
		if apiErr != nil {
			return geoDrilldownPage{}, apiErr // → service asAPIError 映射 10000
		}
		t := c.CreatedAt.UTC()
		cursorTime = &t
		id := c.ObjectID
		cursorID = &id
	}
	start, end := nullableTimeRange(query.StartTime, query.EndTime)
	// limit+1 探测行（审查 P2-11，对齐 users/reports 模式）：此前 SQL 取
	// 恰好 limit 行且「满页即发 next_cursor」——桶内条数恰为 limit 时会发出
	// 幽灵 next_cursor，客户端翻下一页拿到空页（违反「无更多数据
	// next_cursor=null」契约）。
	rows, err := queryable.Query(ctx, sql, query.Bucket, geo.CellLevelShift(query.Level), start, end, cursorTime, cursorID, query.Limit+1)
	if err != nil {
		return geoDrilldownPage{}, err
	}
	defer rows.Close()
	page := geoDrilldownPage{Items: []AdminGeoDrilldownItem{}}
	for rows.Next() {
		var id, title string
		var locName *string
		var lat, lng float64
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &locName, &lat, &lng, &createdAt); err != nil {
			return geoDrilldownPage{}, err
		}
		page.Items = append(page.Items, AdminGeoDrilldownItem{
			ObjectID:     id,
			Title:        title,
			LocationName: locName,
			Latitude:     lat,
			Longitude:    lng,
			CreatedAt:    createdAt.UTC().Format(time.RFC3339),
		})
	}
	if err := rows.Err(); err != nil {
		return geoDrilldownPage{}, err
	}
	if len(page.Items) > query.Limit && query.Limit > 0 {
		page.Items = page.Items[:query.Limit]
		last := page.Items[query.Limit-1]
		t, err := time.Parse(time.RFC3339, last.CreatedAt)
		if err != nil {
			return geoDrilldownPage{}, err
		}
		page.Next = &geoDrilldownCursor{CreatedAt: t, ObjectID: last.ObjectID}
	}
	return page, nil
}

// nullableTimeRange 零值时间 → SQL 可空参数（与 map 模块时间窗同款约定）。
func nullableTimeRange(start, end time.Time) (*time.Time, *time.Time) {
	var s, e *time.Time
	if !start.IsZero() {
		t := start.UTC()
		s = &t
	}
	if !end.IsZero() {
		t := end.UTC()
		e = &t
	}
	return s, e
}

// fillTrendDays 把「日期 → 计数」映射按 start 起逐日展开为固定长度序列（空日补零）。
// PG 实现现已由 SQL 直接产出完整日期序列；内存实现（repo_memory_stats.go）仍复用它，
// 两者在正常时钟下口径一致（日期按 UTC+8 自然日）。
func fillTrendDays(rows map[string]geoBucketCount, start time.Time, days int) []AdminGeoStatsTrendPoint {
	sh := time.FixedZone("UTC+8", 8*60*60)
	out := make([]AdminGeoStatsTrendPoint, 0, days)
	dayStart := time.Date(start.In(sh).Year(), start.In(sh).Month(), start.In(sh).Day(), 0, 0, 0, 0, sh)
	for i := 0; i < days; i++ {
		key := dayStart.AddDate(0, 0, i).Format("2006-01-02")
		b := rows[key]
		out = append(out, AdminGeoStatsTrendPoint{Date: key, User: b.User, Group: b.Group, Moment: b.Moment})
	}
	return out
}
