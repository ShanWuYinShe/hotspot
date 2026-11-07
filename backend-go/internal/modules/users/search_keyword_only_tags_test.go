package users

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestSearchProfilesKeywordOnlyLoadsTagsForPage 覆盖密集关键词 ranked 路径：
// SQL 直接按 relevance/keyset 返回页（不再有 500 候选上限），主查询按页内
// 用户内联标签数组（页大小受 MaxPageSize 约束，array_agg 成本可忽略）。
func TestSearchProfilesKeywordOnlyLoadsTagsForPage(t *testing.T) {
	backend := &userBackendStub{
		queryRowByQuery: map[string]persistence.Row{},
		queryRowsByQuery: map[string][][]any{
			"FROM users u": {{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", time.Now().UTC(), time.Now().UTC(), false, []string{"美食", "摄影"}, true, true, true}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	page, err := repo.SearchProfiles(context.Background(), SearchUsersRequest{Keyword: "Alice", ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("SearchProfiles: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}
	if len(page.Items[0].Tags) != 2 {
		t.Fatalf("page tags = %v, want 2 inline", page.Items[0].Tags)
	}
	if page.Settings == nil || !page.Settings["user_alice"].PresenceVisible {
		t.Fatalf("inline settings = %#v, want presence visible", page.Settings)
	}
}
