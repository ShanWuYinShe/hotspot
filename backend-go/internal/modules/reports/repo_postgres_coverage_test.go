package reports

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

// reportQueryRowStub 提供 QueryRow 结果供 FindExistingReport/TargetOwner 使用。
type reportQueryRowStub struct {
	values []any
	err    error
}

func (s *reportQueryRowStub) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	return reportSingleRowValues{values: s.values, err: s.err}
}

func (s *reportQueryRowStub) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func (s *reportQueryRowStub) Exec(_ context.Context, _ string, _ ...any) error { return nil }

type reportSingleRowValues struct {
	values []any
	err    error
}

func (r reportSingleRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		if d, ok := dest[i].(*string); ok {
			*d = r.values[i].(string)
		}
	}
	return nil
}

func newReportFactory(backend persistence.Queryable) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresFindExistingReport(t *testing.T) {
	repo := NewPostgresRepository(newReportFactory(&reportQueryRowStub{values: []any{"report_1"}}))
	id, found, err := repo.FindExistingReport(context.Background(), "u1", ReportTargetTypeMoment, "m1")
	if err != nil || !found || id != "report_1" {
		t.Fatalf("find existing: id=%q found=%v err=%v", id, found, err)
	}
}

func TestPostgresFindExistingReportMissing(t *testing.T) {
	// 无行 = not-found；DB 故障（如 deadline）必须原样上抛而非吞成 not-found。
	repo := NewPostgresRepository(newReportFactory(&reportQueryRowStub{err: context.DeadlineExceeded}))
	id, found, err := repo.FindExistingReport(context.Background(), "u1", ReportTargetTypeMoment, "m1")
	if !errors.Is(err, context.DeadlineExceeded) || found || id != "" {
		t.Fatalf("db error must propagate: id=%q found=%v err=%v", id, found, err)
	}
	repo = NewPostgresRepository(newReportFactory(&reportQueryRowStub{err: pgx.ErrNoRows}))
	id, found, err = repo.FindExistingReport(context.Background(), "u1", ReportTargetTypeMoment, "m1")
	if err != nil || found || id != "" {
		t.Fatalf("missing report should be not-found: id=%q found=%v err=%v", id, found, err)
	}
}

func TestPostgresTargetOwnerBranches(t *testing.T) {
	for _, tt := range []struct {
		targetType ReportTargetType
		query      string
	}{
		{ReportTargetTypeUser, queryReportTargetOwnerUser},
		{ReportTargetTypeMoment, queryReportTargetOwnerMoment},
		{ReportTargetTypeComment, queryReportTargetOwnerComment},
		{ReportTargetTypeMessage, queryReportTargetOwnerMessage},
		{ReportTargetTypeGroup, queryReportTargetOwnerGroup},
	} {
		repo := NewPostgresRepository(newReportFactory(&reportQueryRowStub{values: []any{"owner_1"}}))
		owner, found, err := repo.TargetOwner(context.Background(), tt.targetType, "t1")
		if err != nil || !found || owner != "owner_1" {
			t.Fatalf("%s: owner=%q found=%v err=%v", tt.targetType, owner, found, err)
		}
	}
	// 非法 target type → 不存在
	repo := NewPostgresRepository(newReportFactory(&reportQueryRowStub{}))
	owner, found, err := repo.TargetOwner(context.Background(), ReportTargetType("bogus"), "t1")
	if err != nil || found || owner != "" {
		t.Fatalf("bogus type: owner=%q found=%v err=%v", owner, found, err)
	}
}

func TestReportSnapshotValidation(t *testing.T) {
	if isValidReportSnapshot(nil) {
		t.Fatal("nil snapshot must be invalid")
	}
	if isValidReportSnapshot(&ReportSnapshot{ChatType: "bogus", TargetChatID: "x", MessageType: "text", SentAt: "now"}) {
		t.Fatal("bogus chat_type must be invalid")
	}
	if isValidReportSnapshot(&ReportSnapshot{ChatType: "private", TargetChatID: "", MessageType: "text", SentAt: "now"}) {
		t.Fatal("empty target_chat_id must be invalid")
	}
	if !isValidReportSnapshot(&ReportSnapshot{ChatType: "group", TargetChatID: "g1", MessageType: "text", SentAt: "now"}) {
		t.Fatal("valid snapshot must pass")
	}
}
