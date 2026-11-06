package moments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 契约计数改为读时聚合后，ListMomentsByAuthor 的 comments_count 必须来自
// comments 表真实行数（不再是 moments 写入列）。造一条动态 + 一条评论，断言
// 聚合值随行数变化。
func TestPostgresIT_ListMomentsByAuthorUsesDerivedCommentCount(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		authorID := testdb.InsertUser(t, txFactory, "m_byauthor_author", "Author")
		commenterID := testdb.InsertUser(t, txFactory, "m_byauthor_commenter", "Commenter")

		moment := createMomentWithPoint(t, ctx, repo, authorID, "ByAuthor Counts", 35.0, 112.0, "public")

		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		if err := qdb.Exec(ctx,
			"INSERT INTO comments (comment_id, moment_id, user_id, content, created_at) VALUES ('comment_byauthor_1', $1, $2, 'nice', now())",
			moment.MomentID, commenterID); err != nil {
			t.Fatalf("insert comment: %v", err)
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
		if page.Items[0].CommentsCount != 1 {
			t.Fatalf("comments count: got %d, want 1 (derived from comments table)", page.Items[0].CommentsCount)
		}
	})
}
