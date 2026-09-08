package types

import (
	"encoding/json"
	"testing"
)

func TestNormalizeLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{name: "negative limit fallback", input: -10, expected: DefaultPageSize},
		{name: "zero limit fallback", input: 0, expected: DefaultPageSize},
		{name: "normal limit 1", input: 1, expected: 1},
		{name: "normal limit 50", input: 50, expected: 50},
		{name: "exact max page size", input: MaxPageSize, expected: MaxPageSize},
		{name: "exceed max page size", input: 101, expected: MaxPageSize},
		{name: "large excess", input: 9999, expected: MaxPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeLimit(tt.input); got != tt.expected {
				t.Fatalf("NormalizeLimit(%d) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPaginationLimit(t *testing.T) {
	if got := PaginationLimit(nil); got != 0 {
		t.Fatalf("PaginationLimit(nil) = %d, want 0", got)
	}

	params := &PaginationParams{Limit: 42, Cursor: "c1"}
	if got := PaginationLimit(params); got != 42 {
		t.Fatalf("PaginationLimit(params) = %d, want 42", got)
	}
}

func TestPaginationLimitWith(t *testing.T) {
	fallback := 15
	max := 60

	tests := []struct {
		name     string
		params   *PaginationParams
		expected int
	}{
		{name: "nil params", params: nil, expected: fallback},
		{name: "zero limit", params: &PaginationParams{Limit: 0}, expected: fallback},
		{name: "negative limit", params: &PaginationParams{Limit: -5}, expected: fallback},
		{name: "within range", params: &PaginationParams{Limit: 30}, expected: 30},
		{name: "equal max", params: &PaginationParams{Limit: 60}, expected: 60},
		{name: "exceed max", params: &PaginationParams{Limit: 80}, expected: max},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PaginationLimitWith(tt.params, fallback, max); got != tt.expected {
				t.Fatalf("PaginationLimitWith(...) = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestPaginationCursor(t *testing.T) {
	if got := PaginationCursor(nil); got != "" {
		t.Fatalf("PaginationCursor(nil) = %q, want empty", got)
	}

	params := &PaginationParams{Limit: 10, Cursor: "next_cursor_token"}
	if got := PaginationCursor(params); got != "next_cursor_token" {
		t.Fatalf("PaginationCursor(params) = %q, want next_cursor_token", got)
	}
}

func TestGeoPointAndRelativeLocationJSON(t *testing.T) {
	dist := 123.456
	bearing := 89.12
	loc := RelativeLocation{
		Point: GeoPoint{
			Latitude:  31.2304,
			Longitude: 121.4737,
		},
		LocationName:   "上海人民广场",
		Distance:       &dist,
		BearingDegrees: &bearing,
	}

	bytes, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed RelativeLocation
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed.Point.Latitude != 31.2304 || parsed.Point.Longitude != 121.4737 {
		t.Fatalf("unexpected point coords: %+v", parsed.Point)
	}
	if parsed.LocationName != "上海人民广场" {
		t.Fatalf("unexpected location name: %s", parsed.LocationName)
	}
	if parsed.Distance == nil || *parsed.Distance != 123.456 {
		t.Fatalf("unexpected distance: %v", parsed.Distance)
	}
	if parsed.BearingDegrees == nil || *parsed.BearingDegrees != 89.12 {
		t.Fatalf("unexpected bearing: %v", parsed.BearingDegrees)
	}

	// 测试 Distance/BearingDegrees 为 nil 的场景（未计算参考点）
	locWithoutRef := RelativeLocation{
		Point: GeoPoint{
			Latitude:  39.9042,
			Longitude: 116.4074,
		},
		LocationName: "北京天安门",
	}
	b2, err := json.Marshal(locWithoutRef)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var parsed2 RelativeLocation
	if err := json.Unmarshal(b2, &parsed2); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if parsed2.Distance != nil || parsed2.BearingDegrees != nil {
		t.Fatalf("expected nil distance and bearing, got dist=%v, bearing=%v", parsed2.Distance, parsed2.BearingDegrees)
	}
}
