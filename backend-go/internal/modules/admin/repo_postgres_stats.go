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

// statsTrends 查询每日新增分布（窗口 [TrendStart, now)）并补零展开。
func (r *PostgresRepository) statsTrends(ctx context.Context, queryable persistence.Queryable, query geoStatsQuery) ([]AdminGeoStatsTrendPoint, error) {
	end := time.Now().UTC()
	rows, err := queryable.Query(ctx, queryAdminStatsTrends, query.TrendStart, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDay := map[string]geoBucketCount{}
	for rows.Next() {
		var day time.Time
		var userN, groupN, momentN int64
		if err := rows.Scan(&day, &userN, &groupN, &momentN); err != nil {
			return nil, err
		}
		byDay[day.Format("2006-01-02")] = geoBucketCount{User: int(userN), Group: int(groupN), Moment: int(momentN)}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return fillTrendDays(byDay, query.TrendStart, query.TrendDays), nil
}

// StatsGeoDrilldown 区域下钻：单网格桶内指定类型实体明细（keyset 分页）。
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
	rows, err := queryable.Query(ctx, sql, query.Bucket, geo.CellLevelShift(query.Level), start, end, cursorTime, cursorID, query.Limit)
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
	if len(page.Items) == query.Limit && query.Limit > 0 {
		last := page.Items[len(page.Items)-1]
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
