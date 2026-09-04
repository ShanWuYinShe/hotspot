package moments

// 迁移自 users 模块 TestListUserMomentsPostgresReturnsNextCursorWithoutDupingNextPage：
// 防重复分页回归（第二页不得重复第一页尾部锚点行）。
import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

type userMomentCursorBackend struct {
	firstPage  [][]any
	secondPage [][]any
	tags       [][]any
}

func (b *userMomentCursorBackend) Ping(context.Context) error                 { return nil }
func (b *userMomentCursorBackend) Close() error                               { return nil }
func (b *userMomentCursorBackend) Exec(context.Context, string, ...any) error { return nil }
func (b *userMomentCursorBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *userMomentCursorBackend) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	switch query {
	case queryListMomentsByAuthor:
		if len(args) >= 4 && args[2] != nil && args[3] == "moment_2" {
			return &testdb.RowsValues{Rows: b.secondPage}, nil
		}
		return &testdb.RowsValues{Rows: b.firstPage}, nil
	case queryListMomentTagsByMomentIDs:
		return &testdb.RowsValues{Rows: b.tags}, nil
	default:
		return &testdb.RowsValues{}, nil
	}
}

func TestListMomentsByAuthorReturnsNextCursorWithoutDupingNextPage(t *testing.T) {
	createdAt1 := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	createdAt2 := time.Date(2026, 3, 29, 11, 0, 0, 0, time.UTC)
	createdAt3 := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	backend := &userMomentCursorBackend{
		firstPage: [][]any{
			{"moment_3", "user_alice", "Alice", "third", createdAt3, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
			{"moment_2", "user_alice", "Alice", "second", createdAt2, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
			{"moment_1", "user_alice", "Alice", "first", createdAt1, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
		},
		secondPage: [][]any{
			{"moment_1", "user_alice", "Alice", "first", createdAt1, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
		},
		tags: [][]any{{"moment_3", "travel"}},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	page1, err := repo.ListMomentsByAuthor(context.Background(), "user_alice", 2, "", true)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("expected 2 items, got %#v", page1.Items)
	}
	if page1.Items[0].MomentID != "moment_3" || page1.Items[1].MomentID != "moment_2" {
		t.Fatalf("unexpected first page ids: %#v", page1.Items)
	}
	if page1.NextCursor == nil {
		t.Fatal("expected next cursor")
	}
	cursorAt, cursorID, err := decodeMomentsByUserCursor(*page1.NextCursor)
	if err != nil {
		t.Fatalf("decode next cursor: %v", err)
	}
	if cursorID != "moment_2" || cursorAt.UTC().Format(time.RFC3339) != createdAt2.Format(time.RFC3339) {
		t.Fatalf("unexpected next cursor anchor: %v %s", cursorAt, cursorID)
	}

	page2, err := repo.ListMomentsByAuthor(context.Background(), "user_alice", 2, *page1.NextCursor, true)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].MomentID != "moment_1" {
		t.Fatalf("unexpected second page items: %#v", page2.Items)
	}
	if page2.NextCursor != nil {
		t.Fatalf("expected empty next cursor on second page, got %v", page2.NextCursor)
	}
	for _, item := range page2.Items {
		if item.MomentID == "moment_2" || item.MomentID == "moment_3" {
			t.Fatalf("expected no duplicate items across pages, got %#v", page2.Items)
		}
	}
}
