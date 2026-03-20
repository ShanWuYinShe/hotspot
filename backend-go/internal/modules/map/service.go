package mapmodule

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Canvas(ctx context.Context, req MapCanvasRequest) MapCanvasResponse {
	_ = ctx
	bounds := geo.Bounds{North: req.Viewport.Bounds.North, South: req.Viewport.Bounds.South, East: req.Viewport.Bounds.East, West: req.Viewport.Bounds.West}
	center := geo.Center(bounds)
	spanMeters := geo.ApproxViewportSpanMeters(bounds)
	level := geo.SuggestedLevel(spanMeters)

	if spanMeters > 50000 {
		covering := geo.CoveringTokens(bounds, level, 32)
		items := make([]MapHeatBucket, 0, len(covering))
		for i, token := range covering {
			if i >= 5 {
				break
			}
			items = append(items, MapHeatBucket{
				CellToken:  token,
				Level:      level,
				Location:   GeoPoint{Latitude: center.Latitude, Longitude: center.Longitude},
				TotalCount: 3,
				Breakdown:  &MapHeatBreakdown{User: 1, Group: 1, Moment: 1},
			})
		}
		return MapCanvasResponse{Heatmap: &MapHeatCanvas{Items: items}}
	}

	point := geo.Point{Latitude: center.Latitude, Longitude: center.Longitude}
	return MapCanvasResponse{Hotspots: &MapHotspotsCanvas{Items: []MapHotspotItem{{
		ObjectType:     "moment",
		ObjectID:       fmt.Sprintf("moment_map_%d", level),
		Title:          "Map hotspot",
		Location:       GeoPoint{Latitude: point.Latitude, Longitude: point.Longitude},
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		DistanceMeters: geo.DistanceMeters(point, point),
	}}}}
}
