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

func TestMergeHeatBucketsProximityMergesNeighboringCells(t *testing.T) {
	// 两个分属不同 S2 祖先格、但质心相距远小于阈值的相邻小簇：应合并为一个桶，
	// 计数与拆分相加，代表坐标为计数加权重心。
	near := MapHeatBucket{
		CellToken: "b", Level: 8, TotalCount: 1,
		Location:  GeoPoint{Latitude: 35.000, Longitude: 112.000},
		Breakdown: &MapHeatBreakdown{User: ptrInt(1)},
	}
	// ~0.009° 纬度 ≈ 1km，远小于 10km 阈值
	far := MapHeatBucket{
		CellToken: "a", Level: 8, TotalCount: 3,
		Location:  GeoPoint{Latitude: 35.009, Longitude: 112.000},
		Breakdown: &MapHeatBreakdown{Moment: ptrInt(3)},
	}
	out := mergeHeatBucketsProximity([]MapHeatBucket{near, far}, 10000)
	if len(out) != 1 {
		t.Fatalf("expected 1 merged bucket, got %d: %#v", len(out), out)
	}
	b := out[0]
	if b.TotalCount != 4 {
		t.Fatalf("expected total 4, got %d", b.TotalCount)
	}
	if b.Breakdown == nil || b.Breakdown.User == nil || b.Breakdown.Moment == nil || *b.Breakdown.User != 1 || *b.Breakdown.Moment != 3 {
		t.Fatalf("unexpected breakdown: %#v", b.Breakdown)
	}
	// 种子是计数更高者（3 > 1），合并桶继承其 token
	if b.CellToken != "a" {
		t.Fatalf("expected seed token a, got %q", b.CellToken)
	}
	// 加权重心偏向 far（权重 3/4）
	if !(b.Location.Latitude > 35.006 && b.Location.Latitude < 35.008) {
		t.Fatalf("unexpected centroid latitude: %v", b.Location.Latitude)
	}
}

func TestMergeHeatBucketsProximityKeepsDistantBuckets(t *testing.T) {
	a := MapHeatBucket{CellToken: "a", Level: 8, TotalCount: 1, Location: GeoPoint{Latitude: 35.0, Longitude: 112.0}}
	b := MapHeatBucket{CellToken: "b", Level: 8, TotalCount: 1, Location: GeoPoint{Latitude: 35.2, Longitude: 112.0}}
	out := mergeHeatBucketsProximity([]MapHeatBucket{a, b}, 10000)
	if len(out) != 2 {
		t.Fatalf("expected 2 buckets beyond threshold, got %d", len(out))
	}
}

func TestMergeHeatBucketsProximityNilBreakdownAndShortCircuit(t *testing.T) {
	// 单桶直通；nil breakdown 与非 nil breakdown 合并不出错
	a := MapHeatBucket{CellToken: "a", Level: 8, TotalCount: 2, Location: GeoPoint{Latitude: 35.0, Longitude: 112.0}}
	b := MapHeatBucket{CellToken: "b", Level: 8, TotalCount: 1, Location: GeoPoint{Latitude: 35.001, Longitude: 112.0}, Breakdown: &MapHeatBreakdown{Group: ptrInt(1)}}
	if got := mergeHeatBucketsProximity([]MapHeatBucket{a}, 10000); len(got) != 1 || got[0].CellToken != "a" {
		t.Fatalf("single bucket short circuit broken: %#v", got)
	}
	out := mergeHeatBucketsProximity([]MapHeatBucket{a, b}, 10000)
	if len(out) != 1 || out[0].Breakdown == nil || out[0].Breakdown.Group == nil || *out[0].Breakdown.Group != 1 {
		t.Fatalf("nil-breakdown merge broken: %#v", out)
	}
}

func ptrInt(v int) *int { return &v }

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
