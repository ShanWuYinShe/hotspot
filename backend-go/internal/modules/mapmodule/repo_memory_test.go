package mapmodule

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/geo"
)

func newTestCanvasQuery(level int) canvasQuery {
	bounds := geo.Bounds{North: 37.8, South: 37.7, East: -122.3, West: -122.4}
	return canvasQuery{
		Bounds:     bounds,
		Center:     geo.Center(bounds),
		SpanMeters: geo.ApproxViewportSpanMeters(bounds),
		Level:      level,
		DataType:   "all",
		Limit:      20,
	}
}

func TestInMemoryRepositoryHeatmapReturnsBuckets(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Heatmap(context.Background(), newTestCanvasQuery(8))
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if len(result.Items) == 0 {
		t.Fatal("expected non-empty heatmap items")
	}
	if result.Items[0].Level != 8 {
		t.Fatalf("expected Level=8, got %d", result.Items[0].Level)
	}
	if result.Items[0].CellToken == "" {
		t.Fatal("expected non-empty CellToken")
	}
}

func TestInMemoryRepositoryHeatmapAtHighLevel(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Heatmap(context.Background(), newTestCanvasQuery(15))
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if len(result.Items) == 0 {
		t.Fatal("expected non-empty heatmap at high S2 level")
	}
}

func TestInMemoryRepositoryHeatmapHasBreakdown(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Heatmap(context.Background(), newTestCanvasQuery(8))
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if result.Items[0].Breakdown == nil {
		t.Fatal("expected non-nil Breakdown")
	}
	if *result.Items[0].Breakdown.User == 0 && *result.Items[0].Breakdown.Group == 0 && *result.Items[0].Breakdown.Moment == 0 {
		t.Fatal("expected Breakdown with at least one count > 0")
	}
}

func TestInMemoryRepositoryHeatmapHasLocation(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Heatmap(context.Background(), newTestCanvasQuery(8))
	if err != nil {
		t.Fatalf("Heatmap returned error: %v", err)
	}
	if result.Items[0].Location.Latitude == 0 && result.Items[0].Location.Longitude == 0 {
		t.Fatal("expected heatmap location with non-zero coordinates")
	}
}

func TestInMemoryRepositoryHotspotsReturnsHotspot(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Hotspots(context.Background(), newTestCanvasQuery(8))
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if len(result.Items) == 0 {
		t.Fatal("expected non-empty hotspots")
	}
	if result.Items[0].ObjectType != "moment" {
		t.Fatalf("expected ObjectType 'moment', got %q", result.Items[0].ObjectType)
	}
	if result.Items[0].ObjectID == "" {
		t.Fatal("expected non-empty ObjectID")
	}
	if result.Items[0].Title == "" {
		t.Fatal("expected non-empty Title")
	}
}

func TestInMemoryRepositoryHotspotsHasLocation(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Hotspots(context.Background(), newTestCanvasQuery(8))
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if result.Items[0].Location.Point.Latitude == 0 {
		t.Fatal("expected Location.Latitude to be set from query.Center")
	}
}

func TestInMemoryRepositoryHotspotsHasDistanceMeters(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Hotspots(context.Background(), newTestCanvasQuery(8))
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if result.Items[0].Location.Distance == nil || *result.Items[0].Location.Distance != 0 {
		t.Fatalf("expected Distance=0 (point to itself), got %v", result.Items[0].Location.Distance)
	}
}

func TestInMemoryRepositoryHotspotsHasCreatedAt(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Hotspots(context.Background(), newTestCanvasQuery(8))
	if err != nil {
		t.Fatalf("Hotspots returned error: %v", err)
	}
	if *result.Items[0].CreatedAt == "" {
		t.Fatal("expected non-empty CreatedAt")
	}
}

func TestInMemoryRepositoryHeatmapAndHotspotsIndependent(t *testing.T) {
	repo := NewInMemoryRepository()

	heatResult, heatErr := repo.Heatmap(context.Background(), newTestCanvasQuery(8))
	spotResult, spotErr := repo.Hotspots(context.Background(), newTestCanvasQuery(10))

	if heatErr != nil || spotErr != nil {
		t.Fatalf("Heatmap err=%v Hotspots err=%v", heatErr, spotErr)
	}
	if len(heatResult.Items) == 0 || len(spotResult.Items) == 0 {
		t.Fatal("both should return data independently")
	}
	if heatResult.Items[0].Level != 8 || spotResult.Items[0].ObjectID == "" {
		t.Fatal("data should be independent")
	}
}
