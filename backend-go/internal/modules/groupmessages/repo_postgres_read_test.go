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

type groupMessageStubRecord struct {
	MessageID string
	GroupID   string
	Type      string
	Content   string
	SentAt    string
}

type groupMessageBackendStub struct {
	items     map[string][]groupMessageStubRecord
	execErrs  map[string]error
	queryErrs map[string]error
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
	}
	return nil
}
func (b *groupMessageBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
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
	backend := &groupMessageBackendStub{items: map[string][]groupMessageStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
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
	backend := &groupMessageBackendStub{items: map[string][]groupMessageStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
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
