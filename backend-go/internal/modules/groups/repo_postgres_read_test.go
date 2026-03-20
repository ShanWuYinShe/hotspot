package groups

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
	if query == queryGetGroup {
		return rowValues{values: []any{"group_1", "Hot Group", "", "desc", false, "南天门", 35.0, 112.0, "open", true, 200}}
	}
	return rowValues{err: persistence.ErrNotImplemented}
}
func (b *backendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if query == querySearchGroups {
		return &rowsValues{rows: [][]any{{"group_1", "Hot Group", "", "desc", false, "南天门", 35.0, 112.0, "open", true, 200}}}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryReadsQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &backendStub{}
	repo := NewPostgresRepository(factory, nil)
	item, ok := repo.Get(context.Background(), "group_1")
	if !ok || item.Group.GroupID != "group_1" {
		t.Fatalf("unexpected item: %#v", item)
	}
	list := repo.Search(context.Background(), "Hot Group")
	if len(list.Items) != 1 || list.Items[0].Group.GroupID != "group_1" {
		t.Fatalf("unexpected list: %#v", list)
	}
}
