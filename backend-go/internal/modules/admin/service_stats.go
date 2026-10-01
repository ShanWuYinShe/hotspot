package admin

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

// StatsGeo 统计分析（POST /admin/api/v1/stats/geo，spec api_spec_admin.md
// 「统计分析（stats）API」）。读路径：权限校验 → 参数规整 → 仓储聚合。
func (s *Service) StatsGeo(ctx context.Context, req AdminGeoStatsRequest) (AdminGeoStatsResponse, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionStatsRead); apiErr != nil || err != nil {
		return AdminGeoStatsResponse{}, apiErr, err
	}
	query, apiErr := newGeoStatsQuery(req)
	if apiErr != nil {
		return AdminGeoStatsResponse{}, apiErr, nil
	}
	resp, err := s.repo.StatsGeo(ctx, query)
	if err != nil {
		return AdminGeoStatsResponse{}, nil, err
	}
	return resp, nil, nil
}

// newGeoStatsQuery 参数规整：level 越界/时间窗倒置 → 10000（参数必显式）；
// limit 按「默认 20、上限 100」由 Service 层兜底裁剪（全局分页纪律）。
func newGeoStatsQuery(req AdminGeoStatsRequest) (geoStatsQuery, *httpx.APIError) {
	query := geoStatsQuery{Level: statsDefaultLevel, Limit: statsDefaultLimit}
	if req.Level != nil {
		if *req.Level < statsMinLevel || *req.Level > statsMaxLevel {
			return geoStatsQuery{}, httpx.InvalidRequest("参数错误")
		}
		query.Level = *req.Level
	}
	if req.Limit != nil {
		if *req.Limit < 0 {
			return geoStatsQuery{}, httpx.InvalidRequest("参数错误")
		}
		if *req.Limit == 0 {
			*req.Limit = statsDefaultLimit
		}
		if *req.Limit > statsMaxLimit {
			*req.Limit = statsMaxLimit
		}
		query.Limit = *req.Limit
	}
	if req.TimeRange != nil {
		if req.TimeRange.StartTime != "" {
			parsed, err := time.Parse(time.RFC3339Nano, req.TimeRange.StartTime)
			if err != nil {
				return geoStatsQuery{}, httpx.InvalidRequest("参数错误")
			}
			query.StartTime = parsed.UTC()
		}
		if req.TimeRange.EndTime != "" {
			parsed, err := time.Parse(time.RFC3339Nano, req.TimeRange.EndTime)
			if err != nil {
				return geoStatsQuery{}, httpx.InvalidRequest("参数错误")
			}
			query.EndTime = parsed.UTC()
		}
		if !query.StartTime.IsZero() && !query.EndTime.IsZero() && query.StartTime.After(query.EndTime) {
			return geoStatsQuery{}, httpx.InvalidRequest("参数错误")
		}
	}
	return query, nil
}
