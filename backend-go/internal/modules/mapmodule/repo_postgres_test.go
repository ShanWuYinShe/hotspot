package mapmodule

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"math"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

type mapBackendStub struct {
	queryRowsByQuery map[string][][]any
	queryErrs        map[string]error
}

func (b *mapBackendStub) Ping(context.Context) error { return nil }
func (b *mapBackendStub) Close() error               { return nil }
func (b *mapBackendStub) Exec(context.Context, string, ...any) error {
	return nil
}
func (b *mapBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *mapBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	rows, ok := b.queryRowsByQuery[query]
	if !ok {
		return &persistence.InMemoryRows{}, nil
	}
	// 真实过滤语义在 SQL（$4=startTime / $5=endTime）；stub 按传入参数等价过滤，
	// 验证 repo 确实把时间窗下推到了查询层。
	if query == queryMapCanvasMoments && len(args) >= 5 {
		start, _ := args[3].(*time.Time)
		end, _ := args[4].(*time.Time)
		filtered := make([][]any, 0, len(rows))
		for _, row := range rows {
			createdAt, _ := row[5].(time.Time)
			if start != nil && createdAt.Before(*start) {
				continue
			}
			if end != nil && createdAt.After(*end) {
				continue
			}
			filtered = append(filtered, row)
		}
		rows = filtered
	}
	return &testdb.RowsValues{Rows: rows}, nil
}

func newMapFactory(backend persistence.PostgresQueryBackend) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabase(t *testing.T) {
	repo := NewPostgresRepository(nil)

	resp, err := repo.Hotspots(context.Background(), canvasQuery{Limit: 10})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response, got %#v", resp)
	}
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabaseOrFallback(t *testing.T) {
	repo := NewPostgresRepository(nil)

	_, err := repo.Hotspots(context.Background(), canvasQuery{Limit: 10})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryAppliesDataTypeFilter(t *testing.T) {
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasGroups: {{"group_1", "g1", "", "loc", 35.0, 112.0}},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))

	resp, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 10, DataType: "group"})
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].ObjectType != "group" {
		t.Fatalf("expected only group result, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryTimeRangeOnlyAffectsMoments(t *testing.T) {
	old := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasUsers:   {{"user_1", "u1", "", "loc", 35.0, 112.0, old}},
		queryMapCanvasMoments: {{"moment_old", "old moment", "loc", 35.0, 112.0, old}, {"moment_new", "new moment", "loc", 35.0, 112.0, newer}},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))

	resp, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 10, DataType: "all", StartTime: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected user plus filtered moment, got %#v", resp.Items)
	}
	if resp.Items[0].ObjectType != "user" && resp.Items[1].ObjectType != "user" {
		t.Fatalf("expected user to remain visible, got %#v", resp.Items)
	}
	for _, item := range resp.Items {
		if item.ObjectID == "moment_old" {
			t.Fatalf("expected old moment filtered out, got %#v", resp.Items)
		}
	}
}

func TestPostgresRepositoryEndTimeFiltersMomentsOnly(t *testing.T) {
	old := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasUsers:   {{"user_1", "u1", "", "loc", 35.0, 112.0, newer}},
		queryMapCanvasMoments: {{"moment_old", "old moment", "loc", 35.0, 112.0, old}, {"moment_new", "new moment", "loc", 35.0, 112.0, newer}},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))

	resp, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 10, DataType: "all", EndTime: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected user plus filtered moment, got %#v", resp.Items)
	}
	for _, item := range resp.Items {
		if item.ObjectID == "moment_new" {
			t.Fatalf("expected new moment filtered out, got %#v", resp.Items)
		}
	}
}

func TestPostgresRepositoryHeatmapAppliesDataTypeFilter(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasUsers: {
			{"user_1", "u1", "", "same", 35.0, 112.0, base},
			{"user_2", "u2", "", "same", 35.0, 112.0, base},
		},
		queryMapCanvasGroups: {
			{"group_1", "g1", "", "same", 35.0, 112.0},
		},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))

	resp, err := repo.Heatmap(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Level: 12, DataType: "user"})
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].TotalCount != 2 {
		t.Fatalf("unexpected heatmap: %#v", resp.Items)
	}
	if resp.Items[0].Breakdown == nil || *resp.Items[0].Breakdown.User != 2 {
		t.Fatalf("unexpected breakdown: %#v", resp.Items[0])
	}
}

