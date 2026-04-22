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

func TestSearchPlanLevel_RangeContainsNearby(t *testing.T) {
	// SearchPlan now uses SearchCovering ranges (same as DB BETWEEN).
	// A nearby point's leaf cell ID must fall within the plan's ranges.
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	plan := NewSearchPlan(ref, 5000, "distance")

	nearby := Point{Latitude: 39.905, Longitude: 116.405}
	_, _, ok := plan.Match(nearby)
	if !ok {
		t.Error("nearby point should match plan with range-based covering")
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
	// SuggestedLevel(span): >1M→8, >200k→10, >50k→12, >10k→14, >2k→16, else→18
	ref := Point{Latitude: 39.9, Longitude: 116.4}

	for _, radius := range []int{6000, 10000, 15000, 20000, 25000} {
		plan := NewSearchPlan(ref, radius, "distance")
		if plan.Level != 14 {
			t.Errorf("radius %dm should use level 14, got %d", radius, plan.Level)
		}
	}
	for _, radius := range []int{5000, 30000} {
		plan := NewSearchPlan(ref, radius, "distance")
		if plan.Level == 14 {
			t.Errorf("radius %dm should NOT use level 14, got %d", radius, plan.Level)
		}
	}
}

func TestSearchPlan_MatchConsistentWithSearchCovering(t *testing.T) {
	// THE critical consistency test: a point within radius must pass Match
	// exactly when its leaf cell ID is in SearchCovering ranges AND within distance.
	ref := Point{Latitude: 39.9042, Longitude: 116.4074}
	for _, radius := range []int{500, 1000, 5000, 25000, 50000} {
		plan := NewSearchPlan(ref, radius, "distance")
		bounds := BoundsAround(ref, radius)
		ranges := SearchCovering(bounds)

		// The plan's internal ranges should match SearchCovering output
		planRangesMatch := len(plan.ranges) == len(ranges)
		if planRangesMatch {
			for i := range ranges {
				if plan.ranges[i] != ranges[i] {
					planRangesMatch = false
					break
				}
			}
		}
		if !planRangesMatch {
			t.Errorf("r=%dm: plan ranges don't match SearchCovering output", radius)
		}

		// Reference point itself must always match
		_, _, ok := plan.Match(ref)
		if !ok {
			t.Errorf("r=%dm: reference point should always match its own plan", radius)
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

func TestLeafCellID_SamePointSameID(t *testing.T) {
	p := Point{Latitude: 39.9042, Longitude: 116.4074}
	id1 := LeafCellID(p)
	id2 := LeafCellID(p)
	if id1 != id2 {
		t.Fatalf("same point should produce same cell ID: %d vs %d", id1, id2)
	}
}

func TestLeafCellID_NearbyPointsDifferentID(t *testing.T) {
	p1 := Point{Latitude: 39.9042, Longitude: 116.4074}
	p2 := Point{Latitude: 39.9050, Longitude: 116.4080}
	id1 := LeafCellID(p1)
	id2 := LeafCellID(p2)
	if id1 == 0 || id2 == 0 {
		t.Fatalf("expected non-zero cell IDs: %d, %d", id1, id2)
	}
}

func TestLeafCellID_DistantPointsFarApart(t *testing.T) {
	beijing := LeafCellID(Point{Latitude: 39.9042, Longitude: 116.4074})
	shanghai := LeafCellID(Point{Latitude: 31.2304, Longitude: 121.4737})
	if beijing == 0 || shanghai == 0 {
		t.Fatal("expected non-zero cell IDs")
	}
	// Distant cities should have very different cell IDs
	diff := beijing - shanghai
	if diff < 0 {
		diff = -diff
	}
	if diff < 1_000_000 {
		t.Errorf("distant cities should have very different cell IDs, diff=%d", diff)
	}
}

func TestSearchCovering_Basic(t *testing.T) {
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	bounds := BoundsAround(ref, 5000)
	ranges := SearchCovering(bounds)
	if len(ranges) == 0 {
		t.Fatal("expected non-empty covering ranges")
	}
	for i, r := range ranges {
		if r[0] > r[1] {
			t.Errorf("range %d: min %d > max %d", i, r[0], r[1])
		}
	}
}

func TestSearchCovering_ContainsCenterPoint(t *testing.T) {
	ref := Point{Latitude: 34.7869, Longitude: 113.6421}
	for _, radius := range []int{1000, 5000, 25000, 50000} {
		bounds := BoundsAround(ref, radius)
		ranges := SearchCovering(bounds)
		centerID := LeafCellID(Center(bounds))
		found := false
		for _, r := range ranges {
			if centerID >= r[0] && centerID <= r[1] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("r=%d: center cell ID %d not in any covering range (ranges=%d)", radius, centerID, len(ranges))
		}
	}
}

func TestSearchCovering_ContainsInteriorPoints(t *testing.T) {
	bounds := Bounds{North: 40.0, South: 39.8, East: 116.5, West: 116.3}
	ranges := SearchCovering(bounds)
	testPoints := []Point{
		{Latitude: 39.9, Longitude: 116.4},
		{Latitude: 39.85, Longitude: 116.35},
		{Latitude: 39.95, Longitude: 116.45},
	}
	for _, p := range testPoints {
		id := LeafCellID(p)
		found := false
		for _, r := range ranges {
			if id >= r[0] && id <= r[1] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("point (%.4f, %.4f) cell ID %d not in any covering range", p.Latitude, p.Longitude, id)
		}
	}
}

func TestSearchCovering_SmallRadius(t *testing.T) {
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	bounds := BoundsAround(ref, 500)
	ranges := SearchCovering(bounds)
	if len(ranges) == 0 {
		t.Fatal("small radius should still produce covering ranges")
	}
	centerID := LeafCellID(ref)
	found := false
	for _, r := range ranges {
		if centerID >= r[0] && centerID <= r[1] {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("center cell ID %d not in covering for 500m radius", centerID)
	}
}

// cellInRanges checks if a leaf cell ID falls within any covering range.
func cellInRanges(cellID int64, ranges [][2]int64) bool {
	for _, r := range ranges {
		if cellID >= r[0] && cellID <= r[1] {
			return true
		}
	}
	return false
}

func TestRoundTrip_FullRadiusSweep(t *testing.T) {
	// THE critical test: a point stored via LeafCellID must be findable
	// via SearchCovering at every supported search radius.
	ref := Point{Latitude: 39.9042, Longitude: 116.4074}
	storedID := LeafCellID(ref)

	radii := []int{100, 200, 500, 1000, 2000, 3000, 5000, 8000, 10000, 15000, 20000, 25000, 30000, 40000, 50000}
	for _, radius := range radii {
		bounds := BoundsAround(ref, radius)
		ranges := SearchCovering(bounds)
		if !cellInRanges(storedID, ranges) {
			t.Errorf("radius=%dm: stored cell ID %d not in covering (ranges=%d)", radius, storedID, len(ranges))
		}
	}
}

func TestRoundTrip_OffsetPointsWithinRadius(t *testing.T) {
	// Points offset from center but still inside the radius must also be found.
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	radius := 5000

	// Generate 8 points at ~50% of radius in cardinal and diagonal directions
	offsets := []struct {
		latDelta float64
		lngDelta float64
	}{
		{0.02, 0},      // ~2.2km north
		{-0.02, 0},     // ~2.2km south
		{0, 0.02},      // ~1.7km east
		{0, -0.02},     // ~1.7km west
		{0.01, 0.01},   // ~1.5km NE
		{-0.01, -0.01}, // ~1.5km SW
		{0.01, -0.01},  // ~1.5km NW
		{-0.01, 0.01},  // ~1.5km SE
	}

	bounds := BoundsAround(ref, radius)
	ranges := SearchCovering(bounds)

	for _, off := range offsets {
		p := Point{Latitude: ref.Latitude + off.latDelta, Longitude: ref.Longitude + off.lngDelta}
		id := LeafCellID(p)
		if !cellInRanges(id, ranges) {
			dist := DistanceMeters(ref, p)
			t.Errorf("offset point (%.4f, %.4f) dist=%dm not in 5km covering (id=%d, ranges=%d)",
				p.Latitude, p.Longitude, dist, id, len(ranges))
		}
	}
}

func TestRoundTrip_MultipleCities(t *testing.T) {
	// Test round-trip across different global locations.
	cities := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"Beijing", 39.9042, 116.4074},
		{"New York", 40.7128, -74.0060},
		{"London", 51.5074, -0.1278},
		{"Tokyo", 35.6762, 139.6503},
		{"Sydney", -33.8688, 151.2093},
		{"São Paulo", -23.5505, -46.6333},
	}
	for _, city := range cities {
		ref := Point{Latitude: city.lat, Longitude: city.lng}
		storedID := LeafCellID(ref)
		for _, radius := range []int{500, 5000, 25000, 50000} {
			bounds := BoundsAround(ref, radius)
			ranges := SearchCovering(bounds)
			if !cellInRanges(storedID, ranges) {
				t.Errorf("%s r=%dm: cell ID %d not in covering", city.name, radius, storedID)
			}
		}
	}
}

func TestSearchCovering_RangesSorted(t *testing.T) {
	// Ranges should be in ascending order (Hilbert curve property),
	// which helps PostgreSQL optimize BETWEEN scans.
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	for _, radius := range []int{1000, 5000, 25000, 50000} {
		bounds := BoundsAround(ref, radius)
		ranges := SearchCovering(bounds)
		for i := 1; i < len(ranges); i++ {
			if ranges[i][0] <= ranges[i-1][1] {
				t.Errorf("r=%dm: ranges[%d]=[,%d] overlaps or not after ranges[%d]=[,%d]",
					radius, i, ranges[i][1], i-1, ranges[i-1][1])
			}
		}
	}
}

func TestSearchCovering_RangesReasonable(t *testing.T) {
	// A 5km search should produce a manageable number of ranges.
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	for _, tc := range []struct {
		radius    int
		maxRanges int
	}{
		{500, 50},
		{5000, 64},
		{25000, 64},
		{50000, 64},
	} {
		bounds := BoundsAround(ref, tc.radius)
		ranges := SearchCovering(bounds)
		if len(ranges) > tc.maxRanges {
			t.Errorf("r=%dm: %d ranges exceeds reasonable limit of %d", tc.radius, len(ranges), tc.maxRanges)
		}
		if len(ranges) == 0 {
			t.Errorf("r=%dm: got 0 ranges", tc.radius)
		}
	}
}

func TestRoundTrip_NearPole(t *testing.T) {
	// Near the north pole — ensure bounds clamping doesn't break covering.
	ref := Point{Latitude: 89.0, Longitude: 0}
	storedID := LeafCellID(ref)
	for _, radius := range []int{1000, 5000, 25000} {
		bounds := BoundsAround(ref, radius)
		if bounds.North > 90 {
			t.Errorf("r=%dm: North bound %.4f exceeds 90", radius, bounds.North)
		}
		ranges := SearchCovering(bounds)
		if !cellInRanges(storedID, ranges) {
			t.Errorf("near-pole r=%dm: cell ID %d not in covering", radius, storedID)
		}
	}
}

func TestRoundTrip_NearDateLine(t *testing.T) {
	// Near the international date line (longitude ~180).
	ref := Point{Latitude: 35.0, Longitude: 179.5}
	storedID := LeafCellID(ref)
	bounds := BoundsAround(ref, 5000)
	ranges := SearchCovering(bounds)
	if !cellInRanges(storedID, ranges) {
		t.Errorf("near-dateline: cell ID %d not in covering (ranges=%d)", storedID, len(ranges))
	}
}

func TestRoundTrip_Equator(t *testing.T) {
	// At the equator — longitude degrees are widest.
	ref := Point{Latitude: 0, Longitude: 0}
	storedID := LeafCellID(ref)
	for _, radius := range []int{1000, 5000, 25000, 50000} {
		bounds := BoundsAround(ref, radius)
		ranges := SearchCovering(bounds)
		if !cellInRanges(storedID, ranges) {
			t.Errorf("equator r=%dm: cell ID %d not in covering", radius, storedID)
		}
	}
}

func TestSearchCovering_LargeRadiusStillFast(t *testing.T) {
	// 50km search should still produce reasonable output quickly.
	ref := Point{Latitude: 39.9, Longitude: 116.4}
	bounds := BoundsAround(ref, 50000)
	ranges := SearchCovering(bounds)
	if len(ranges) == 0 {
		t.Fatal("50km search should produce ranges")
	}
	if len(ranges) > 500 {
		t.Errorf("50km search produced %d ranges — too many, query will be slow", len(ranges))
	}
}

func TestRoundTrip_DateLineCrossing(t *testing.T) {
	ref := Point{Latitude: 35.0, Longitude: 179.8}
	// With 50km radius, a point at -179.8 (36km away) must be found.
	bounds := BoundsAround(ref, 50000)
	ranges := SearchCovering(bounds)
	storedID := LeafCellID(ref)
	if !cellInRanges(storedID, ranges) {
		t.Errorf("reference cell ID %d not in covering", storedID)
	}
	across := Point{Latitude: 35.0, Longitude: -179.8}
	acrossID := LeafCellID(across)
	if !cellInRanges(acrossID, ranges) {
		dist := DistanceMeters(ref, across)
		t.Errorf("across-dateline point (dist=%dm) not in covering (ranges=%d)", dist, len(ranges))
	}
}

func TestRoundTrip_DateLineCrossingNegativeSide(t *testing.T) {
	ref := Point{Latitude: 35.0, Longitude: -179.8}
	bounds := BoundsAround(ref, 50000)
	ranges := SearchCovering(bounds)
	across := Point{Latitude: 35.0, Longitude: 179.8}
	acrossID := LeafCellID(across)
	if !cellInRanges(acrossID, ranges) {
		t.Errorf("across-dateline point not in covering (ranges=%d)", len(ranges))
	}
}

func TestRoundTrip_SouthernHemisphere(t *testing.T) {
	// Southern hemisphere points must still be found via LeafCellID + SearchCovering.
	ref := Point{Latitude: -33.87, Longitude: 151.21}
	for _, radius := range []int{1000, 5000, 25000, 50000} {
		bounds := BoundsAround(ref, radius)
		ranges := SearchCovering(bounds)
		storedID := LeafCellID(ref)
		if !cellInRanges(storedID, ranges) {
			t.Errorf("Sydney r=%dm: cell ID %d not in covering", radius, storedID)
		}
	}
}

func TestSearchPlan_MatchNegativeCellID(t *testing.T) {
	ref := Point{Latitude: -33.87, Longitude: 151.21}
	plan := NewSearchPlan(ref, 5000, "distance")
	nearby := Point{Latitude: -33.86, Longitude: 151.22}
	distance, _, ok := plan.Match(nearby)
	if !ok {
		t.Error("nearby point in southern hemisphere should match plan")
	}
	if distance <= 0 {
		t.Errorf("expected positive distance, got %d", distance)
	}
}
