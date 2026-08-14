package reports

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type reportRowValues struct {
	values []any
	err    error
}

func (r reportRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case **string:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(string); ok {
				*d = &v
			}
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(time.Time); ok {
				*d = &v
			}
		}
	}
	return nil
}

type reportRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *reportRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *reportRowsValues) Scan(dest ...any) error {
	row := reportRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *reportRowsValues) Close() error { return nil }
func (r *reportRowsValues) Err() error   { return r.err }

type reportBackendStub struct {
	reports   map[string][]ReportItem
	execErrs  map[string]error
	queryErrs map[string]error
}

func newReportBackendStub() *reportBackendStub {
	return &reportBackendStub{reports: make(map[string][]ReportItem), execErrs: make(map[string]error), queryErrs: make(map[string]error)}
}

func (b *reportBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	if query == queryInsertReport {
		userID := args[1].(string)
		item := ReportItem{ReportID: args[0].(string), TargetType: ReportTargetType(args[2].(string)), TargetID: args[3].(string), Reason: ReportReason(args[4].(string)), Description: args[5].(string), Status: ReportStatus(args[6].(string)), CreatedAt: args[7].(string)}
		b.reports[userID] = append(b.reports[userID], item)
	}
	return nil
}

func (b *reportBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return reportRowValues{err: persistence.ErrNotImplemented}
}

func (b *reportBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if query != queryListReports {
		return &persistence.InMemoryRows{}, nil
	}
	userID := args[0].(string)
	items := append([]ReportItem(nil), b.reports[userID]...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ReportID > items[j].ReportID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, []any{item.ReportID, item.TargetType, item.TargetID, item.Reason, item.Description, item.Status, item.CreatedAt, nil, nil, ""})
	}
	return &reportRowsValues{rows: rows}, nil
}

func TestPostgresRepositoryCreateAndList(t *testing.T) {
	backend := newReportBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	created, err := repo.Create(context.Background(), "user_alice", CreateReportRequest{TargetType: "moment", TargetID: "moment_1", Reason: "spam", Description: "bad"})
	if err != nil || created.ReportID == "" {
		t.Fatalf("unexpected create result: %#v err=%v", created, err)
	}
	list, err := repo.List(context.Background(), "user_alice", ListReportsRequest{})
	if err != nil || len(list.Items) != 1 || list.Items[0].ReportID != created.ReportID {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
}

func TestPostgresRepositoryListReturnsEmptyWithoutFallback(t *testing.T) {
	backend := newReportBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	list, err := repo.List(context.Background(), "user_alice", ListReportsRequest{})
	if err != nil || len(list.Items) != 0 {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := newReportBackendStub()
	backend.execErrs[queryInsertReport] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Create(context.Background(), "user_alice", CreateReportRequest{TargetType: "moment", TargetID: "moment_1", Reason: "spam"})
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestPostgresRepositoryListReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newReportBackendStub()
	backend.queryErrs[queryListReports] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.List(context.Background(), "user_alice", ListReportsRequest{})
	if err == nil {
		t.Fatal("expected list error")
	}
}