func TestPostgresRepositoryReturnsEmptyWhenCursorPastEnd(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	items := []mapCandidate{{ObjectType: "user", ObjectID: "user_1", CreatedAt: base.Format(time.RFC3339), Distance: 1}}
	cursor := encodeMapCursor(items[0])
	resp, err := paginateHotspots(items, 10, cursor)
	if err != nil {
		t.Fatalf("paginateHotspots returned error: %v", err)
	}
	if len(resp.Items) != 0 || resp.NextCursor != nil {
		t.Fatalf("expected empty page after cursor, got %#v", resp)
	}
}

func TestPostgresRepositoryAggregatesHeatmapBreakdownAndSortsByTotalCount(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasUsers: {
			{"user_1", "u1", "", "same", 35.0, 112.0, base},
			{"user_2", "u2", "", "same", 35.0, 112.0, base},
		},
		queryMapCanvasGroups: {
			{"group_1", "g1", "", "same", 35.0, 112.0},
		},
		queryMapCanvasMoments: {
			{"moment_1", "m1", "same", 35.0, 112.0, base},
			{"moment_2", "m2", "far", 36.0, 113.0, base},
		},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))

	resp, err := repo.Heatmap(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Level: 12, DataType: "all"})
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if len(resp.Items) < 2 {
		t.Fatalf("expected multiple buckets, got %#v", resp.Items)
	}
	if resp.Items[0].TotalCount != 4 || *resp.Items[0].Breakdown.User != 2 || *resp.Items[0].Breakdown.Group != 1 || *resp.Items[0].Breakdown.Moment != 1 {
		t.Fatalf("unexpected top bucket: %#v", resp.Items[0])
	}
	if resp.Items[0].TotalCount < resp.Items[1].TotalCount {
		t.Fatalf("expected buckets sorted by count desc, got %#v", resp.Items)
	}
}

// 回归：桶的代表坐标必须是成员球面质心，而不是 S2 格心或首条候选坐标。
// spec api_spec_public.md「MapHeatBucket.location」＝聚合单元的代表坐标；
// 同一批成员在不同缩放下被分到不同层级网格时，质心能让标记稳定在数据中心
// 附近，而格心必然随网格几何跳动。
func TestPostgresRepositoryHeatBucketLocationUsesMemberCentroid(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	center := geo.CellCenter(geo.CellTokenAtLevel(geo.Point{Latitude: 35.0, Longitude: 112.0}, 12))
	first := geo.Point{Latitude: center.Latitude, Longitude: center.Longitude}
	second := geo.Point{Latitude: center.Latitude + 0.00001, Longitude: center.Longitude + 0.00001}
	if got := geo.CellTokenAtLevel(second, 12); got != geo.CellTokenAtLevel(first, 12) {
		t.Fatalf("test setup: nearby points unexpectedly span two level-12 cells: %q vs %q", geo.CellTokenAtLevel(first, 12), got)
	}
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasUsers: {
			{"user_1", "u1", "", "same", first.Latitude, first.Longitude, base},
		},
		queryMapCanvasMoments: {
			{"moment_1", "m1", "same", second.Latitude, second.Longitude, base},
		},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))

	resp, err := repo.Heatmap(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Level: 12, DataType: "all"})
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected one bucket for nearby members, got %#v", resp.Items)
	}
	bucket := resp.Items[0]
	wantLatitude := (first.Latitude + second.Latitude) / 2
	wantLongitude := (first.Longitude + second.Longitude) / 2
	if math.Abs(bucket.Location.Latitude-wantLatitude) > 1e-9 || math.Abs(bucket.Location.Longitude-wantLongitude) > 1e-9 {
		t.Fatalf("expected bucket location near member centroid (%v, %v), got %v", wantLatitude, wantLongitude, bucket.Location)
	}
	if (bucket.Location.Latitude == first.Latitude && bucket.Location.Longitude == first.Longitude) ||
		(bucket.Location.Latitude == second.Latitude && bucket.Location.Longitude == second.Longitude) {
		t.Fatalf("bucket location should not snap to either member coordinate: %v", bucket.Location)
	}

	// 输入顺序不得改变同一成员集合的质心：排除首条/末条候选决定落位的旧行为。
	reversed := aggregateHeatBuckets([]mapCandidate{
		{ObjectType: "moment", ObjectID: "m1", Location: GeoPoint{Latitude: second.Latitude, Longitude: second.Longitude}},
		{ObjectType: "user", ObjectID: "u1", Location: GeoPoint{Latitude: first.Latitude, Longitude: first.Longitude}},
	}, 12)
	if len(reversed) != 1 || reversed[0].Location != bucket.Location {
		t.Fatalf("expected order-independent centroid %v, got %#v", bucket.Location, reversed)
	}
}

