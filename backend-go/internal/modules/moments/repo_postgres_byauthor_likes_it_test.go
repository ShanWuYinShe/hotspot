package moments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 审计 F-08 收口：queryListMomentsByAuthor 的点赞数必须来自 reactions 实时
// COUNT（likes_count 列已无写路径、恒为 0）。造一条动态 + 两条 reaction，
// 断言 ListMomentsByAuthor 返回活跃点赞数而非死列。
func TestPostgresIT_ListMomentsByAuthorUsesLiveReactionCount(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		authorID := testdb.InsertUser(t, txFactory, "m_byauthor_author", "Author")
		reactorA := testdb.InsertUser(t, txFactory, "m_byauthor_react_a", "ReactorA")
		reactorB := testdb.InsertUser(t, txFactory, "m_byauthor_react_b", "ReactorB")

		moment := createMomentWithPoint(t, ctx, repo, authorID, "ByAuthor Likes", 35.0, 112.0, "public")

		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		for _, uid := range []string{reactorA, reactorB} {
			if err := qdb.Exec(ctx,
				"INSERT INTO reactions (target_type, target_id, user_id, emoji) VALUES ('moment', $1, $2, 'like')",
				moment.MomentID, uid); err != nil {
				t.Fatalf("insert reaction: %v", err)
			}
		}

		page, err := repo.ListMomentsByAuthor(ctx, authorID, 10, "", false)
		if err != nil {
			t.Fatalf("ListMomentsByAuthor: %v", err)
		}
		if len(page.Items) != 1 {
			t.Fatalf("expected 1 moment, got %d", len(page.Items))
		}
		if page.Items[0].MomentID != moment.MomentID {
			t.Fatalf("moment id: got %q, want %q", page.Items[0].MomentID, moment.MomentID)
		}
		if page.Items[0].LikesCount != 2 {
			t.Fatalf("likes count: got %d, want 2 (live reactions count)", page.Items[0].LikesCount)
		}
	})
}
