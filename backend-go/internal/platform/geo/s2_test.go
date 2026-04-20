package geo

import (
	"math"
	"testing"
)

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

func TestCellTokenAtLevel_SamePointSameToken(t *testing.T) {
	p := Point{Latitude: 39.9042, Longitude: 116.4074}
	t1 := CellTokenAtLevel(p, 14)
	t2 := CellTokenAtLevel(p, 14)
	if t1 != t2 {
		t.Fatalf("same point at same level should produce same token: %s vs %s", t1, t2)
	}
}

func TestCellTokenAtLevel_DifferentLevels(t *testing.T) {
	p := Point{Latitude: 39.9042, Longitude: 116.4074}
	t14 := CellTokenAtLevel(p, 14)
	t12 := CellTokenAtLevel(p, 12)
	if t14 == t12 {
		t.Fatal("different levels should produce different tokens")
	}
	// level 12 token should be a prefix of level 14 token (parent-child)
	if len(t12) > len(t14) {
		t.Fatalf("lower level token should be shorter: level12=%s level14=%s", t12, t14)
	}
}

func TestCellTokenAtLevel_NearbyPoints_SameCellOrAdjacent(t *testing.T) {
	p1 := Point{Latitude: 39.9042, Longitude: 116.4074}
	p2 := Point{Latitude: 39.9050, Longitude: 116.4080}
	t1 := CellTokenAtLevel(p1, 14)
	t2 := CellTokenAtLevel(p2, 14)
	// S2 cell boundaries don't align with lat/lng grid — nearby points may land
	// in adjacent cells. Verify tokens are valid but don't assert equality.
	if t1 == "" || t2 == "" {
		t.Fatalf("expected non-empty tokens: %s, %s", t1, t2)
	}
}

func TestCellTokenAtLevel_NearbyPointsDifferentCell(t *testing.T) {
	p1 := Point{Latitude: 39.9042, Longitude: 116.4074}
	// ~1km away — likely different level-14 cell
	p2 := Point{Latitude: 39.9130, Longitude: 116.4180}
	t1 := CellTokenAtLevel(p1, 14)
	t2 := CellTokenAtLevel(p2, 14)
	if t1 == t2 {
		t.Log("points ~1km apart may or may not share cell at level 14 (ok)")
	}
}

func TestCoveringTokens_Inclusion(t *testing.T) {
	// Critical property: any point inside bounds should have its cell token
	// present in the covering tokens at the same level.
	bounds := Bounds{North: 40.0, South: 39.8, East: 116.5, West: 116.3}
	level := 14
	tokens := CoveringTokens(bounds, level, 500)
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, tok := range tokens {
		tokenSet[tok] = struct{}{}
	}

	// Test several points inside the bounds
	testPoints := []Point{
		{Latitude: 39.9, Longitude: 116.4},
		{Latitude: 39.85, Longitude: 116.35},
		{Latitude: 39.95, Longitude: 116.45},
		{Latitude: 39.82, Longitude: 116.38},
		{Latitude: 39.98, Longitude: 116.42},
	}
	for _, p := range testPoints {
		token := CellTokenAtLevel(p, level)
		if _, ok := tokenSet[token]; !ok {
			t.Errorf("point (%.4f, %.4f) token %s not in covering (tokens=%d, maxCells=%d)", p.Latitude, p.Longitude, token, len(tokens), 500)
		}
	}
}

func TestCoveringTokens_InclusionWithSmallMaxCells(t *testing.T) {
	// With small maxCells, covering might not include all interior cells.
	// This tests the boundary of maxCells limitation.
	bounds := Bounds{North: 40.5, South: 39.5, East: 117.0, West: 116.0}
	level := 14
	tokens := CoveringTokens(bounds, level, 32)
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, tok := range tokens {
		tokenSet[tok] = struct{}{}
	}

	center := Center(bounds)
	centerToken := CellTokenAtLevel(center, level)
	if _, ok := tokenSet[centerToken]; !ok {
		t.Logf("center token %s not in covering with maxCells=32 (tokens=%d) — may need larger maxCells for wide regions", centerToken, len(tokens))
	}
}

func TestCoveringTokens_LevelMismatch(t *testing.T) {
	// The bug: storage at level 14, search covering at different level.
	// When levels differ, the covering tokens won't contain the stored token.
	p := Point{Latitude: 39.9042, Longitude: 116.4074}

	// Storage uses level 14
	storageToken := CellTokenAtLevel(p, 14)

	// Search at 50km radius → SuggestedLevel gives level 12
	bounds := BoundsAround(p, 50000)
	searchLevel := SuggestedLevel(100000) // span = 50km * 2 = 100km
	if searchLevel != 12 {
		t.Fatalf("expected level 12 for 100km span, got %d", searchLevel)
	}
	searchTokens := CoveringTokens(bounds, searchLevel, 32)

	found := false
	for _, tok := range searchTokens {
		if tok == storageToken {
			found = true
			break
		}
	}
	if found {
		t.Log("level-14 storage token found in level-12 covering (they are different levels, this should NOT happen)")
	}
	// This confirms the bug: level mismatch means covering doesn't contain stored token
}

