package groupmessages

import (
	"context"
	"errors"
	"sort"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type groupMessageRowValues struct {
	values []any
	err    error
}

func (r groupMessageRowValues) Scan(dest ...any) error {
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

type groupMessageRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *groupMessageRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *groupMessageRowsValues) Scan(dest ...any) error {
	row := groupMessageRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *groupMessageRowsValues) Close() error { return nil }
func (r *groupMessageRowsValues) Err() error   { return r.err }

type groupMessageFallbackRepo struct{ used bool }

func (r *groupMessageFallbackRepo) Send(context.Context, SendGroupMessageRequest) (MessageSent, error) {
	r.used = true
	return MessageSent{}, nil
}
func (r *groupMessageFallbackRepo) History(context.Context, string) (PaginatedGroupMessages, error) {
	r.used = true
	return PaginatedGroupMessages{}, nil
}
func (r *groupMessageFallbackRepo) Update(context.Context, UpdateGroupMessageRequest) error {
	r.used = true
	return nil
}
func (r *groupMessageFallbackRepo) Recall(context.Context, RecallGroupMessageRequest) error {
	r.used = true
	return nil
}
func (r *groupMessageFallbackRepo) MarkAsRead(context.Context, string, MarkGroupMessagesReadRequest) error {
	r.used = true
	return nil
}
func (r *groupMessageFallbackRepo) MarkAllRead(context.Context, string) error {
	r.used = true
	return nil
}

type groupMessageStubRecord struct {
	MessageID string
	GroupID   string
	Type      string
	Content   string
	SentAt    string
}

type groupMessageReadState struct {
	MessageID string
	SentAt    string
}

type groupMessageBackendStub struct {
	items      map[string][]groupMessageStubRecord
	readStates map[string]groupMessageReadState
	execErrs   map[string]error
	queryErrs  map[string]error
}

func groupMessageReadStateKey(groupID, userID string) string {
	return groupID + ":" + userID
}

func (b *groupMessageBackendStub) Ping(context.Context) error { return nil }
func (b *groupMessageBackendStub) Close() error               { return nil }
func (b *groupMessageBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	if query == queryInsertGroupMessage {
		record := groupMessageStubRecord{MessageID: args[0].(string), GroupID: args[1].(string), Type: args[3].(string), Content: args[4].(string), SentAt: args[5].(string)}
		b.items[record.GroupID] = append(b.items[record.GroupID], record)
		return nil
	}
	if query == queryUpsertGroupMessageReadState {
		groupID := args[0].(string)
		userID := args[1].(string)
		messageID := args[2].(string)
		sentAt := args[3].(string)
		key := groupMessageReadStateKey(groupID, userID)
		current, ok := b.readStates[key]
		if !ok || current.SentAt <= sentAt {
			b.readStates[key] = groupMessageReadState{MessageID: messageID, SentAt: sentAt}
		}
		return nil
	}
	if query == queryMarkGroupMessagesReadToLatest {
		userID := args[0].(string)
		groupID := args[1].(string)
		records := b.items[groupID]
		if len(records) == 0 {
			return nil
		}
		sorted := append([]groupMessageStubRecord(nil), records...)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].SentAt == sorted[j].SentAt {
				return sorted[i].MessageID > sorted[j].MessageID
			}
			return sorted[i].SentAt > sorted[j].SentAt
		})
		latest := sorted[0]
		key := groupMessageReadStateKey(groupID, userID)
		current, ok := b.readStates[key]
		if !ok || current.SentAt <= latest.SentAt {
			b.readStates[key] = groupMessageReadState{MessageID: latest.MessageID, SentAt: latest.SentAt}
		}
		return nil
	}
	if query == queryMarkAllGroupMessagesRead {
		userID := args[0].(string)
		for groupID, records := range b.items {
			if len(records) == 0 {
				continue
			}
			sorted := append([]groupMessageStubRecord(nil), records...)
			sort.Slice(sorted, func(i, j int) bool {
				if sorted[i].SentAt == sorted[j].SentAt {
					return sorted[i].MessageID > sorted[j].MessageID
				}
				return sorted[i].SentAt > sorted[j].SentAt
			})
			latest := sorted[0]
			key := groupMessageReadStateKey(groupID, userID)
			current, ok := b.readStates[key]
			if !ok || current.SentAt <= latest.SentAt {
				b.readStates[key] = groupMessageReadState{MessageID: latest.MessageID, SentAt: latest.SentAt}
			}
		}
	}
	return nil
}
func (b *groupMessageBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if query == queryGroupMessageReadStateByMessage {
		groupID := args[1].(string)
		messageID := args[2].(string)
		for _, record := range b.items[groupID] {
			if record.MessageID == messageID {
				return groupMessageRowValues{values: []any{record.SentAt, record.MessageID}}
			}
		}
		return groupMessageRowValues{err: persistence.ErrNotImplemented}
	}
	return groupMessageRowValues{err: persistence.ErrNotImplemented}
}
func (b *groupMessageBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if query == queryGroupMessagesHistory {
		groupID := args[0].(string)
		records := append([]groupMessageStubRecord(nil), b.items[groupID]...)
		sort.Slice(records, func(i, j int) bool {
			if records[i].SentAt == records[j].SentAt {
				return records[i].MessageID > records[j].MessageID
			}
			return records[i].SentAt > records[j].SentAt
		})
		rows := make([][]any, 0, len(records))
		for _, record := range records {
			rows = append(rows, []any{record.MessageID, record.Type, record.Content, record.SentAt})
		}
		return &groupMessageRowsValues{rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositorySendAndHistory(t *testing.T) {
	backend := &groupMessageBackendStub{items: map[string][]groupMessageStubRecord{}, readStates: map[string]groupMessageReadState{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &groupMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	sent, err := repo.Send(context.Background(), SendGroupMessageRequest{GroupID: "group_1", Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	history, err := repo.History(context.Background(), "group_1")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history.Items) != 1 || history.Items[0].MessageID != sent.MessageID {
		t.Fatalf("unexpected history: %#v", history)
	}
	if fallback.used {
		t.Fatal("expected postgres path without fallback")
	}
}

func TestPostgresRepositoryHistoryReturnsEmptyWithoutFallback(t *testing.T) {
	backend := &groupMessageBackendStub{items: map[string][]groupMessageStubRecord{}, readStates: map[string]groupMessageReadState{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &groupMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	history, err := repo.History(context.Background(), "group_1")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history.Items) != 0 {
		t.Fatalf("expected empty history, got %#v", history)
	}
	if fallback.used {
		t.Fatal("expected empty postgres result without fallback")
	}
}

func TestPostgresRepositorySendReturnsErrorOnExecFailure(t *testing.T) {
	backend := &groupMessageBackendStub{items: map[string][]groupMessageStubRecord{}, execErrs: map[string]error{queryInsertGroupMessage: errors.New("boom")}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &groupMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.Send(context.Background(), SendGroupMessageRequest{GroupID: "group_1", Type: "text", Content: "hello"})
	if err == nil {
		t.Fatal("expected send error")
	}
	if fallback.used {
		t.Fatal("expected exec failure without fallback")
	}
}

func TestPostgresRepositoryHistoryReturnsErrorOnQueryFailure(t *testing.T) {
	backend := &groupMessageBackendStub{items: map[string][]groupMessageStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{queryGroupMessagesHistory: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &groupMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.History(context.Background(), "group_1")
	if err == nil {
		t.Fatal("expected history error")
	}
	if fallback.used {
		t.Fatal("expected query failure without fallback")
	}
}

func TestPostgresRepositoryMarkReadSupportMatchesCurrentPhase(t *testing.T) {
	backend := &groupMessageBackendStub{items: map[string][]groupMessageStubRecord{}, readStates: map[string]groupMessageReadState{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &groupMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	backend.items["group_1"] = []groupMessageStubRecord{
		{MessageID: "gm_1", GroupID: "group_1", Type: "text", Content: "first", SentAt: "2026-03-31T10:00:00Z"},
		{MessageID: "gm_2", GroupID: "group_1", Type: "text", Content: "second", SentAt: "2026-03-31T10:01:00Z"},
	}
	backend.items["group_2"] = []groupMessageStubRecord{
		{MessageID: "gm_3", GroupID: "group_2", Type: "text", Content: "other", SentAt: "2026-03-31T10:02:00Z"},
	}

	fallback.used = false
	if err := repo.MarkAsRead(context.Background(), "user_alice", MarkGroupMessagesReadRequest{GroupID: "group_1", UpToMessageID: "gm_1"}); err != nil {
		t.Fatalf("expected mark_as_read postgres behavior, got %v", err)
	}
	if fallback.used {
		t.Fatal("expected mark_as_read on postgres without fallback")
	}
	state := backend.readStates[groupMessageReadStateKey("group_1", "user_alice")]
	if state.MessageID != "gm_1" || state.SentAt != "2026-03-31T10:00:00Z" {
		t.Fatalf("unexpected mark_as_read state: %#v", state)
	}

	fallback.used = false
	if err := repo.MarkAsRead(context.Background(), "user_alice", MarkGroupMessagesReadRequest{GroupID: "group_1"}); err != nil {
		t.Fatalf("expected mark_as_read latest postgres behavior, got %v", err)
	}
	state = backend.readStates[groupMessageReadStateKey("group_1", "user_alice")]
	if state.MessageID != "gm_2" || state.SentAt != "2026-03-31T10:01:00Z" {
		t.Fatalf("unexpected mark_as_read latest state: %#v", state)
	}
	if _, ok := backend.readStates[groupMessageReadStateKey("group_2", "user_alice")]; ok {
		t.Fatalf("expected mark_as_read latest to stay within group_1, got %#v", backend.readStates)
	}

	if err := repo.MarkAsRead(context.Background(), "user_alice", MarkGroupMessagesReadRequest{GroupID: "group_1", UpToMessageID: "gm_missing"}); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound, got %v", err)
	}

	fallback.used = false
	if err := repo.MarkAllRead(context.Background(), "user_alice"); err != nil {
		t.Fatalf("expected mark_all_read postgres behavior, got %v", err)
	}
	if fallback.used {
		t.Fatal("expected mark_all_read on postgres without fallback")
	}
	state = backend.readStates[groupMessageReadStateKey("group_1", "user_alice")]
	if state.MessageID != "gm_2" || state.SentAt != "2026-03-31T10:01:00Z" {
		t.Fatalf("unexpected mark_all_read state for group_1: %#v", state)
	}
	state = backend.readStates[groupMessageReadStateKey("group_2", "user_alice")]
	if state.MessageID != "gm_3" || state.SentAt != "2026-03-31T10:02:00Z" {
		t.Fatalf("unexpected mark_all_read state for group_2: %#v", state)
	}
}
