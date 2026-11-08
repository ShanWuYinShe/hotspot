package comments

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// coverBackend 是可编程的 PostgresQueryBackend 替身，用于覆盖
// repo_postgres.go 的错误分支（既有 backendStub 只模拟成功路径）。
type coverBackend struct {
	execErr  error
	queryRow func(query string, args ...any) persistence.Row
	query    func(query string, args ...any) (persistence.Rows, error)
}

func (b *coverBackend) Exec(context.Context, string, ...any) error { return b.execErr }

func (b *coverBackend) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if b.queryRow == nil {
		return persistence.InMemoryRow{}
	}
	return b.queryRow(query, args...)
}

func (b *coverBackend) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if b.query == nil {
		return &persistence.InMemoryRows{}, nil
	}
	return b.query(query, args...)
}

// coverRows 是可编程的 Rows 替身：scanErr 触发逐行扫描失败，err 触发
// rows.Err() 失败（Pg 驱动在遍历结束后才暴露的传输错误）。
type coverRows struct {
	rows    [][]any
	idx     int
	scanErr error
	err     error
}

func (r *coverRows) Next() bool { return r.idx < len(r.rows) }

func (r *coverRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := rowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}

func (r *coverRows) Close() error { return nil }
func (r *coverRows) Err() error   { return r.err }

func newCoverRepo(backend persistence.PostgresQueryBackend) *PostgresRepository {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return NewPostgresRepository(factory)
}

