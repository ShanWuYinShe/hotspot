package users

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestSearchProfilesKeywordOnlyLoadsTagsForPage 覆盖 keyword-only 搜索路径：
// 主查询不带标签列（避免 500 候选逐行 array_agg），分页后按页面 ID 批量补标签。
func TestSearchProfilesKeywordOnlyLoadsTagsForPage(t *testing.T) {
	backend := &userBackendStub{
		queryRowByQuery: map[string]persistence.Row{},
		queryRowsByQuery: map[string][][]any{
			"FROM users u":           {{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", time.Now().UTC(), time.Now().UTC(), false, true, true, true}},
			"FROM user_profile_tags": {{"user_alice", "美食"}, {"user_alice", "摄影"}},
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
		t.Fatalf("page tags = %v, want 2 (loaded after pagination)", page.Items[0].Tags)
	}
	if page.Settings == nil || !page.Settings["user_alice"].PresenceVisible {
		t.Fatalf("inline settings = %#v, want presence visible", page.Settings)
	}
}
