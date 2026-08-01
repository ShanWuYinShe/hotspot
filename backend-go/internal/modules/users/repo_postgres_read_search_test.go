package users

import (
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/geo"
)

// TestBuildSearchUsersQuery_PushesDownDistanceAndOrder verifies the generated
// SQL computes distance in the DB (Haversine), filters by radius, orders and
// paginates there — i.e. the AGENTS.md rule that distance/range/sort must
// happen in PostgreSQL, not in Go. This is a white-box test of the query
// builder; it does not hit a database.
func TestBuildSearchUsersQuery_PushesDownDistanceAndOrder(t *testing.T) {
	ref := geo.Point{Latitude: 35.0, Longitude: 112.0}
	req := SearchUsersRequest{Radius: intPtr(5000), SortBy: "distance", ReferencePoint: GeoPoint{Latitude: ref.Latitude, Longitude: ref.Longitude}}
	// SearchProfiles fills cursor.SortBy from the normalized request sort on
	// the first page; mirror that here.
	sqlStr, args, err := buildSearchUsersQuery(req, ref, 5000, searchUsersCursor{SortBy: "distance"}, 20)
	if err != nil {
		t.Fatalf("buildSearchUsersQuery failed: %v", err)
	}
	if sqlStr == "" {
		t.Fatal("expected non-empty SQL")
	}
	checks := map[string]string{
		"distance column":   "AS distance_meters",
		"relevance column":  "AS relevance_score",
		"radius filter":     "distance_meters <= ",
		"S2 cell prefilter": "unnest(",
		"distance ordering": "ORDER BY distance_meters ASC",
		"keyset limit":      "LIMIT ",
	}
	for name, needle := range checks {
		if !strings.Contains(sqlStr, needle) {
			t.Errorf("SQL missing %s: expected to contain %q\nSQL: %s", name, needle, sqlStr)
		}
	}
	if len(args) == 0 {
		t.Error("expected positional args, got none")
	}
}

// TestBuildSearchUsersQuery_NewestOrder verifies the newest sort path orders by
// created_at instead of distance.
func TestBuildSearchUsersQuery_NewestOrder(t *testing.T) {
	ref := geo.Point{Latitude: 35.0, Longitude: 112.0}
	req := SearchUsersRequest{Radius: intPtr(5000), SortBy: "newest", ReferencePoint: GeoPoint{Latitude: ref.Latitude, Longitude: ref.Longitude}}
	sqlStr, _, err := buildSearchUsersQuery(req, ref, 5000, searchUsersCursor{SortBy: "newest"}, 20)
	if err != nil {
		t.Fatalf("buildSearchUsersQuery failed: %v", err)
	}
	if !strings.Contains(sqlStr, "ORDER BY created_at DESC") {
		t.Errorf("expected newest ordering, SQL: %s", sqlStr)
	}
}

// TestBuildSearchUsersQuery_RelevanceOrderWithKeyword verifies the relevance
// path is chosen when no explicit distance/newest sort is set, and that the
// relevance expression references the keyword.
func TestBuildSearchUsersQuery_RelevanceOrderWithKeyword(t *testing.T) {
	ref := geo.Point{Latitude: 35.0, Longitude: 112.0}
	req := SearchUsersRequest{Keyword: "alice", ReferencePoint: GeoPoint{Latitude: ref.Latitude, Longitude: ref.Longitude}}
	sqlStr, _, err := buildSearchUsersQuery(req, ref, 5000, searchUsersCursor{}, 20)
	if err != nil {
		t.Fatalf("buildSearchUsersQuery failed: %v", err)
	}
	if !strings.Contains(sqlStr, "ORDER BY relevance_score DESC") {
		t.Errorf("expected relevance ordering, SQL: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "LOWER(u.nickname)") {
		t.Errorf("expected nickname relevance CASE, SQL: %s", sqlStr)
	}
}

// TestDecodeSearchUsersCursor_RejectsTampered verifies an invalid cursor is
// rejected (the keyset cursor is opaque AES-GCM; clients cannot forge it).
func TestDecodeSearchUsersCursor_RejectsTampered(t *testing.T) {
	if _, err := decodeSearchUsersCursor("not-a-valid-cursor"); err == nil {
		t.Fatal("expected error for tampered cursor, got nil")
	}
	// Empty cursor decodes to a zero value (first page), not an error.
	if c, err := decodeSearchUsersCursor(""); err != nil || c.UserID != "" {
		t.Fatalf("expected zero cursor for empty input, got %+v err=%v", c, err)
	}
}

func intPtr(v int) *int { return &v }
