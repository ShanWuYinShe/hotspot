package notifications

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
		case *bool:
			*d = r.values[i].(bool)
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
	if query == queryListNotifications {
		return &rowsValues{rows: [][]any{{"notif_1", "system_announcement", false, "2026-03-20T00:00:00Z"}}}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryReadsQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &backendStub{}
	repo := NewPostgresRepository(factory, nil)
	list := repo.List(context.Background(), "user_alice", NotificationListRequest{})
	if len(list.Items) != 1 || list.Items[0].NotificationID != "notif_1" {
		t.Fatalf("unexpected list: %#v", list)
	}
	count := repo.UnreadCount(context.Background(), "user_alice")
	if count.Total != 1 {
		t.Fatalf("unexpected count: %#v", count)
	}
}
