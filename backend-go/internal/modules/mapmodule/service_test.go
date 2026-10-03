package mapmodule

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

type stubMapRepository struct {
	heatmapResp   MapHeatCanvas
	hotspotsResp  MapHotspotsCanvas
	heatmapErr    error
	hotspotsErr   error
	lastHeatQuery canvasQuery
	lastHotQuery  canvasQuery
}

func (r *stubMapRepository) Heatmap(_ context.Context, query canvasQuery) (MapHeatCanvas, error) {
	r.lastHeatQuery = query
	return r.heatmapResp, r.heatmapErr
}

func (r *stubMapRepository) Hotspots(_ context.Context, query canvasQuery) (MapHotspotsCanvas, error) {
	r.lastHotQuery = query
	return r.hotspotsResp, r.hotspotsErr
}

func TestServiceCanvasUsesHeatmapForLargeViewport(t *testing.T) {
	repo := &stubMapRepository{heatmapResp: MapHeatCanvas{Items: []MapHeatBucket{{CellToken: "a", Level: 8, TotalCount: 2}}}}
	service := NewServiceWithRepo(repo)

	resp, err := service.Canvas(context.Background(), MapCanvasRequest{Viewport: MapViewport{Bounds: Bounds{North: 10, South: 0, East: 10, West: 0}}})
	if err != nil {
		t.Fatalf("Canvas returned error: %v", err)
	}
	if resp.Heatmap == nil || resp.Hotspots != nil {
		t.Fatalf("expected heatmap-only response, got %#v", resp)
	}
	if repo.lastHeatQuery.Level != heatmapAggregationLevel(repo.lastHeatQuery.SpanMeters) {
		t.Fatalf("expected heatmap aggregation level, got %#v", repo.lastHeatQuery)
	}
	if repo.lastHeatQuery.Level != 5 {
		t.Fatalf("expected continental viewport to use level 5, got %d", repo.lastHeatQuery.Level)
	}

	query, err := newCanvasQuery(MapCanvasRequest{Viewport: MapViewport{Bounds: Bounds{North: 10, South: 0, East: 10, West: 0}}})
	if err != nil {
		t.Fatalf("newCanvasQuery returned error: %v", err)
	}
	if query.Level != geo.SuggestedLevel(query.SpanMeters) {
		t.Fatalf("expected hotspots/search path to retain suggested level, got %#v", query)
	}
}

func TestHeatmapAggregationLevelMergesNearbySmallClusters(t *testing.T) {
	base := geo.Point{Latitude: 35.0, Longitude: 112.0}
	parent := geo.CellTokenAtLevel(base, 6)
	members := make([]mapCandidate, 0, 2)
	objectIDs := []string{"merge-a", "merge-b"}
	seenFineCells := map[string]bool{}
	for _, dLat := range []float64{-0.4, -0.2, 0, 0.2, 0.4} {
		for _, dLng := range []float64{-0.4, -0.2, 0, 0.2, 0.4} {
			point := geo.Point{Latitude: base.Latitude + dLat, Longitude: base.Longitude + dLng}
			if geo.CellTokenAtLevel(point, 6) != parent {
				continue
			}
			fine := geo.CellTokenAtLevel(point, 8)
			if seenFineCells[fine] {
				continue
			}
			seenFineCells[fine] = true
			members = append(members, mapCandidate{
				ObjectType: "user",
				ObjectID:   objectIDs[len(members)],
				Location:   GeoPoint{Latitude: point.Latitude, Longitude: point.Longitude},
			})
			if len(members) == 2 {
				break
			}
		}
		if len(members) == 2 {
			break
		}
	}
	if len(members) != 2 {
		t.Fatal("test setup: expected two distinct level-8 cells inside one level-6 cell")
	}

	buckets := aggregateHeatBuckets(members, heatmapAggregationLevel(600000))
	if len(buckets) != 1 || buckets[0].TotalCount != 2 {
		t.Fatalf("expected one merged level-6 bucket, got %#v", buckets)
	}
}

func TestHeatmapAggregationLevelCoarsensWithViewportSpan(t *testing.T) {
	tests := []struct {
		span float64
		want int
	}{
		{50001, 10},
		{62500, 10},
		{62501, 9},
		{150000, 8},
		{300000, 7},
		{800000, 6},
		{1500000, 5},
		{3000000, 4},
		{9000000, 2},
		{18000000, 1},
	}
	for _, tt := range tests {
		if got := heatmapAggregationLevel(tt.span); got != tt.want {
			t.Errorf("heatmapAggregationLevel(%.0f) = %d, want %d", tt.span, got, tt.want)
		}
	}
}

