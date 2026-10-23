package reactions

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestAggregateReactionsBatchKeepsPerTargetOrdering 防回归（P2-21）：批量
// 聚合的行序即调用方按行序 append 的 reactions 顺序，必须与单目标版同口径
// （组内 cnt DESC, emoji ASC），缺失时 MomentItem/CommentItem.reactions 的
// 显示顺序跨请求不稳定。
func TestAggregateReactionsBatchKeepsPerTargetOrdering(t *testing.T) {
	if !strings.Contains(queryAggregateReactionsBatch, "ORDER BY r.target_id, cnt DESC, r.emoji ASC") {
		t.Fatal("queryAggregateReactionsBatch must keep per-target ordering (cnt DESC, emoji ASC)")
	}
}

// reactionRowStub 是最小可用的 Query/QueryRow stub，按查询串返回预设行。
type reactionRowStub struct {
	rowValues []any
	rows      [][]any
	rowsByKey map[string][][]any
	rowErr    error
	rowsErr   error
}

func (s *reactionRowStub) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	return reactionSingleRow{values: s.rowValues, err: s.rowErr}
}

func (s *reactionRowStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if s.rowsErr != nil {
		return nil, s.rowsErr
	}
	rows := s.rows
	if s.rowsByKey != nil {
		if byQuery, ok := s.rowsByKey[query]; ok {
			rows = byQuery
		}
	}
	// 模拟 ListReactions 的 (reacted_at, user_id) 游标过滤
	if query == queryListReactions && len(args) >= 5 {
		if at, ok := args[3].(time.Time); ok && !at.IsZero() {
			cursorUser := args[4].(string)
			filtered := make([][]any, 0, len(rows))
			for _, row := range rows {
				reactedAt := row[3].(time.Time)
				userID := row[0].(string)
				if reactedAt.Before(at) || (reactedAt.Equal(at) && userID < cursorUser) {
					filtered = append(filtered, row)
				}
			}
			rows = filtered
		}
	}
	return &reactionRows{rows: rows, err: s.rowsErr}, nil
}

func (s *reactionRowStub) Exec(_ context.Context, _ string, _ ...any) error { return nil }

type reactionSingleRow struct {
	values []any
	err    error
}

func (r reactionSingleRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *bool:
			*d = r.values[i].(bool)
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		case *time.Time:
			*d = r.values[i].(time.Time)
		}
	}
	return nil
}

type reactionRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *reactionRows) Next() bool { return r.idx < len(r.rows) }
func (r *reactionRows) Scan(dest ...any) error {
	row := reactionSingleRow{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *reactionRows) Close() error { return nil }
func (r *reactionRows) Err() error   { return r.err }

func newReactionFactory(backend persistence.Queryable) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresReactAndUnreact(t *testing.T) {
	// queryUpsertReaction 现按 SQL 内分支返回 inserted/switched/unchanged
	// 三态字符串，repo 映射为 ReactionOutcome。stub 逐态覆盖映射逻辑。
	for _, tc := range []struct {
		sqlResult string
		want      ReactionOutcome
	}{
		{"inserted", ReactionInserted},
		{"switched", ReactionSwitched},
		{"unchanged", ReactionUnchanged},
	} {
		backend := &reactionRowStub{rowValues: []any{tc.sqlResult}}
		repo := NewPostgresRepository(newReactionFactory(backend))
		outcome, err := repo.React(context.Background(), ReactTargetMoment, "moment_1", "u1", "👍")
		if err != nil || outcome != tc.want {
			t.Fatalf("react(%s): outcome=%v err=%v", tc.sqlResult, outcome, err)
		}
	}

	backend := &reactionRowStub{rowValues: []any{true}}
	repo := NewPostgresRepository(newReactionFactory(backend))
	deleted, err := repo.Unreact(context.Background(), ReactTargetMoment, "moment_1", "u1")
	if err != nil || !deleted {
		t.Fatalf("unreact: deleted=%v err=%v", deleted, err)
	}
}

func TestPostgresReactNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	_, err := repo.React(context.Background(), ReactTargetMoment, "m1", "u1", "👍")
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresAggregateAndList(t *testing.T) {
	backend := &reactionRowStub{
		rowValues: []any{"like", 3, true},
		rowsByKey: map[string][][]any{
			queryAggregateReactions: {
				{"like", 3, true},
			},
			queryListReactions: {
				{"u1", "Alice", "👍", time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)},
				{"u2", "Bob", "👍", time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)},
				{"u3", "Cara", "❤️", time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)},
			},
		},
	}
	repo := NewPostgresRepository(newReactionFactory(backend))

	agg, err := repo.Aggregate(context.Background(), ReactTargetMoment, "moment_1", "viewer")
	if err != nil || len(agg) != 1 || agg[0].Emoji != "like" || !agg[0].ReactedByMe {
		t.Fatalf("aggregate: %#v err=%v", agg, err)
	}

	items, next, err := repo.ListReactions(context.Background(), ReactTargetMoment, "moment_1", "", 2, "")
	if err != nil || len(items) != 2 || next == "" {
		t.Fatalf("list: items=%d next=%q err=%v", len(items), next, err)
	}
	if items[0].User.UserID != "u1" || items[0].Emoji != "👍" {
		t.Fatalf("unexpected first item: %#v", items[0])
	}

	// 翻页：用上一页 cursor 继续
	page2, next2, err := repo.ListReactions(context.Background(), ReactTargetMoment, "moment_1", "", 2, next)
	if err != nil || len(page2) != 1 || next2 != "" {
		t.Fatalf("page2: %d items next=%q err=%v", len(page2), next2, err)
	}
	if page2[0].User.UserID != "u3" {
		t.Fatalf("unexpected second page item: %#v", page2[0])
	}
}

func TestPostgresAggregateBatchEmptyReturnsEmptyMap(t *testing.T) {
	repo := NewPostgresRepository(newReactionFactory(&reactionRowStub{}))
	result, err := repo.AggregateBatch(context.Background(), ReactTargetMoment, nil, "viewer")
	if err != nil || len(result) != 0 {
		t.Fatalf("expected empty map, got %#v err=%v", result, err)
	}
}

func TestPostgresAggregateBatchAndListReactedTargets(t *testing.T) {
	backend := &reactionRowStub{
		rowsByKey: map[string][][]any{
			queryAggregateReactionsBatch: {
				{"m1", "👍", 2, true},
				{"m2", "❤️", 1, false},
			},
			queryListReactedTargets: {
				{"m2", time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)},
				{"m1", time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)},
			},
		},
	}
	repo := NewPostgresRepository(newReactionFactory(backend))

	batch, err := repo.AggregateBatch(context.Background(), ReactTargetMoment, []string{"m1", "m2"}, "viewer")
	if err != nil || len(batch) != 2 {
		t.Fatalf("batch: %#v err=%v", batch, err)
	}
	if len(batch["m1"]) != 1 || !batch["m1"][0].ReactedByMe || len(batch["m2"]) != 1 {
		t.Fatalf("unexpected batch result: %#v", batch)
	}

	ids, next, err := repo.ListReactedTargets(context.Background(), ReactTargetMoment, "u1", 10, "")
	if err != nil || len(ids) != 2 || next != "" {
		t.Fatalf("list reacted targets: %#v next=%q err=%v", ids, next, err)
	}
}

func TestPostgresListReactedTargetsBadCursor(t *testing.T) {
	repo := NewPostgresRepository(newReactionFactory(&reactionRowStub{}))
	_, _, err := repo.ListReactedTargets(context.Background(), ReactTargetMoment, "u1", 10, "%%%not-base64%%%")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestServiceConstructorsWire(t *testing.T) {
	svc := NewService()
	if svc == nil || svc.repo == nil {
		t.Fatal("NewService must wire in-memory repo")
	}
	svc2 := NewServiceWithRepo(NewInMemoryRepository())
	if svc2 == nil || svc2.repo == nil {
		t.Fatal("NewServiceWithRepo must wire repo")
	}
}
