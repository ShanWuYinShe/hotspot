package geo

import "testing"

func TestGeoHelpers(t *testing.T) {
	bounds := Bounds{North: 10, South: 0, East: 10, West: 0}
	if err := ValidateBounds(bounds); err != nil {
		t.Fatalf("validate bounds: %v", err)
	}
	center := Center(bounds)
	if center.Latitude != 5 || center.Longitude != 5 {
		t.Fatalf("unexpected center: %#v", center)
	}
	if level := SuggestedLevel(100000); level == 0 {
		t.Fatal("expected non-zero level")
	}
	if token := CellTokenAtLevel(Point{Latitude: 1, Longitude: 2}, 12); token == "" {
		t.Fatal("expected token")
	}
	if tokens := CoveringTokens(bounds, 8, 32); len(tokens) == 0 {
		t.Fatal("expected covering tokens")
	}
	if d := DistanceMeters(Point{Latitude: 0, Longitude: 0}, Point{Latitude: 0, Longitude: 1}); d <= 0 {
		t.Fatal("expected distance")
	}
}
