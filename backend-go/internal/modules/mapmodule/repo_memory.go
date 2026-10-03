package mapmodule

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/types"
)

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) Heatmap(_ context.Context, query canvasQuery) (MapHeatCanvas, error) {
	covering := geo.CoveringTokens(query.Bounds, query.Level, 32)
	items := make([]MapHeatBucket, 0, len(covering))
	for i, token := range covering {
		if i >= 5 {
			break
		}
		items = append(items, MapHeatBucket{
			CellToken: token,
			Level:     query.Level,
			// 与 postgres 实现同口径：代表坐标取 S2 格心（spec api_spec_public.md
			// 「MapHeatBucket.location」），保证双实现可互换。
			Location:   fallbackHeatBucketCenter(token, GeoPoint{Latitude: query.Center.Latitude, Longitude: query.Center.Longitude}),
			TotalCount: 3,
			Breakdown:  &MapHeatBreakdown{User: intPtr(1), Group: intPtr(1), Moment: intPtr(1)},
		})
	}
	return MapHeatCanvas{Items: items}, nil
}

func (r *InMemoryRepository) Hotspots(_ context.Context, query canvasQuery) (MapHotspotsCanvas, error) {
	point := query.Center
	createdAt := time.Now().UTC().Format(time.RFC3339)
	distance := float64(geo.DistanceMeters(point, point))
	return MapHotspotsCanvas{Items: []MapHotspotItem{{
		ObjectType: "moment",
		ObjectID:   fmt.Sprintf("moment_map_%d", query.Level),
		Title:      "Map hotspot",
		Location: types.RelativeLocation{
			Point:          types.GeoPoint{Latitude: point.Latitude, Longitude: point.Longitude},
			LocationName:   "",
			Distance:       &distance,
			BearingDegrees: intPtrToFloat64(0),
		},
		CreatedAt: &createdAt,
	}}}, nil
}

func intPtr(v int) *int { return &v }

func intPtrToFloat64(v int) *float64 {
	f := float64(v)
	return &f
}
