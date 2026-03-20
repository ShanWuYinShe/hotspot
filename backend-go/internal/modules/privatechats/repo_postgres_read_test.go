package privatechats

import (
	"context"
	"testing"
	"time"

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
		case *time.Time:
			*d = r.values[i].(time.Time)
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
func (b *backendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return rowValues{err: persistence.ErrNotImplemented}
}
func (b *backendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if query == queryListPrivateChats {
		return &rowsValues{rows: [][]any{{"user_alice:user_bob", time.Now().UTC()}}}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryReadsQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &backendStub{}
	repo := NewPostgresRepository(factory, nil)
	items := repo.List(context.Background(), "user_alice")
	if len(items) != 1 {
		t.Fatalf("unexpected items: %#v", items)
	}
}
