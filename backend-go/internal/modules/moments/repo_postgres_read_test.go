package moments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type rowValues struct {
	values []any
	err    error
}

func (r rowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		case *float64:
			*d = r.values[i].(float64)
		}
	}
	return nil
}

type rowsValues struct {
	rows [][]any
	idx  int
}

func (r *rowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *rowsValues) Scan(dest ...any) error {
	row := rowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *rowsValues) Close() error { return nil }
func (r *rowsValues) Err() error   { return nil }

type backendStub struct{}

func (b *backendStub) Exec(context.Context, string, ...any) error { return nil }
func (b *backendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if query == queryGetMoment {
		return rowValues{values: []any{"moment_1", "user_alice", "hello", "2026-03-20T00:00:00Z", 1, 2, 3, "南天门", 35.0, 112.0, "public"}}
	}
	return rowValues{err: persistence.ErrNotImplemented}
}
func (b *backendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if query == queryListMyMoments {
		return &rowsValues{rows: [][]any{{"moment_1", "user_alice", "hello", "2026-03-20T00:00:00Z", 1, 2, 3, "南天门", 35.0, 112.0, "public"}}}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryReadsQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &backendStub{}
	repo := NewPersistenceRepository(factory)
	item, ok := repo.Get(context.Background(), "moment_1")
	if !ok || item.MomentID != "moment_1" || item.Description != "hello" {
		t.Fatalf("unexpected item: %#v", item)
	}
	list := repo.ListMine(context.Background(), "user_alice")
	if len(list.Items) != 1 || list.Items[0].MomentID != "moment_1" {
		t.Fatalf("unexpected list: %#v", list)
	}
}