func TestPostgresRepositorySortsHotspotsByDistanceThenTieBreakers(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasUsers: {
			{"user_far", "far", "", "", 35.01, 112.0, base},
			{"user_near", "near", "", "", 35.0001, 112.0, base},
		},
		queryMapCanvasMoments: {
			{"moment_new", "moment", "", 35.0001, 112.0, base.Add(time.Hour)},
			{"moment_old", "moment", "", 35.0001, 112.0, base},
		},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))

	resp, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 10, DataType: "all"})
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if len(resp.Items) != 4 {
		t.Fatalf("expected 4 items, got %#v", resp.Items)
	}
	if resp.Items[len(resp.Items)-1].ObjectID != "user_far" {
		t.Fatalf("expected farthest item last, got %#v", resp.Items)
	}
	if indexOfMapItem(resp.Items, "moment_new") > indexOfMapItem(resp.Items, "moment_old") {
		t.Fatalf("expected newer moment before older moment on ties, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryPaginatesHotspotsWithOpaqueCursor(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &mapBackendStub{queryRowsByQuery: map[string][][]any{
		queryMapCanvasUsers: {
			{"user_1", "u1", "", "", 35.0001, 112.0, base},
			{"user_2", "u2", "", "", 35.0002, 112.0, base},
			{"user_3", "u3", "", "", 35.0003, 112.0, base},
		},
	}}
	repo := NewPostgresRepository(newMapFactory(backend))
	query := canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 2, DataType: "user"}

	first, err := repo.Hotspots(context.Background(), query)
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if len(first.Items) != 2 || first.NextCursor == nil {
		t.Fatalf("expected first page with next cursor, got %#v", first)
	}
	second, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 2, DataType: "user", Cursor: *first.NextCursor})
	if err != nil {
		t.Fatalf("Hotspots second page returned error: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ObjectID != "user_3" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestPostgresRepositoryRejectsInvalidCursor(t *testing.T) {
	_, err := paginateHotspots([]mapCandidate{{ObjectType: "user", ObjectID: "u1", Distance: 1}}, 1, "not-a-cursor")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestPostgresRepositoryReturnsEmptyForNoMatches(t *testing.T) {
	repo := NewPostgresRepository(newMapFactory(&mapBackendStub{}))

	heatmap, err := repo.Heatmap(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Level: 12, DataType: "all"})
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if len(heatmap.Items) != 0 {
		t.Fatalf("expected empty heatmap, got %#v", heatmap)
	}
	hotspots, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 10, DataType: "all"})
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if len(hotspots.Items) != 0 {
		t.Fatalf("expected empty hotspots, got %#v", hotspots)
	}
}

func TestPostgresRepositoryPropagatesQueryError(t *testing.T) {
	repo := NewPostgresRepository(newMapFactory(&mapBackendStub{queryErrs: map[string]error{queryMapCanvasUsers: errors.New("boom")}}))

	_, err := repo.Hotspots(context.Background(), canvasQuery{Bounds: testMapBounds(), Center: testMapCenter(), Limit: 10, DataType: "user"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected query error, got %v", err)
	}
}

func testMapBounds() geo.Bounds {
	return geo.Bounds{North: 36, South: 34, East: 113, West: 111}
}

func testMapCenter() geo.Point {
	return geo.Point{Latitude: 35, Longitude: 112}
}

func indexOfMapItem(items []MapHotspotItem, id string) int {
	for idx, item := range items {
		if item.ObjectID == id {
			return idx
		}
	}
	return -1
}
