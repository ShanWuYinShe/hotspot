package mapmodule

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
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

func TestServiceCanvasUsesPersistenceRepositoryWhenRepoMissing(t *testing.T) {
	service := NewServiceWithRepo(nil)

	_, err := service.Canvas(context.Background(), MapCanvasRequest{Viewport: MapViewport{Bounds: Bounds{North: 1, South: 0, East: 1, West: 0}}})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
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
	if repo.lastHeatQuery.Level != geo.SuggestedLevel(repo.lastHeatQuery.SpanMeters) {
		t.Fatalf("expected suggested level, got %#v", repo.lastHeatQuery)
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
