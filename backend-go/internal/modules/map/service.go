package mapmodule

import (
	"context"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
)

type Service struct{ repo Repository }

const heatmapViewportThresholdMeters = 50000

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Canvas(ctx context.Context, req MapCanvasRequest) (MapCanvasResponse, error) {
	query := newCanvasQuery(req)
	if query.SpanMeters > heatmapViewportThresholdMeters {
		heatmap, err := s.repo.Heatmap(ctx, query)
		if err != nil {
			observability.L().Error("map.canvas heatmap failed", "error", err)
			return MapCanvasResponse{}, err
		}
		return MapCanvasResponse{Heatmap: &heatmap}, nil
	}

	hotspots, err := s.repo.Hotspots(ctx, query)
	if err != nil {
		observability.L().Error("map.canvas hotspots failed", "error", err)
		return MapCanvasResponse{}, err
	}
	return MapCanvasResponse{Hotspots: &hotspots}, nil
}

func newCanvasQuery(req MapCanvasRequest) canvasQuery {
	bounds := geo.Bounds{North: req.Viewport.Bounds.North, South: req.Viewport.Bounds.South, East: req.Viewport.Bounds.East, West: req.Viewport.Bounds.West}
	spanMeters := geo.ApproxViewportSpanMeters(bounds)
	return canvasQuery{
		Bounds:     bounds,
		Center:     geo.Center(bounds),
		SpanMeters: spanMeters,
		Level:      geo.SuggestedLevel(spanMeters),
	}
}
