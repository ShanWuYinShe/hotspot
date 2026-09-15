package moments

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// insertTestRelation 直插 user_relations（测试关注/拉黑关系的最小构造器）。
func insertTestRelation(t *testing.T, factory *persistence.Factory, userID, targetUserID string, following, blocked bool) {
	t.Helper()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	now := time.Now().UTC()
	const q = `INSERT INTO user_relations (user_id, target_user_id, is_following, followed_at, is_blocked, blocked_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, target_user_id) DO UPDATE SET is_following = EXCLUDED.is_following, is_blocked = EXCLUDED.is_blocked`
	if err := qdb.Exec(context.Background(), q, userID, targetUserID, following, now, blocked, now); err != nil {
		t.Fatalf("insert user_relations %s->%s: %v", userID, targetUserID, err)
	}
}

// TestPostgresIT_BlockFiltersDiscoveryFeeds 验证发现面（following/nearby/search）
// 的双向拉黑过滤：任一方拉黑即互不可见（安全修复 M-2 的 SQL 侧兜底）。
func TestPostgresIT_BlockFiltersDiscoveryFeeds(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		authorID := testdb.InsertUser(t, txFactory, "m_blk_author", "Author")
		otherID := testdb.InsertUser(t, txFactory, "m_blk_other", "Other")
		viewerID := testdb.InsertUser(t, txFactory, "m_blk_viewer", "Viewer")

		createMomentWithPoint(t, ctx, repo, authorID, "block alpha moment", 35.001, 112.001, "public")
		createMomentWithPoint(t, ctx, repo, otherID, "block beta moment", 35.001, 112.002, "public")

		// viewer 关注 author 与 other：拉黑前两条 feed 都可见。
		insertTestRelation(t, txFactory, viewerID, authorID, true, false)
		insertTestRelation(t, txFactory, viewerID, otherID, true, false)

		assertDiscoveryCounts(t, ctx, repo, viewerID, 2, "baseline")

		// 方向一：viewer 拉黑 author → author 的动态从三个发现面消失。
		insertTestRelation(t, txFactory, viewerID, authorID, true, true)
		assertDiscoveryCounts(t, ctx, repo, viewerID, 1, "viewer blocks author")

		// 方向二：解除对 author 的拉黑，other 拉黑 viewer → other 的动态消失
		//（被拉黑方不得通过 feed 继续看到对方内容）。
		insertTestRelation(t, txFactory, viewerID, authorID, true, false)
		insertTestRelation(t, txFactory, otherID, viewerID, false, true)
		assertDiscoveryCounts(t, ctx, repo, viewerID, 1, "other blocks viewer")
	})
}

func assertDiscoveryCounts(t *testing.T, ctx context.Context, repo *PostgresRepository, viewerID string, want int, phase string) {
	t.Helper()
	following, err := repo.ListFollowingMoments(ctx, momentsQuery{ViewerUserID: viewerID, Limit: 20})
	if err != nil {
		t.Fatalf("[%s] ListFollowingMoments: %v", phase, err)
	}
	if got := len(following.Items); got != want {
		t.Fatalf("[%s] following feed: want %d items, got %d", phase, want, got)
	}
	nearby, err := repo.ListNearbyMoments(ctx, momentsQuery{
		ViewerUserID:      viewerID,
		ReferencePoint:    GeoPoint{Latitude: 35.0, Longitude: 112.0},
		HasReferencePoint: true,
		Radius:            5000,
		Limit:             20,
	})
	if err != nil {
		t.Fatalf("[%s] ListNearbyMoments: %v", phase, err)
	}
	if got := len(nearby.Items); got != want {
		t.Fatalf("[%s] nearby feed: want %d items, got %d", phase, want, got)
	}
	kw := "block"
	search, err := repo.Search(ctx, SearchMomentsRequest{
		Keyword:    &kw,
		SortBy:     "newest",
		Pagination: &PaginationParams{Limit: 20},
	}, viewerID)
	if err != nil {
		t.Fatalf("[%s] Search: %v", phase, err)
	}
	if got := len(search.Items); got != want {
		t.Fatalf("[%s] search: want %d items, got %d", phase, want, got)
	}
}
