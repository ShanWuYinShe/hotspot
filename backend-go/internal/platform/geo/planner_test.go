package geo

import "testing"

func TestSearchPlanMatch(t *testing.T) {
	plan := NewSearchPlan(Point{Latitude: 39.9, Longitude: 116.4}, 5000, "distance")
	distance, bearing, ok := plan.Match(Point{Latitude: 39.91, Longitude: 116.41})
	if !ok {
		t.Fatal("expected candidate to match")
	}
	if distance <= 0 || bearing < 0 || bearing > 360 {
		t.Fatalf("unexpected distance/bearing: %d %d", distance, bearing)
	}
	if _, _, ok := plan.Match(Point{Latitude: 40.5, Longitude: 117.5}); ok {
		t.Fatal("expected far candidate not to match")
	}
}
