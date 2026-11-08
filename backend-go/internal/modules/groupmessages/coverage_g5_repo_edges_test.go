package groupmessages

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"

	"github.com/jackc/pgx/v5"
)

// gmCovQueryable 是可编排的 persistence.Queryable 测试替身：用于覆盖
// PostgresRepository 的「未配置 / 查询失败 / 扫描失败 / 无行」分支，
// 不依赖真实库（真实库主路径由 IT 与契约测试覆盖）。
type gmCovQueryable struct {
	exec     func(ctx context.Context, sql string, args ...any) error
	queryRow func(ctx context.Context, sql string, args ...any) persistence.Row
	query    func(ctx context.Context, sql string, args ...any) (persistence.Rows, error)
}

func (q *gmCovQueryable) Exec(ctx context.Context, sql string, args ...any) error {
	return q.exec(ctx, sql, args...)
}

func (q *gmCovQueryable) QueryRow(ctx context.Context, sql string, args ...any) persistence.Row {
	return q.queryRow(ctx, sql, args...)
}

func (q *gmCovQueryable) Query(ctx context.Context, sql string, args ...any) (persistence.Rows, error) {
	return q.query(ctx, sql, args...)
}

func newGmCovQueryable() *gmCovQueryable {
	return &gmCovQueryable{
		exec: func(context.Context, string, ...any) error { return nil },
		queryRow: func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: errors.New("unexpected QueryRow")}
		},
		query: func(context.Context, string, ...any) (persistence.Rows, error) {
			return nil, errors.New("unexpected Query")
		},
	}
}

// gmCovScanErrRows 模拟「有行但 Scan 失败」：rows.Next() 恒 true，
// Scan 直接返回错误，用来覆盖各 repo 循环内的扫描错误分支。
type gmCovScanErrRows struct{}

func (gmCovScanErrRows) Next() bool        { return true }
func (gmCovScanErrRows) Scan(...any) error { return errors.New("scan boom") }
func (gmCovScanErrRows) Close() error      { return nil }
func (gmCovScanErrRows) Err() error        { return nil }

// gmCovRepoWith 把替身绑定到 ctx（RepoQueryable 优先取 ctx 内事务），
// 使 repo 无需真实 factory 即可执行。
func gmCovRepoWith(q persistence.Queryable) (*PostgresRepository, context.Context) {
	ctx := persistence.WithTx(context.Background(), q, nil)
	return NewPostgresRepository(nil), ctx
}

// gmCovHistoryRow 返回 queryGroupMessagesHistory 的 23 列行（列序与 Scan 一致）。
func gmCovHistoryRow(messageID, fileID string) []any {
	return []any{
		messageID, "text", "hello", "2026-01-01T00:00:00.000000Z", "", "",
		fileID, "", "", "", "", "", "", "", "", "",
		"u1", "nick", "", "", "", "", []string{},
	}
}

// TestPostgresRepositoryNotConfiguredBranches：factory 未接 PG 时所有仓储方法
// 返回 ErrNotImplemented（模块回落内存实现的契约）。
func TestPostgresRepositoryNotConfiguredBranches(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()

	if _, err := repo.History(ctx, "g1", nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("History: %v", err)
	}
	if _, err := repo.RecallAsAdmin(ctx, "m1", "op1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("RecallAsAdmin: %v", err)
	}
	if err := repo.MarkAsRead(ctx, "u1", MarkGroupMessagesReadRequest{GroupID: "g1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MarkAsRead: %v", err)
	}
	if _, err := repo.MarkAllRead(ctx, "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MarkAllRead: %v", err)
	}
	if _, err := repo.GetReadCount(ctx, "g1", "m1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetReadCount: %v", err)
	}
	if _, err := repo.GetReadCounts(ctx, "g1", []string{"m1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetReadCounts: %v", err)
	}
	if _, err := repo.LatestMessageID(ctx, "g1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("LatestMessageID: %v", err)
	}
	if _, err := repo.FilterMemberIDs(ctx, "g1", []string{"u1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("FilterMemberIDs: %v", err)
	}
	if _, err := repo.IsMember(ctx, "g1", "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("IsMember: %v", err)
	}
	if _, _, _, err := repo.MemberMuteStatus(ctx, "g1", "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MemberMuteStatus: %v", err)
	}
	if _, _, err := repo.GetMessageForReply(ctx, "m1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetMessageForReply: %v", err)
	}
	if _, _, err := repo.MessageExists(ctx, "m1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MessageExists: %v", err)
	}
	if _, _, err := repo.MessageGroupID(ctx, "m1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MessageGroupID: %v", err)
	}
	if _, _, err := repo.GetGroupName(ctx, "g1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetGroupName: %v", err)
	}
	if _, _, err := repo.LookupMessageMeta(ctx, "m1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("LookupMessageMeta: %v", err)
	}
	if _, _, err := repo.GetMessageReportSnapshot(ctx, "m1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetMessageReportSnapshot: %v", err)
	}
	if _, _, err := repo.GetMediaForMessage(ctx, "f1", "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("GetMediaForMessage: %v", err)
	}
}