func TestNormalizeCommentLimitClamps(t *testing.T) {
	cases := map[int]int{-5: 20, 0: 20, 1: 1, 30: 30, 100: 100, 101: 100, 500: 100}
	for input, want := range cases {
		if got := normalizeCommentLimit(input); got != want {
			t.Fatalf("normalizeCommentLimit(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestPostgresRepositoryUnconfiguredReturnsNotImplemented(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()
	if _, err := repo.Create(ctx, "user_1", CreateCommentRequest{MomentID: "moment_1", Content: "hi"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("create must report ErrNotImplemented, got %v", err)
	}
	if _, found, err := repo.Get(ctx, "comment_1"); !errors.Is(err, persistence.ErrNotImplemented) || found {
		t.Fatalf("get must report ErrNotImplemented, got found=%v err=%v", found, err)
	}
	if _, err := repo.List(ctx, "moment_1", 20, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("list must report ErrNotImplemented, got %v", err)
	}
	if _, err := repo.ListReplies(ctx, "comment_1", 20, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("list replies must report ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryCreateNicknameFallbacks(t *testing.T) {
	t.Run("empty nickname falls back to user id", func(t *testing.T) {
		backend := &coverBackend{queryRow: func(query string, _ ...any) persistence.Row {
			if query == queryGetNickname {
				return rowValues{values: []any{"", ""}}
			}
			return rowValues{err: persistence.ErrNotImplemented}
		}}
		created, err := newCoverRepo(backend).Create(context.Background(), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "hi"})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if created.User.UserID != "user_alice" || created.User.Nickname != "user_alice" {
			t.Fatalf("empty nickname must fall back to user id: %#v", created.User)
		}
		if created.User.AvatarURL != nil {
			t.Fatalf("empty avatar must stay nil: %#v", created.User.AvatarURL)
		}
	})

	t.Run("nickname lookup error falls back to user id", func(t *testing.T) {
		backend := &coverBackend{queryRow: func(query string, _ ...any) persistence.Row {
			if query == queryGetNickname {
				return rowValues{err: errors.New("lookup failed")}
			}
			return rowValues{err: persistence.ErrNotImplemented}
		}}
		created, err := newCoverRepo(backend).Create(context.Background(), "user_bob", CreateCommentRequest{MomentID: "moment_1", Content: "hi"})
		if err != nil {
			t.Fatalf("create must tolerate nickname lookup failure: %v", err)
		}
		if created.User.Nickname != "user_bob" {
			t.Fatalf("nickname lookup failure must fall back to user id, got %q", created.User.Nickname)
		}
	})

	t.Run("nickname lookup no rows keeps user id", func(t *testing.T) {
		backend := &coverBackend{queryRow: func(query string, _ ...any) persistence.Row {
			if query == queryGetNickname {
				return rowValues{err: pgx.ErrNoRows}
			}
			return rowValues{err: persistence.ErrNotImplemented}
		}}
		created, err := newCoverRepo(backend).Create(context.Background(), "user_carol", CreateCommentRequest{MomentID: "moment_1", Content: "hi"})
		if err != nil {
			t.Fatalf("create must tolerate missing user row: %v", err)
		}
		if created.User.Nickname != "user_carol" {
			t.Fatalf("missing user row must fall back to user id, got %q", created.User.Nickname)
		}
	})

	t.Run("nickname and avatar are read from users", func(t *testing.T) {
		backend := &coverBackend{queryRow: func(query string, _ ...any) persistence.Row {
			if query == queryGetNickname {
				return rowValues{values: []any{"Alice", "https://cdn.example/alice.png"}}
			}
			return rowValues{err: persistence.ErrNotImplemented}
		}}
		created, err := newCoverRepo(backend).Create(context.Background(), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "hi", ParentID: "comment_parent"})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if created.User.Nickname != "Alice" || created.User.AvatarURL == nil || *created.User.AvatarURL != "https://cdn.example/alice.png" {
			t.Fatalf("user profile must be echoed from users table: %#v", created.User)
		}
		if created.ParentID == nil || *created.ParentID != "comment_parent" {
			t.Fatalf("parent id must be echoed: %#v", created.ParentID)
		}
	})
}

func TestPostgresRepositoryGetErrorPaths(t *testing.T) {
	repo := newCoverRepo(&coverBackend{queryRow: func(string, ...any) persistence.Row {
		return rowValues{err: pgx.ErrNoRows}
	}})
	item, found, err := repo.Get(context.Background(), "comment_missing")
	if err != nil || found || item.CommentID != "" {
		t.Fatalf("missing comment must be (zero,false,nil), got %#v found=%v err=%v", item, found, err)
	}

	repo = newCoverRepo(&coverBackend{queryRow: func(string, ...any) persistence.Row {
		return rowValues{err: errors.New("scan failed")}
	}})
	if _, _, err := repo.Get(context.Background(), "comment_1"); err == nil {
		t.Fatal("scan failure must be surfaced")
	}
}

func TestPostgresRepositorySoftDeleteScanError(t *testing.T) {
	repo := newCoverRepo(&coverBackend{queryRow: func(string, ...any) persistence.Row {
		return rowValues{err: errors.New("delete failed")}
	}})
	if _, err := repo.SoftDelete(context.Background(), "comment_1"); err == nil {
		t.Fatal("soft delete scan failure must be surfaced")
	}
}

func TestPostgresRepositoryListErrorPaths(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return nil, errors.New("query failed")
		}})
		if _, err := repo.List(context.Background(), "moment_1", 20, ""); err == nil {
			t.Fatal("list query failure must be surfaced")
		}
	})
	t.Run("scan error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return &coverRows{rows: [][]any{{}}, scanErr: errors.New("scan failed")}, nil
		}})
		if _, err := repo.List(context.Background(), "moment_1", 20, ""); err == nil {
			t.Fatal("list scan failure must be surfaced")
		}
	})
	t.Run("rows iter error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return &coverRows{err: errors.New("rows failed")}, nil
		}})
		if _, err := repo.List(context.Background(), "moment_1", 20, ""); err == nil {
			t.Fatal("list rows failure must be surfaced")
		}
	})
}

func TestPostgresRepositoryListRepliesErrorPaths(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return nil, errors.New("query failed")
		}})
		if _, err := repo.ListReplies(context.Background(), "comment_1", 20, ""); err == nil {
			t.Fatal("reply query failure must be surfaced")
		}
	})
	t.Run("scan error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return &coverRows{rows: [][]any{{}}, scanErr: errors.New("scan failed")}, nil
		}})
		if _, err := repo.ListReplies(context.Background(), "comment_1", 20, ""); err == nil {
			t.Fatal("reply scan failure must be surfaced")
		}
	})
	t.Run("rows iter error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return &coverRows{err: errors.New("rows failed")}, nil
		}})
		if _, err := repo.ListReplies(context.Background(), "comment_1", 20, ""); err == nil {
			t.Fatal("reply rows failure must be surfaced")
		}
	})
}