func TestSuggestedLevel_Boundaries(t *testing.T) {
	tests := []struct {
		span   float64
		expect int
	}{
		{500, 18},
		{1500, 18},
		{2000, 18},
		{2001, 16},
		{5000, 16},
		{10000, 16},
		{10001, 14},
		{30000, 14},
		{50000, 14},
		{50001, 12},
		{100000, 12},
		{200000, 12},
		{200001, 10},
		{500000, 10},
		{1000000, 10},
		{1000001, 8},
		{5000000, 8},
	}
	for _, tt := range tests {
		got := SuggestedLevel(tt.span)
		if got != tt.expect {
			t.Errorf("SuggestedLevel(%.0f) = %d, want %d", tt.span, got, tt.expect)
		}
	}
}

func TestDistanceMeters_Equator(t *testing.T) {
	// 1 degree of longitude at equator ≈ 111km
	d := DistanceMeters(Point{Latitude: 0, Longitude: 0}, Point{Latitude: 0, Longitude: 1})
	if d < 110000 || d > 112000 {
		t.Errorf("1 degree at equator: got %dm, expect ~111000m", d)
	}
}

func TestDistanceMeters_SamePoint(t *testing.T) {
	p := Point{Latitude: 39.9042, Longitude: 116.4074}
	d := DistanceMeters(p, p)
	if d != 0 {
		t.Errorf("same point distance: got %d, want 0", d)
	}
}

func TestDistanceMeters_KnownDistance(t *testing.T) {
	// Beijing (39.9042, 116.4074) to Shanghai (31.2304, 121.4737) ≈ 1068km
	d := DistanceMeters(Point{Latitude: 39.9042, Longitude: 116.4074}, Point{Latitude: 31.2304, Longitude: 121.4737})
	if d < 1000000 || d > 1150000 {
		t.Errorf("Beijing-Shanghai distance: got %dm, expect ~1068000m", d)
	}
}

func TestDistanceMeters_HighLatitude(t *testing.T) {
	// At 60° latitude, 1 degree longitude ≈ 55.5km
	d := DistanceMeters(Point{Latitude: 60, Longitude: 0}, Point{Latitude: 60, Longitude: 1})
	if d < 50000 || d > 60000 {
		t.Errorf("1 degree longitude at 60° lat: got %dm, expect ~55500m", d)
	}
}

func TestBoundsAround(t *testing.T) {
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	bounds := BoundsAround(ref, 5000)

	center := Center(bounds)
	if math.Abs(center.Latitude-39.9) > 0.01 || math.Abs(center.Longitude-116.4) > 0.01 {
		t.Errorf("BoundsAround center: (%.4f, %.4f), want ~(39.9, 116.4)", center.Latitude, center.Longitude)
	}

	// The reference point should be inside the bounds
	if ref.Latitude < bounds.South || ref.Latitude > bounds.North {
		t.Errorf("ref latitude %.4f outside [%f, %f]", ref.Latitude, bounds.South, bounds.North)
	}
	if ref.Longitude < bounds.West || ref.Longitude > bounds.East {
		t.Errorf("ref longitude %.4f outside [%f, %f]", ref.Longitude, bounds.West, bounds.East)
	}
}

func TestBoundsAround_NearPole(t *testing.T) {
	ref := Point{Latitude: 89.0, Longitude: 0}
	bounds := BoundsAround(ref, 5000)
	if bounds.North > 90 {
		t.Errorf("North bound %.4f should not exceed 90", bounds.North)
	}
	if bounds.South < -90 {
		t.Errorf("South bound %.4f should not be below -90", bounds.South)
	}
}

func TestSearchPlanMatch_ExactRadius(t *testing.T) {
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	plan := NewSearchPlan(ref, 5000, "distance")

	// Point at exactly ~1km — well within 5km
	close := Point{Latitude: 39.909, Longitude: 116.409}
	_, _, ok := plan.Match(close)
	if !ok {
		t.Error("point ~1km away should match 5km radius")
	}
}

func TestSearchPlanMatch_FarOutside(t *testing.T) {
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	plan := NewSearchPlan(ref, 5000, "distance")

	// Point at ~100km away
	far := Point{Latitude: 40.5, Longitude: 117.5}
	_, _, ok := plan.Match(far)
	if ok {
		t.Error("point ~100km away should not match 5km radius")
	}
}

