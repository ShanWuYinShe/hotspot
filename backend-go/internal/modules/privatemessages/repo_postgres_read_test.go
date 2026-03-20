package private_messages

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
	if query == queryPrivateMessagesHistory {
		return &rowsValues{rows: [][]any{{"pm_1", "text", "hello", "", "user_alice", "2026-03-20T00:00:00Z"}}}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryReadsQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &backendStub{}
	repo := NewPostgresRepository(factory, nil)
	history := repo.History(context.Background(), "user_alice", "user_bob")
	if len(history.Items) != 1 || history.Items[0].MessageID != "pm_1" {
		t.Fatalf("unexpected history: %#v", history)
	}
}
