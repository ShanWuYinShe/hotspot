package mapmodule

import (
	"context"

	"hotspot/backend-go/internal/platform/geo"
)

type canvasQuery struct {
	Bounds     geo.Bounds
	Center     geo.Point
	SpanMeters float64
	Level      int
}

type Repository interface {
	Heatmap(ctx context.Context, query canvasQuery) (MapHeatCanvas, error)
	Hotspots(ctx context.Context, query canvasQuery) (MapHotspotsCanvas, error)
}
