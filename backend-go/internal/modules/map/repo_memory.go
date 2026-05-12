package mapmodule

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) Heatmap(_ context.Context, query canvasQuery) (MapHeatCanvas, error) {
	// Generate a few cell tokens distributed across bounds for stub data
	nw := geo.Point{Latitude: query.Bounds.North, Longitude: query.Bounds.West}
	ne := geo.Point{Latitude: query.Bounds.North, Longitude: query.Bounds.East}
	sw := geo.Point{Latitude: query.Bounds.South, Longitude: query.Bounds.West}
	se := geo.Point{Latitude: query.Bounds.South, Longitude: query.Bounds.East}
	tokens := []string{
		geo.CellTokenAtLevel(query.Center, query.Level),
		geo.CellTokenAtLevel(nw, query.Level),
		geo.CellTokenAtLevel(ne, query.Level),
		geo.CellTokenAtLevel(sw, query.Level),
		geo.CellTokenAtLevel(se, query.Level),
	}
	items := make([]MapHeatBucket, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, MapHeatBucket{
			CellToken:  token,
			Level:      query.Level,
			Location:   GeoPoint{Latitude: query.Center.Latitude, Longitude: query.Center.Longitude},
			TotalCount: 3,
			Breakdown:  &MapHeatBreakdown{User: 1, Group: 1, Moment: 1},
		})
	}
	return MapHeatCanvas{Items: items}, nil
}

func (r *InMemoryRepository) Hotspots(_ context.Context, query canvasQuery) (MapHotspotsCanvas, error) {
	point := query.Center
	return MapHotspotsCanvas{Items: []MapHotspotItem{{
		ObjectType:     "moment",
		ObjectID:       fmt.Sprintf("moment_map_%d", query.Level),
		Title:          "Map hotspot",
		Location:       GeoPoint{Latitude: point.Latitude, Longitude: point.Longitude},
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		DistanceMeters: geo.DistanceMeters(point, point),
	}}}, nil
}