// TestPostgresRepositoryQueryErrorBranches 覆盖查询/扫描错误的上抛分支：
// 这些错误不能被吞成 not-found，否则真实故障会被静默降级。
func TestPostgresRepositoryQueryErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	recent := time.Now().UTC().Format(time.RFC3339Nano)
	lookupValues := []any{"g1", "u1", "text", false, recent}

	t.Run("Update generic error", func(t *testing.T) {
		q := newGmCovQueryable()
		calls := 0
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			calls++
			if calls == 1 {
				return testdb.RowValues{Values: lookupValues}
			}
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.Update(ctx, UpdateGroupMessageRequest{MessageID: "m1", UserID: "u1", Content: "x"}); !errors.Is(err, boom) {
			t.Fatalf("Update err = %v", err)
		}
	})

	t.Run("Update concurrent recall", func(t *testing.T) {
		q := newGmCovQueryable()
		calls := 0
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			calls++
			if calls == 1 {
				return testdb.RowValues{Values: lookupValues}
			}
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.Update(ctx, UpdateGroupMessageRequest{MessageID: "m1", UserID: "u1", Content: "x"}); !errors.Is(err, ErrMessageRecalled) {
			t.Fatalf("Update err = %v", err)
		}
	})

	t.Run("Recall invalid sent_at fails closed", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Values: []any{"g1", "u1", "text", false, "not-a-time"}}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.Recall(ctx, time.Minute, RecallGroupMessageRequest{MessageID: "m1", UserID: "u1"}); !errors.Is(err, ErrMessageRecallWindow) {
			t.Fatalf("Recall err = %v", err)
		}
	})

	t.Run("Recall concurrent recall", func(t *testing.T) {
		q := newGmCovQueryable()
		calls := 0
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			calls++
			if calls == 1 {
				return testdb.RowValues{Values: lookupValues}
			}
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.Recall(ctx, time.Minute, RecallGroupMessageRequest{MessageID: "m1", UserID: "u1"}); !errors.Is(err, ErrMessageRecalled) {
			t.Fatalf("Recall err = %v", err)
		}
	})

	t.Run("Recall update error", func(t *testing.T) {
		q := newGmCovQueryable()
		calls := 0
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			calls++
			if calls == 1 {
				return testdb.RowValues{Values: lookupValues}
			}
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.Recall(ctx, time.Minute, RecallGroupMessageRequest{MessageID: "m1", UserID: "u1"}); !errors.Is(err, boom) {
			t.Fatalf("Recall err = %v", err)
		}
	})

	t.Run("RecallAsAdmin concurrent recall", func(t *testing.T) {
		q := newGmCovQueryable()
		calls := 0
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			calls++
			if calls == 1 {
				return testdb.RowValues{Values: lookupValues}
			}
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.RecallAsAdmin(ctx, "m1", "admin"); !errors.Is(err, ErrMessageRecalled) {
			t.Fatalf("RecallAsAdmin err = %v", err)
		}
	})

	t.Run("RecallAsAdmin update error", func(t *testing.T) {
		q := newGmCovQueryable()
		calls := 0
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			calls++
			if calls == 1 {
				return testdb.RowValues{Values: lookupValues}
			}
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.RecallAsAdmin(ctx, "m1", "admin"); !errors.Is(err, boom) {
			t.Fatalf("RecallAsAdmin err = %v", err)
		}
	})

	t.Run("MarkAsRead state lookup error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		err := repo.MarkAsRead(ctx, "u1", MarkGroupMessagesReadRequest{GroupID: "g1", UpToMessageID: "m1"})
		if !errors.Is(err, boom) {
			t.Fatalf("MarkAsRead err = %v", err)
		}
	})

	t.Run("MarkAllRead exec error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.exec = func(context.Context, string, ...any) error { return boom }
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.MarkAllRead(ctx, "u1"); !errors.Is(err, boom) {
			t.Fatalf("MarkAllRead err = %v", err)
		}
	})

	t.Run("MarkAllRead query error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(context.Context, string, ...any) (persistence.Rows, error) { return nil, boom }
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.MarkAllRead(ctx, "u1"); !errors.Is(err, boom) {
			t.Fatalf("MarkAllRead err = %v", err)
		}
	})

	t.Run("MarkAllRead scan error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(context.Context, string, ...any) (persistence.Rows, error) {
			return gmCovScanErrRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.MarkAllRead(ctx, "u1"); err == nil {
			t.Fatal("expected scan error")
		}
	})

	t.Run("GetReadCount error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.GetReadCount(ctx, "g1", "m1"); !errors.Is(err, boom) {
			t.Fatalf("GetReadCount err = %v", err)
		}
	})

	t.Run("GetReadCounts empty stays zero round-trip", func(t *testing.T) {
		repo, ctx := gmCovRepoWith(newGmCovQueryable())
		counts, err := repo.GetReadCounts(ctx, "g1", nil)
		if err != nil || len(counts) != 0 {
			t.Fatalf("GetReadCounts empty = %#v err=%v", counts, err)
		}
	})

	t.Run("GetReadCounts query error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(context.Context, string, ...any) (persistence.Rows, error) { return nil, boom }
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.GetReadCounts(ctx, "g1", []string{"m1"}); !errors.Is(err, boom) {
			t.Fatalf("GetReadCounts err = %v", err)
		}
	})

	t.Run("GetReadCounts scan error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(context.Context, string, ...any) (persistence.Rows, error) {
			return gmCovScanErrRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.GetReadCounts(ctx, "g1", []string{"m1"}); err == nil {
			t.Fatal("expected scan error")
		}
	})

	t.Run("LatestMessageID error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.LatestMessageID(ctx, "g1"); !errors.Is(err, boom) {
			t.Fatalf("LatestMessageID err = %v", err)
		}
	})

	t.Run("FilterMemberIDs query error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(context.Context, string, ...any) (persistence.Rows, error) { return nil, boom }
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.FilterMemberIDs(ctx, "g1", []string{"u1"}); !errors.Is(err, boom) {
			t.Fatalf("FilterMemberIDs err = %v", err)
		}
	})

	t.Run("FilterMemberIDs scan error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(context.Context, string, ...any) (persistence.Rows, error) {
			return gmCovScanErrRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.FilterMemberIDs(ctx, "g1", []string{"u1"}); err == nil {
			t.Fatal("expected scan error")
		}
	})

	t.Run("IsMember error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.IsMember(ctx, "g1", "u1"); !errors.Is(err, boom) {
			t.Fatalf("IsMember err = %v", err)
		}
	})

	t.Run("MessageExists error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, _, err := repo.MessageExists(ctx, "m1"); !errors.Is(err, boom) {
			t.Fatalf("MessageExists err = %v", err)
		}
	})

	t.Run("MessageGroupID error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, _, err := repo.MessageGroupID(ctx, "m1"); !errors.Is(err, boom) {
			t.Fatalf("MessageGroupID err = %v", err)
		}
	})

	t.Run("lookupMessageMetaFull not configured", func(t *testing.T) {
		repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
		if _, _, _, _, _, err := repo.lookupMessageMetaFull(context.Background(), "m1"); !errors.Is(err, persistence.ErrNotImplemented) {
			t.Fatalf("lookupMessageMetaFull err = %v", err)
		}
	})

	t.Run("GetGroupName error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.queryRow = func(context.Context, string, ...any) persistence.Row {
			return testdb.RowValues{Err: boom}
		}
		repo, ctx := gmCovRepoWith(q)
		if _, _, err := repo.GetGroupName(ctx, "g1"); !errors.Is(err, boom) {
			t.Fatalf("GetGroupName err = %v", err)
		}
	})
}

