package mapmodule

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
)

type Service struct{ repo Repository }

const heatmapViewportThresholdMeters = 50000

func NewService() *Service { return NewServiceWithRepo(NewPostgresRepository(nil)) }
func NewServiceWithRepo(repo Repository) *Service {
	if repo == nil {
		repo = NewPostgresRepository(nil)
	}
	return &Service{repo: repo}
}

func (s *Service) Canvas(ctx context.Context, req MapCanvasRequest) (MapCanvasResponse, error) {
	query, err := newCanvasQuery(req)
	if err != nil {
		return MapCanvasResponse{}, err
	}
	if query.SpanMeters > heatmapViewportThresholdMeters {
		// 聚合粒度必须随可视跨度变化：query.Level 继续沿用搜索预过滤口径，
		// 而热力桶改用与可视尺度匹配的层级，避免大陆/全球视口仍显示城市尺度小簇。
		query.Level = heatmapAggregationLevel(query.SpanMeters)
		heatmap, err := s.repo.Heatmap(ctx, query)
		if err != nil {
			observability.L().Error("map.canvas heatmap failed", "error", err)
			return MapCanvasResponse{}, err
		}
		if len(heatmap.Items) == 0 {
			return MapCanvasResponse{Mode: MapCanvasModeEmpty}, nil
		}
		return MapCanvasResponse{Mode: MapCanvasModeHeatmap, Heatmap: &heatmap}, nil
	}

	hotspots, err := s.repo.Hotspots(ctx, query)
	if err != nil {
		observability.L().Error("map.canvas hotspots failed", "error", err)
		return MapCanvasResponse{}, err
	}
	if len(hotspots.Items) == 0 {
		return MapCanvasResponse{Mode: MapCanvasModeEmpty}, nil
	}
	return MapCanvasResponse{Mode: MapCanvasModeHotspots, Hotspots: &hotspots}, nil
}

// heatmapAggregationLevel 根据热力视口跨度选择 S2 聚合层级。
// 目标是让热力桶边长约占视口跨度的六分之一：相邻 S2 层级边长相差一倍，
// 因此跨度每翻一番就降低一级。大尺度下相邻小簇会被合并到同一祖先桶；
// 小尺度热力视口（刚超过 50km 阈值）仍使用 level 10，保持街道/城区细节。
func heatmapAggregationLevel(spanMeters float64) int {
	switch {
	case spanMeters > 16000000:
		return 1
	case spanMeters > 8000000:
		return 2
	case spanMeters > 4000000:
		return 3
	case spanMeters > 2000000:
		return 4
	case spanMeters > 1000000:
		return 5
	case spanMeters > 500000:
		return 6
	case spanMeters > 250000:
		return 7
	case spanMeters > 125000:
		return 8
	case spanMeters > 62500:
		return 9
	default:
		return 10
	}
}

func newCanvasQuery(req MapCanvasRequest) (canvasQuery, error) {
	bounds := geo.Bounds{North: req.Viewport.Bounds.North, South: req.Viewport.Bounds.South, East: req.Viewport.Bounds.East, West: req.Viewport.Bounds.West}
	spanMeters := geo.ApproxViewportSpanMeters(bounds)
	query := canvasQuery{
		Bounds:     bounds,
		Center:     geo.Center(bounds),
		SpanMeters: spanMeters,
		Level:      geo.SuggestedLevel(spanMeters),
		DataType:   normalizeMapDataType(""),
		Limit:      normalizeMapLimit(0),
	}
	if req.Filter != nil {
		query.DataType = normalizeMapDataType(req.Filter.DataType)
		if req.Filter.TimeRange != nil {
			if req.Filter.TimeRange.StartTime != "" {
				parsed, err := time.Parse(time.RFC3339Nano, req.Filter.TimeRange.StartTime)
				if err != nil {
					return canvasQuery{}, err
				}
				query.StartTime = parsed.UTC()
			}
			if req.Filter.TimeRange.EndTime != "" {
				parsed, err := time.Parse(time.RFC3339Nano, req.Filter.TimeRange.EndTime)
				if err != nil {
					return canvasQuery{}, err
				}
				query.EndTime = parsed.UTC()
			}
		}
	}
	if req.Pagination != nil {
		query.Limit = normalizeMapLimit(req.Pagination.Limit)
		query.Cursor = req.Pagination.Cursor
	}
	return query, nil
}