func TestServiceCanvasUsesHotspotsForSmallViewportAndParsesFilters(t *testing.T) {
	repo := &stubMapRepository{hotspotsResp: MapHotspotsCanvas{Items: []MapHotspotItem{{ObjectType: "user", ObjectID: "u1"}}}}
	service := NewServiceWithRepo(repo)

	resp, err := service.Canvas(context.Background(), MapCanvasRequest{
		Viewport: MapViewport{Bounds: Bounds{North: 0.2, South: 0, East: 0.2, West: 0}},
		Filter: &MapCanvasFilter{
			DataType: "moment",
			TimeRange: &TimeRangeParams{
				StartTime: "2026-03-01T00:00:00Z",
				EndTime:   "2026-03-02T00:00:00Z",
			},
		},
		Pagination: &PaginationParams{Limit: 25, Cursor: "abc"},
	})
	if err != nil {
		t.Fatalf("Canvas returned error: %v", err)
	}
	if resp.Hotspots == nil || resp.Heatmap != nil {
		t.Fatalf("expected hotspots-only response, got %#v", resp)
	}
	if repo.lastHotQuery.DataType != "moment" || repo.lastHotQuery.Limit != 25 || repo.lastHotQuery.Cursor != "abc" {
		t.Fatalf("unexpected query: %#v", repo.lastHotQuery)
	}
	if repo.lastHotQuery.StartTime.Format(time.RFC3339) != "2026-03-01T00:00:00Z" || repo.lastHotQuery.EndTime.Format(time.RFC3339) != "2026-03-02T00:00:00Z" {
		t.Fatalf("unexpected time range: %#v", repo.lastHotQuery)
	}
}

func TestServiceCanvasReturnsEmptyWhenHeatmapItemsEmpty(t *testing.T) {
	service := NewServiceWithRepo(&stubMapRepository{heatmapResp: MapHeatCanvas{}})

	resp, err := service.Canvas(context.Background(), MapCanvasRequest{Viewport: MapViewport{Bounds: Bounds{North: 10, South: 0, East: 10, West: 0}}})
	if err != nil {
		t.Fatalf("Canvas returned error: %v", err)
	}
	if resp.Heatmap != nil || resp.Hotspots != nil {
		t.Fatalf("expected empty response, got %#v", resp)
	}
}

func TestServiceCanvasReturnsEmptyWhenHotspotsItemsEmpty(t *testing.T) {
	service := NewServiceWithRepo(&stubMapRepository{hotspotsResp: MapHotspotsCanvas{}})

	resp, err := service.Canvas(context.Background(), MapCanvasRequest{Viewport: MapViewport{Bounds: Bounds{North: 0.2, South: 0, East: 0.2, West: 0}}})
	if err != nil {
		t.Fatalf("Canvas returned error: %v", err)
	}
	if resp.Heatmap != nil || resp.Hotspots != nil {
		t.Fatalf("expected empty response, got %#v", resp)
	}
}

func TestServiceCanvasReturnsParseErrorForInvalidTimeRange(t *testing.T) {
	service := NewServiceWithRepo(&stubMapRepository{})

	_, err := service.Canvas(context.Background(), MapCanvasRequest{
		Viewport: MapViewport{Bounds: Bounds{North: 0.2, South: 0, East: 0.2, West: 0}},
		Filter:   &MapCanvasFilter{TimeRange: &TimeRangeParams{StartTime: "bad-time"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceCanvasDefaultsHotspotLimitTo50(t *testing.T) {
	repo := &stubMapRepository{hotspotsResp: MapHotspotsCanvas{Items: []MapHotspotItem{{ObjectType: "user", ObjectID: "u1"}}}}
	service := NewServiceWithRepo(repo)

	_, err := service.Canvas(context.Background(), MapCanvasRequest{Viewport: MapViewport{Bounds: Bounds{North: 0.2, South: 0, East: 0.2, West: 0}}})
	if err != nil {
		t.Fatalf("Canvas returned error: %v", err)
	}
	if repo.lastHotQuery.Limit != 50 {
		t.Fatalf("expected default limit 50, got %d", repo.lastHotQuery.Limit)
	}
}

func TestServiceCanvasClampsHotspotLimitTo200(t *testing.T) {
	repo := &stubMapRepository{hotspotsResp: MapHotspotsCanvas{Items: []MapHotspotItem{{ObjectType: "user", ObjectID: "u1"}}}}
	service := NewServiceWithRepo(repo)

	_, err := service.Canvas(context.Background(), MapCanvasRequest{
		Viewport:   MapViewport{Bounds: Bounds{North: 0.2, South: 0, East: 0.2, West: 0}},
		Pagination: &PaginationParams{Limit: 999},
	})
	if err != nil {
		t.Fatalf("Canvas returned error: %v", err)
	}
	if repo.lastHotQuery.Limit != 200 {
		t.Fatalf("expected clamped limit 200, got %d", repo.lastHotQuery.Limit)
	}
}

func TestServiceCanvasDefaultsDataTypeToAll(t *testing.T) {
	repo := &stubMapRepository{hotspotsResp: MapHotspotsCanvas{Items: []MapHotspotItem{{ObjectType: "user", ObjectID: "u1"}}}}
	service := NewServiceWithRepo(repo)

	_, err := service.Canvas(context.Background(), MapCanvasRequest{Viewport: MapViewport{Bounds: Bounds{North: 0.2, South: 0, East: 0.2, West: 0}}})
	if err != nil {
		t.Fatalf("Canvas returned error: %v", err)
	}
	if repo.lastHotQuery.DataType != "all" {
		t.Fatalf("expected default data type all, got %#v", repo.lastHotQuery)
	}
}
