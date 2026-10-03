package admin

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/geo"
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
	if req.TrendDays != nil && *req.TrendDays != 0 {
		if *req.TrendDays < statsTrendMinDays || *req.TrendDays > statsTrendMaxDays {
			return geoStatsQuery{}, httpx.InvalidRequest("参数错误")
		}
		query.TrendDays = *req.TrendDays
		sh := time.FixedZone("UTC+8", 8*60*60)
		now := time.Now().In(sh)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, sh)
		query.TrendStart = today.AddDate(0, 0, -(*req.TrendDays - 1))
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

// StatsGeoDrilldown 区域下钻（POST /admin/api/v1/stats/geo/drilldown）：
// cell_token 定位单个网格桶，返回其中指定类型实体明细（keyset 分页）。
func (s *Service) StatsGeoDrilldown(ctx context.Context, req AdminGeoDrilldownRequest) (AdminGeoDrilldownPage, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionStatsRead); apiErr != nil || err != nil {
		return AdminGeoDrilldownPage{}, apiErr, err
	}
	query, apiErr := newGeoDrilldownQuery(req)
	if apiErr != nil {
		return AdminGeoDrilldownPage{}, apiErr, nil
	}
	page, err := s.repo.StatsGeoDrilldown(ctx, query)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return AdminGeoDrilldownPage{}, apiErr, nil
		}
		return AdminGeoDrilldownPage{}, nil, err
	}
	resp := AdminGeoDrilldownPage{Items: page.Items}
	if page.Next != nil {
		c := encodeGeoDrilldownCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

// newGeoDrilldownQuery 下钻参数规整：cell_token 必须为合法 S2 token 且与
// level 自洽（服务端以 token 实际层级为准）；entity_type 白名单三选一。
func newGeoDrilldownQuery(req AdminGeoDrilldownRequest) (geoDrilldownQuery, *httpx.APIError) {
	query := geoDrilldownQuery{Limit: statsDefaultLimit}
	tokenLevel := geo.CellTokenLevel(strings.TrimSpace(req.CellToken))
	if tokenLevel < 0 {
		return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
	}
	if req.Level != nil && *req.Level != tokenLevel {
		return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
	}
	if tokenLevel < statsMinLevel || tokenLevel > statsMaxLevel {
		return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
	}
	query.Level = tokenLevel
	query.Bucket = geo.CellTokenLeafID(strings.TrimSpace(req.CellToken)) >> geo.CellLevelShift(tokenLevel)
	switch req.EntityType {
	case "user", "group", "moment":
		query.EntityType = req.EntityType
	default:
		return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
	}
	if req.TimeRange != nil {
		if req.TimeRange.StartTime != "" {
			parsed, err := time.Parse(time.RFC3339Nano, req.TimeRange.StartTime)
			if err != nil {
				return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
			}
			query.StartTime = parsed.UTC()
		}
		if req.TimeRange.EndTime != "" {
			parsed, err := time.Parse(time.RFC3339Nano, req.TimeRange.EndTime)
			if err != nil {
				return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
			}
			query.EndTime = parsed.UTC()
		}
		if !query.StartTime.IsZero() && !query.EndTime.IsZero() && query.StartTime.After(query.EndTime) {
			return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
		}
	}
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return geoDrilldownQuery{}, httpx.InvalidRequest("参数错误")
		}
		if req.Pagination.Limit == 0 {
			req.Pagination.Limit = statsDefaultLimit
		}
		if req.Pagination.Limit > statsMaxLimit {
			req.Pagination.Limit = statsMaxLimit
		}
		query.Limit = req.Pagination.Limit
		query.Cursor = req.Pagination.Cursor
	}
	return query, nil
}
