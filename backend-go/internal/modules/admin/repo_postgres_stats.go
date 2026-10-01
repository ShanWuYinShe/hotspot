package admin

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/geo"
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
		Regions:     []AdminRegionRankItem{},
		Places:      []AdminPlaceRankItem{},
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
