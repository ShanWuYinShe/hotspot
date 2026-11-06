package moments

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestPostgresListNearbyMomentsQueryCount 锁定附近流的 DB 往返：关注标记已随
// 候选粗筛一并计算，页面详情再 1 次查询——共 2 次（架构审计 perf D2，此前
// 「候选 + 关注批量 + 详情」3 次）。
func TestPostgresListNearbyMomentsQueryCount(t *testing.T) {
	now := time.Now().UTC()
	backend := newBackendStub()
	backend.nearbyCandidates = [][]any{
		{"m_followed", "u_followed", now, 1, 34.2604, 117.1855, true},
		{"m_other", "u_other", now.Add(-time.Minute), 50, 34.2604, 117.1855, false},
	}
	detailRows := func(momentID, userID, description string, following bool) []any {
		return []any{momentID, userID, userID + "_nick", "", description, 0, 0, 0, "", []string{}, false}
	}
	backend.nearbyDetails = map[string][]any{
		"m_followed": detailRows("m_followed", "u_followed", "followed", true),
		"m_other":    detailRows("m_other", "u_other", "other", false),
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	page, err := repo.ListNearbyMoments(context.Background(), momentsQuery{
		ViewerUserID:      "viewer",
		ReferencePoint:    GeoPoint{Latitude: 34.2604, Longitude: 117.1855},
		HasReferencePoint: true,
		Radius:            5000,
		Limit:             20,
	})
	if err != nil {
		t.Fatalf("ListNearbyMoments: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(page.Items))
	}
	// 关注加分让 m_followed 排在更高热度的 m_other 之前（距离相同）。
	if page.Items[0].MomentID != "m_followed" {
		t.Fatalf("following candidate should rank first, got %q", page.Items[0].MomentID)
	}
	if len(backend.queries) != 2 {
		t.Fatalf("nearby roundtrips = %d (%v), want 2", len(backend.queries), backend.queries)
	}
	if backend.queries[0] != queryNearbyMoments || backend.queries[1] != queryNearbyMomentDetails {
		t.Fatalf("unexpected query order: %v", backend.queries)
	}
}
