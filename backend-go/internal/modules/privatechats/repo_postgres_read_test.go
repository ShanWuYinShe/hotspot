package privatechats

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type privateChatNonQueryableDB struct{}

func (privateChatNonQueryableDB) Ping(context.Context) error { return nil }
func (privateChatNonQueryableDB) Close() error               { return nil }

type privateChatStubRecord struct {
	ChatKey   string
	UserAID   string
	UserBID   string
	CreatedAt time.Time
}

type privateChatUserStateRecord struct {
	ChatKey      string
	UserID       string
	IsMuted      bool
	IsPinned     bool
	VisibleSince *time.Time
	DeletedAt    *time.Time
}

type privateChatBackendStub struct {
	chats        map[string]privateChatStubRecord
	states       map[string]map[string]privateChatUserStateRecord
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
	switch query {
	case queryInsertPrivateChat:
		chatKey := args[0].(string)
		if _, ok := b.chats[chatKey]; ok {
			return nil
		}
		createdAt, _ := time.Parse(time.RFC3339Nano, args[3].(string))
		b.chats[chatKey] = privateChatStubRecord{ChatKey: chatKey, UserAID: args[1].(string), UserBID: args[2].(string), CreatedAt: createdAt}
	case queryUpsertPrivateChatUserState:
		chatKey := args[0].(string)
		userID := args[1].(string)
		visibleSinceAny := args[4]
		var visibleSince *time.Time
		if visibleSinceStr, ok := visibleSinceAny.(string); ok && visibleSinceStr != "" {
			parsed, _ := time.Parse(time.RFC3339Nano, visibleSinceStr)
			visibleSince = &parsed
		}
		deletedAtAny := args[5]
		var deletedAt *time.Time
		if deletedAtStr, ok := deletedAtAny.(string); ok && deletedAtStr != "" {
			parsed, _ := time.Parse(time.RFC3339Nano, deletedAtStr)
			deletedAt = &parsed
		}
		if b.states == nil {
			b.states = map[string]map[string]privateChatUserStateRecord{}
		}
		if b.states[chatKey] == nil {
			b.states[chatKey] = map[string]privateChatUserStateRecord{}
		}
		b.states[chatKey][userID] = privateChatUserStateRecord{ChatKey: chatKey, UserID: userID, IsMuted: args[2].(bool), IsPinned: args[3].(bool), VisibleSince: visibleSince, DeletedAt: deletedAt}
	}
	return nil
}
func (b *privateChatBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if err := b.queryRowErrs[query]; err != nil {
		return testdb.RowValues{Err: err}
	}
	switch query {
	case queryGetPrivateChatCreatedAt:
		record, ok := b.chats[args[0].(string)]
		if !ok {
			return testdb.RowValues{Err: persistence.ErrNotImplemented}
		}
		return testdb.RowValues{Values: []any{record.CreatedAt}}
	case queryGetPrivateChatState:
		userID := args[0].(string)
		chatKey := args[1].(string)
		record, ok := b.chats[chatKey]
		if !ok {
			return testdb.RowValues{Err: persistence.ErrNotImplemented}
		}
		targetID := record.UserAID
		if record.UserAID == userID {
			targetID = record.UserBID
		}
		state := privateChatUserStateRecord{}
		if b.states != nil && b.states[chatKey] != nil {
			state = b.states[chatKey][userID]
		}
		return testdb.RowValues{Values: []any{record.ChatKey, targetID, record.CreatedAt, state.IsMuted, state.IsPinned, state.VisibleSince, state.DeletedAt}}
	}
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
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
			targetID := record.UserAID
			if record.UserAID == userID {
				targetID = record.UserBID
			}
			state := privateChatUserStateRecord{}
			if b.states != nil && b.states[record.ChatKey] != nil {
				state = b.states[record.ChatKey][userID]
			}
			rows = append(rows, []any{record.ChatKey, targetID, record.CreatedAt, state.IsMuted, state.IsPinned, state.VisibleSince, state.DeletedAt})
		}
		return &testdb.RowsValues{Rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryFailsClosedWhenQueryableUnavailable(t *testing.T) {
	repo := NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     privateChatNonQueryableDB{},
	})

	if _, err := repo.GetOrCreate(context.Background(), "user_alice", "user_bob"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for get_or_create, got %v", err)
	}
	if _, _, err := repo.List(context.Background(), "user_alice", ListOptions{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for list, got %v", err)
	}
}

func TestPostgresRepositoryGetOrCreateAndList(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, states: map[string]map[string]privateChatUserStateRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}, queryRowErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	createdAt, err := repo.GetOrCreate(context.Background(), "user_alice", "user_bob")
	if err != nil {
		t.Fatalf("get or create: %v", err)
	}
	items, nextCursor, err := repo.List(context.Background(), "user_alice", ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected items: %#v", items)
	}
	if items[0].ChatKey != key("user_alice", "user_bob") || items[0].TargetID != "user_bob" || items[0].CreatedAt.IsZero() || createdAt.IsZero() {
		t.Fatalf("expected created state, got list=%#v created=%v", items, createdAt)
	}
	if nextCursor != "" {
		t.Fatalf("expected empty next cursor, got %q", nextCursor)
	}
}

func TestPostgresRepositoryListReturnsEmptyWithoutFallback(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, states: map[string]map[string]privateChatUserStateRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}, queryRowErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	items, nextCursor, err := repo.List(context.Background(), "user_alice", ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %#v", items)
	}
	if nextCursor != "" {
		t.Fatalf("expected empty next cursor, got %q", nextCursor)
	}
}

func TestPostgresRepositoryGetOrCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, states: map[string]map[string]privateChatUserStateRecord{}, execErrs: map[string]error{queryInsertPrivateChat: errors.New("boom")}, queryErrs: map[string]error{}, queryRowErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.GetOrCreate(context.Background(), "user_alice", "user_bob")
	if err == nil {
		t.Fatal("expected get or create error")
	}
}

func TestPostgresRepositoryGetOrCreateReturnsErrorOnQueryRowFailure(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, states: map[string]map[string]privateChatUserStateRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}, queryRowErrs: map[string]error{queryGetPrivateChatCreatedAt: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.GetOrCreate(context.Background(), "user_alice", "user_bob")
	if err == nil {
		t.Fatal("expected get or create error")
	}
}

func TestPostgresRepositoryListReturnsErrorOnQueryFailure(t *testing.T) {
	backend := &privateChatBackendStub{chats: map[string]privateChatStubRecord{}, states: map[string]map[string]privateChatUserStateRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{queryListPrivateChats: errors.New("boom")}, queryRowErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, _, err := repo.List(context.Background(), "user_alice", ListOptions{})
	if err == nil {
		t.Fatal("expected list error")
	}
}