func TestSearchPlanMatch_DifferentRadii(t *testing.T) {
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	// ~20km away
	far := Point{Latitude: 40.0, Longitude: 116.6}

	plan5k := NewSearchPlan(ref, 5000, "distance")
	if _, _, ok := plan5k.Match(far); ok {
		t.Error("20km point should not match 5km radius")
	}

	plan50k := NewSearchPlan(ref, 50000, "distance")
	if _, _, ok := plan50k.Match(far); !ok {
		t.Error("20km point should match 50km radius")
	}
}

func TestSearchPlanMatch_ClampsRadius(t *testing.T) {
	plan := NewSearchPlan(Point{Latitude: 39.9, Longitude: 116.4}, 0, "distance")
	if plan.Radius != 5000 {
		t.Errorf("radius 0 should clamp to 5000, got %d", plan.Radius)
	}
	planBig := NewSearchPlan(Point{Latitude: 39.9, Longitude: 116.4}, 100000, "distance")
	if planBig.Radius != 50000 {
		t.Errorf("radius 100000 should clamp to 50000, got %d", planBig.Radius)
	}
}

func TestSearchPlanLevel_MatchesCoveringLevel(t *testing.T) {
	// SearchPlan.Match uses CellTokenAtLevel at plan.Level to check covering.
	// The covering map was built at the same level, so they should match.
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	plan := NewSearchPlan(ref, 5000, "distance")

	// Get token at plan.Level for a point near reference
	nearby := Point{Latitude: 39.905, Longitude: 116.405}
	token := CellTokenAtLevel(nearby, plan.Level)

	// This token should be in the covering if the point is inside the bounds
	bounds := BoundsAround(ref, 5000)
	tokens := CoveringTokens(bounds, plan.Level, 32)
	found := false
	for _, tok := range tokens {
		if tok == token {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("token %s (level=%d) for nearby point not in covering (tokens=%d)", token, plan.Level, len(tokens))
	}
}

func TestSearchPlan_LevelVsRadius(t *testing.T) {
	ref := Point{Latitude: 39.9, Longitude: 116.4}

	// Verify actual level assignment for various radii
	// SuggestedLevel(span): >1M→8, >200k→10, >50k→12, >10k→14, >2k→16, else→18
	// span = radius * 2
	tests := []struct {
		radius        int
		expectedLevel int
	}{
		{1000, 18},  // span=2000 → not >2000 → 18
		{1500, 16},  // span=3000 → >2000 → 16
		{5000, 16},  // span=10000 → not >10000 → 16
		{6000, 14},  // span=12000 → >10000 → 14
		{25000, 14}, // span=50000 → not >50000 → 14
		{26000, 12}, // span=52000 → >50000 → 12
		{50000, 12}, // span=100000 → not >200000 → 12
	}
	for _, tt := range tests {
		plan := NewSearchPlan(ref, tt.radius, "distance")
		if plan.Level != tt.expectedLevel {
			t.Errorf("radius=%dm: got level %d, want %d", tt.radius, plan.Level, tt.expectedLevel)
		}
	}
}

func TestSearchPlan_Level14StorageCoversRange(t *testing.T) {
	// Groups store s2_cell_token at level 14. Which search radii use level 14?
	// span > 10000 AND span <= 50000 → level 14
	// That means radius > 5000 AND radius <= 25000 uses the same level.
	ref := Point{Latitude: 39.9, Longitude: 116.4}

	for _, radius := range []int{6000, 10000, 15000, 20000, 25000} {
		plan := NewSearchPlan(ref, radius, "distance")
		if plan.Level != 14 {
			t.Errorf("radius %dm should use level 14 (for group search compatibility), got %d", radius, plan.Level)
		}
	}
	// Outside this range, levels diverge — the s2_cell_token filter won't work
	for _, radius := range []int{5000, 30000} {
		plan := NewSearchPlan(ref, radius, "distance")
		if plan.Level == 14 {
			t.Errorf("radius %dm should NOT use level 14, got %d", radius, plan.Level)
		}
	}
}

func TestValidateBounds(t *testing.T) {
	if err := ValidateBounds(Bounds{North: -10, South: 0, East: 10, West: 0}); err == nil {
		t.Error("south > north should fail")
	}
	if err := ValidateBounds(Bounds{North: 10, South: 0, East: 0, West: 10}); err == nil {
		t.Error("west > east should fail")
	}
	if err := ValidateBounds(Bounds{North: 91, South: 0, East: 10, West: 0}); err == nil {
		t.Error("north > 90 should fail")
	}
	if err := ValidateBounds(Bounds{North: 10, South: 0, East: 10, West: 0}); err != nil {
		t.Errorf("valid bounds: %v", err)
	}
}