func TestPostgresRepositoryListTopRepliesEarlyReturnAndErrors(t *testing.T) {
	repo := newCoverRepo(&coverBackend{})
	if result, err := repo.ListTopRepliesByParentIDs(context.Background(), nil, 3); err != nil || len(result) != 0 {
		t.Fatalf("empty parents must be a no-op, got %#v err=%v", result, err)
	}
	if result, err := repo.ListTopRepliesByParentIDs(context.Background(), []string{"comment_1"}, 0); err != nil || len(result) != 0 {
		t.Fatalf("non-positive limit must be a no-op, got %#v err=%v", result, err)
	}

	t.Run("rows are grouped by parent", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return &coverRows{rows: [][]any{{
				"comment_child", "user_1", "u1", "", "comment_parent", "reply", "2026-03-28T10:00:00Z", 0, false, []string{"user_2"},
			}}}, nil
		}})
		result, err := repo.ListTopRepliesByParentIDs(context.Background(), []string{"comment_parent"}, 3)
		if err != nil {
			t.Fatalf("batch top replies: %v", err)
		}
		replies := result["comment_parent"]
		if len(replies) != 1 || replies[0].CommentID != "comment_child" || replies[0].ParentID == nil || *replies[0].ParentID != "comment_parent" {
			t.Fatalf("unexpected grouped replies: %#v", result)
		}
		if replies[0].CreatedAt != "2026-03-28T10:00:00Z" {
			t.Fatalf("created_at must be normalized to UTC RFC3339, got %q", replies[0].CreatedAt)
		}
	})

	t.Run("query error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return nil, errors.New("query failed")
		}})
		if _, err := repo.ListTopRepliesByParentIDs(context.Background(), []string{"comment_parent"}, 3); err == nil {
			t.Fatal("batch query failure must be surfaced")
		}
	})
	t.Run("scan error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return &coverRows{rows: [][]any{{}}, scanErr: errors.New("scan failed")}, nil
		}})
		if _, err := repo.ListTopRepliesByParentIDs(context.Background(), []string{"comment_parent"}, 3); err == nil {
			t.Fatal("batch scan failure must be surfaced")
		}
	})
	t.Run("rows iter error", func(t *testing.T) {
		repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
			return &coverRows{err: errors.New("rows failed")}, nil
		}})
		if _, err := repo.ListTopRepliesByParentIDs(context.Background(), []string{"comment_parent"}, 3); err == nil {
			t.Fatal("batch rows failure must be surfaced")
		}
	})
}

// TestPostgresRepositoryListBeyondLimitPaginates 覆盖「多取一行用于判断
// 是否还有下一页」的分支：limit+1 行时截断并返回游标。
func TestPostgresRepositoryListBeyondLimitPaginates(t *testing.T) {
	rows := func(n int) [][]any {
		out := make([][]any, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, []any{
				"comment_x", "user_1", "u1", "", "", "hi", "2026-03-28T10:00:00Z", 0, false, []string{},
			})
		}
		return out
	}
	repo := newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
		return &coverRows{rows: rows(3)}, nil
	}})
	page, err := repo.List(context.Background(), "moment_1", 2, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == nil || *page.NextCursor == "" {
		t.Fatalf("expected truncated page with cursor, got %#v", page)
	}

	repo = newCoverRepo(&coverBackend{query: func(string, ...any) (persistence.Rows, error) {
		return &coverRows{rows: rows(3)}, nil
	}})
	replyPage, err := repo.ListReplies(context.Background(), "comment_1", 2, "")
	if err != nil {
		t.Fatalf("list replies: %v", err)
	}
	if len(replyPage.Items) != 2 || replyPage.NextCursor == nil {
		t.Fatalf("expected truncated reply page with cursor, got %#v", replyPage)
	}
}
