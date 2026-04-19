package groups

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type groupRowValues struct {
	values []any
	err    error
}

func (r groupRowValues) Scan(dest ...any) error {
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
		case *[]string:
			*d = r.values[i].([]string)
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else {
				value := r.values[i].(time.Time)
				*d = &value
			}
		}
	}
	return nil
}

type groupRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *groupRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *groupRowsValues) Scan(dest ...any) error {
	row := groupRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *groupRowsValues) Close() error { return nil }
func (r *groupRowsValues) Err() error   { return r.err }

type groupBackendStub struct {
	rows      [][]any
	queryErrs map[string]error
	queryRow  persistence.Row
	execErrs  map[string]error
}

func (b *groupBackendStub) Ping(context.Context) error { return nil }
func (b *groupBackendStub) Close() error               { return nil }
func (b *groupBackendStub) Exec(_ context.Context, query string, _ ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	return nil
}
func (b *groupBackendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if query == queryGetGroup {
		return b.queryRow
	}
	if query == queryCountGroupMembers {
		return groupRowValues{values: []any{1}}
	}
	if query == queryGetGroupMembership {
		return groupRowValues{err: pgx.ErrNoRows}
	}
	return groupRowValues{err: persistence.ErrNotImplemented}
}
func (b *groupBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if strings.Contains(query, "FROM groups g") {
		if err := b.queryErrs[querySearchGroups]; err != nil {
			return nil, err
		}
		return &groupRowsValues{rows: b.rows}, nil
	}
	if query == querySearchGroups {
		return &groupRowsValues{rows: b.rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryCreateGetAndSearch(t *testing.T) {
	backend := &groupBackendStub{
		rows:      [][]any{{"group_1", "Hot Group", "", "desc", false, "南天门", 35.0, 112.0, "open", true, 200, "user_alice", nil, nil, []string{"hiking"}, 1, 0, 1}},
		queryErrs: map[string]error{},
		queryRow:  groupRowValues{values: []any{"group_1", "Hot Group", "", "desc", false, "南天门", 35.0, 112.0, "open", true, 200, "user_alice", nil, nil, []string{"hiking"}}},
		execErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	created, err := repo.Create(context.Background(), "user_alice", CreateGroupRequest{Name: "Hot Group", Location: Location{LocationName: "南天门"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Group.GroupID == "" {
		t.Fatalf("expected created group id, got %#v", created)
	}
	if created.OwnerID != "user_alice" || created.CreatedAt == nil {
		t.Fatalf("expected owner metadata on created group, got %#v", created)
	}
	item, ok, err := repo.Get(context.Background(), "group_1")
	if err != nil || !ok || item.Group.GroupID != "group_1" {
		t.Fatalf("unexpected item: %#v err=%v", item, err)
	}
	if item.OwnerID != "user_alice" {
		t.Fatalf("expected owner id in fetched item, got %#v", item)
	}
	list, err := repo.Search(context.Background(), SearchGroupsRequest{Keyword: "Hot Group", ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil || len(list.Items) != 1 || list.Items[0].Group.GroupID != "group_1" {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
}

func TestPostgresRepositoryGetReturnsNotFoundWithoutFallback(t *testing.T) {
	backend := &groupBackendStub{rows: nil, queryErrs: map[string]error{}, queryRow: groupRowValues{err: errors.New("no rows")}, execErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, ok, err := repo.Get(context.Background(), "missing")
	if err == nil && ok {
		t.Fatalf("expected not found, got ok=%v err=%v", ok, err)
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := &groupBackendStub{rows: nil, queryErrs: map[string]error{}, queryRow: groupRowValues{err: persistence.ErrNotImplemented}, execErrs: map[string]error{queryInsertGroup: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Create(context.Background(), "user_alice", CreateGroupRequest{Name: "Hot Group", Location: Location{LocationName: "南天门"}})
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestPostgresRepositorySearchReturnsErrorOnQueryFailure(t *testing.T) {
	backend := &groupBackendStub{rows: nil, queryErrs: map[string]error{querySearchGroups: errors.New("boom")}, queryRow: groupRowValues{err: persistence.ErrNotImplemented}, execErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Search(context.Background(), SearchGroupsRequest{Keyword: "Hot Group", ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err == nil {
		t.Fatal("expected search error")
	}
}
