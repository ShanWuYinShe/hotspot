package reports

import (
	"context"
	"errors"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

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

// TestPostgresCreatePersistsServerSnapshot：服务端生成的消息快照随举报一并
// 落库（spec api_spec_types.md ReportSnapshot：客户端不提交快照）。
func TestPostgresCreatePersistsServerSnapshot(t *testing.T) {
	execStub := &reportExecCaptureStub{}
	repo := NewPostgresRepository(newReportFactory(execStub))
	snapshot := &ReportSnapshot{ChatType: "group", TargetChatID: "g1", MessageType: "text", SnapshotContent: types.StrPtr("hi"), SentAt: "2026-01-01T00:00:00Z"}
	if _, err := repo.Create(context.Background(), "u1", CreateReportRequest{TargetType: ReportTargetTypeMessage, TargetID: "m1", Reason: ReportReasonSpam}, snapshot); err != nil {
		t.Fatalf("create: %v", err)
	}
	if execStub.lastArgs == nil || len(execStub.lastArgs) < 9 {
		t.Fatalf("expected 9 insert args, got %#v", execStub.lastArgs)
	}
	snapshotJSON, _ := execStub.lastArgs[8].(string)
	if !strings.Contains(snapshotJSON, `"chat_type":"group"`) || !strings.Contains(snapshotJSON, `"target_chat_id":"g1"`) {
		t.Fatalf("expected serialized snapshot in insert args, got %q", snapshotJSON)
	}
}

// TestPostgresCreateWithoutSnapshot：非消息举报快照参数为 nil，落库 NULL。
func TestPostgresCreateWithoutSnapshot(t *testing.T) {
	execStub := &reportExecCaptureStub{}
	repo := NewPostgresRepository(newReportFactory(execStub))
	if _, err := repo.Create(context.Background(), "u1", CreateReportRequest{TargetType: ReportTargetTypeMoment, TargetID: "m1", Reason: ReportReasonSpam}, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	if execStub.lastArgs == nil || len(execStub.lastArgs) < 9 {
		t.Fatalf("expected 9 insert args, got %#v", execStub.lastArgs)
	}
	if execStub.lastArgs[8] != nil {
		t.Fatalf("expected nil snapshot arg, got %#v", execStub.lastArgs[8])
	}
}

// reportExecCaptureStub 捕获 Exec 参数供落库断言。
type reportExecCaptureStub struct {
	reportQueryRowStub
	lastArgs []any
}

func (s *reportExecCaptureStub) Exec(_ context.Context, _ string, args ...any) error {
	s.lastArgs = args
	return nil
}