// TestPostgresRepositoryHistoryErrorBranches 覆盖 History 的扫描/已读计数/
// 媒体回填错误分支：任何一步失败都必须整体上抛，不能返回半页数据。
func TestPostgresRepositoryHistoryErrorBranches(t *testing.T) {
	boom := errors.New("boom")

	t.Run("history scan error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(_ context.Context, sql string, _ ...any) (persistence.Rows, error) {
			if sql == queryGroupMessagesHistory {
				return gmCovScanErrRows{}, nil
			}
			return &persistence.InMemoryRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.History(ctx, "g1", nil); err == nil {
			t.Fatal("expected scan error")
		}
	})

	t.Run("history rows.Err error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(_ context.Context, sql string, _ ...any) (persistence.Rows, error) {
			if sql == queryGroupMessagesHistory {
				return &testdb.RowsValues{Rows: [][]any{gmCovHistoryRow("m1", "")}, TailErr: boom}, nil
			}
			return &persistence.InMemoryRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.History(ctx, "g1", nil); !errors.Is(err, boom) {
			t.Fatalf("history err = %v", err)
		}
	})

	t.Run("read counts query error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(_ context.Context, sql string, _ ...any) (persistence.Rows, error) {
			switch sql {
			case queryGroupMessagesHistory:
				return &testdb.RowsValues{Rows: [][]any{gmCovHistoryRow("m1", "")}}, nil
			case queryGroupMessageReadCounts:
				return nil, boom
			}
			return &persistence.InMemoryRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.History(ctx, "g1", nil); !errors.Is(err, boom) {
			t.Fatalf("history err = %v", err)
		}
	})

	t.Run("read counts scan error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(_ context.Context, sql string, _ ...any) (persistence.Rows, error) {
			switch sql {
			case queryGroupMessagesHistory:
				return &testdb.RowsValues{Rows: [][]any{gmCovHistoryRow("m1", "")}}, nil
			case queryGroupMessageReadCounts:
				return gmCovScanErrRows{}, nil
			}
			return &persistence.InMemoryRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.History(ctx, "g1", nil); err == nil {
			t.Fatal("expected read count scan error")
		}
	})

	t.Run("media query error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(_ context.Context, sql string, _ ...any) (persistence.Rows, error) {
			switch sql {
			case queryGroupMessagesHistory:
				return &testdb.RowsValues{Rows: [][]any{gmCovHistoryRow("m1", "file_1")}}, nil
			case queryGroupMessageReadCounts:
				return &testdb.RowsValues{Rows: [][]any{{"m1", 0}}}, nil
			case queryGroupMediaFilesByIDs:
				return nil, boom
			}
			return &persistence.InMemoryRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.History(ctx, "g1", nil); !errors.Is(err, boom) {
			t.Fatalf("history err = %v", err)
		}
	})

	t.Run("media scan error", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(_ context.Context, sql string, _ ...any) (persistence.Rows, error) {
			switch sql {
			case queryGroupMessagesHistory:
				return &testdb.RowsValues{Rows: [][]any{gmCovHistoryRow("m1", "file_1")}}, nil
			case queryGroupMessageReadCounts:
				return &testdb.RowsValues{Rows: [][]any{{"m1", 0}}}, nil
			case queryGroupMediaFilesByIDs:
				return gmCovScanErrRows{}, nil
			}
			return &persistence.InMemoryRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		if _, err := repo.History(ctx, "g1", nil); err == nil {
			t.Fatal("expected media scan error")
		}
	})

	t.Run("media backfill success", func(t *testing.T) {
		q := newGmCovQueryable()
		q.query = func(_ context.Context, sql string, _ ...any) (persistence.Rows, error) {
			switch sql {
			case queryGroupMessagesHistory:
				return &testdb.RowsValues{Rows: [][]any{gmCovHistoryRow("m1", "file_1")}}, nil
			case queryGroupMessageReadCounts:
				return &testdb.RowsValues{Rows: [][]any{{"m1", 2}}}, nil
			case queryGroupMediaFilesByIDs:
				return &testdb.RowsValues{Rows: [][]any{{"file_1", "uploads/f1.jpg", "f1.jpg", 10, "image/jpeg", "thumb.jpg", 640, 480, 0}}}, nil
			}
			return &persistence.InMemoryRows{}, nil
		}
		repo, ctx := gmCovRepoWith(q)
		history, err := repo.History(ctx, "g1", nil)
		if err != nil {
			t.Fatalf("history: %v", err)
		}
		if len(history.Items) != 1 {
			t.Fatalf("items = %d", len(history.Items))
		}
		item := history.Items[0]
		if item.Media == nil || item.Media["url"] != "uploads/f1.jpg" {
			t.Fatalf("media = %#v", item.Media)
		}
		if item.ReadCount == nil || *item.ReadCount != 2 {
			t.Fatalf("read_count = %#v", item.ReadCount)
		}
	})
}
