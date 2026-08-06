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
		heatmap, err := s.repo.Heatmap(ctx, query)
		if err != nil {
			observability.L().Error("map.canvas heatmap failed", "error", err)
			return MapCanvasResponse{}, err
		}
		if len(heatmap.Items) == 0 {
			return MapCanvasResponse{}, nil
		}
		return MapCanvasResponse{Heatmap: &heatmap}, nil
	}

	hotspots, err := s.repo.Hotspots(ctx, query)
	if err != nil {
		observability.L().Error("map.canvas hotspots failed", "error", err)
		return MapCanvasResponse{}, err
	}
	if len(hotspots.Items) == 0 {
		return MapCanvasResponse{}, nil
	}
	return MapCanvasResponse{Hotspots: &hotspots}, nil
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
				parsed, err := time.Parse(time.RFC3339, req.Filter.TimeRange.StartTime)
				if err != nil {
					return canvasQuery{}, err
				}
				query.StartTime = parsed.UTC()
			}
			if req.Filter.TimeRange.EndTime != "" {
				parsed, err := time.Parse(time.RFC3339, req.Filter.TimeRange.EndTime)
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
