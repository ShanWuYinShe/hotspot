package privatechats

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type privateChatRowValues struct {
	values []any
	err    error
}

func (r privateChatRowValues) Scan(dest ...any) error {
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

type privateChatRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *privateChatRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *privateChatRowsValues) Scan(dest ...any) error {
	row := privateChatRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *privateChatRowsValues) Close() error { return nil }
func (r *privateChatRowsValues) Err() error   { return r.err }

type privateChatFallbackRepo struct{ used bool }

type privateChatNonQueryableDB struct{}

func (privateChatNonQueryableDB) Ping(context.Context) error { return nil }
func (privateChatNonQueryableDB) Close() error               { return nil }

func (r *privateChatFallbackRepo) markUsed() { r.used = true }
func (r *privateChatFallbackRepo) GetOrCreate(context.Context, string, string) (time.Time, error) {
	r.markUsed()
	return time.Now().UTC(), nil
}
func (r *privateChatFallbackRepo) List(context.Context, string) (map[string]time.Time, error) {
	r.markUsed()
	return map[string]time.Time{"fallback": time.Now().UTC()}, nil
}

type privateChatStubRecord struct {
	ChatKey   string
	UserAID   string
	UserBID   string
	CreatedAt time.Time
}

type privateChatBackendStub struct {
	chats        map[string]privateChatStubRecord
	execErrs     map[string]error
	queryErrs    map[string]error
	queryRowErrs map[string]error
}

func (b *privateChatBackendStub) Ping(context.Context) error { return nil }
func (b *privateChatBackendStub) Close() error               { return nil }
func (b *privateChatBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	if query == queryInsertPrivateChat {
		chatKey := args[0].(string)
		if _, ok := b.chats[chatKey]; ok {
			return nil
		}
		createdAt, _ := time.Parse(time.RFC3339, args[3].(string))
		b.chats[chatKey] = privateChatStubRecord{ChatKey: chatKey, UserAID: args[1].(string), UserBID: args[2].(string), CreatedAt: createdAt}
	}
	return nil
}
func (b *privateChatBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if err := b.queryRowErrs[query]; err != nil {
		return privateChatRowValues{err: err}
	}
	if query == queryGetPrivateChatCreatedAt {
		record, ok := b.chats[args[0].(string)]
		if !ok {
			return privateChatRowValues{err: persistence.ErrNotImplemented}
		}
		return privateChatRowValues{values: []any{record.CreatedAt}}
	}
	return privateChatRowValues{err: persistence.ErrNotImplemented}
}
func (b *privateChatBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if query == queryListPrivateChats {
		userID := args[0].(string)
		rows := [][]any{}
		for _, record := range b.chats {
			if record.UserAID != userID && record.UserBID != userID {
				continue
			}
			rows = append(rows, []any{record.ChatKey, record.CreatedAt})
		}
		return &privateChatRowsValues{rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryFailsClosedWhenQueryableUnavailable(t *testing.T) {
	repo := NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     privateChatNonQueryableDB{},
	}, nil)

	if _, err := repo.GetOrCreate(context.Background(), "user_alice", "user_bob"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for get_or_create, got %v", err)
	}
	if _, err := repo.List(context.Background(), "user_alice"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for list, got %v", err)
	}
}

func TestPostgresRepositoryGetOrCreateAndList(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}, queryRowErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateChatFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	createdAt, err := repo.GetOrCreate(context.Background(), "user_alice", "user_bob")
	if err != nil {
		t.Fatalf("get or create: %v", err)
	}
	items, err := repo.List(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected items: %#v", items)
	}
	if items[key("user_alice", "user_bob")].IsZero() || createdAt.IsZero() {
		t.Fatalf("expected created time, got list=%#v created=%v", items, createdAt)
	}
	if fallback.used {
		t.Fatal("expected postgres path without fallback")
	}
}

func TestPostgresRepositoryListReturnsEmptyWithoutFallback(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}, queryRowErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateChatFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	items, err := repo.List(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %#v", items)
	}
	if fallback.used {
		t.Fatal("expected empty postgres result without fallback")
	}
}

func TestPostgresRepositoryGetOrCreateReturnsErrorOnQueryRowFailure(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}, queryRowErrs: map[string]error{queryGetPrivateChatCreatedAt: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateChatFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.GetOrCreate(context.Background(), "user_alice", "user_bob")
	if err == nil {
		t.Fatal("expected get or create error")
	}
	if fallback.used {
		t.Fatal("expected query row failure without fallback")
	}
}

func TestPostgresRepositoryListReturnsErrorOnQueryFailure(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{queryListPrivateChats: errors.New("boom")}, queryRowErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateChatFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.List(context.Background(), "user_alice")
	if err == nil {
		t.Fatal("expected list error")
	}
	if fallback.used {
		t.Fatal("expected query failure without fallback")
	}
}
