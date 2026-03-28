package private_messages

import (
	"context"
	"errors"
	"sort"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type privateMessageRowValues struct {
	values []any
	err    error
}

func (r privateMessageRowValues) Scan(dest ...any) error {
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

type privateMessageRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *privateMessageRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *privateMessageRowsValues) Scan(dest ...any) error {
	row := privateMessageRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *privateMessageRowsValues) Close() error { return nil }
func (r *privateMessageRowsValues) Err() error   { return r.err }

type privateMessageFallbackRepo struct{ used bool }

func (r *privateMessageFallbackRepo) markUsed() { r.used = true }
func (r *privateMessageFallbackRepo) Send(context.Context, string, string, SendPrivateMessageRequest) (MessageSent, error) {
	r.markUsed()
	return MessageSent{MessageID: "fallback", SentAt: "fallback"}, nil
}
func (r *privateMessageFallbackRepo) History(context.Context, string, string) (PaginatedMessages, error) {
	r.markUsed()
	return PaginatedMessages{Items: []PrivateMessageItem{{MessageID: "fallback"}}}, nil
}
func (r *privateMessageFallbackRepo) Recent(context.Context, string, string, int) (PaginatedMessages, error) {
	r.markUsed()
	return PaginatedMessages{}, nil
}
func (r *privateMessageFallbackRepo) Last(context.Context, string, string) (*PrivateMessageItem, error) {
	r.markUsed()
	return &PrivateMessageItem{MessageID: "fallback"}, nil
}
func (r *privateMessageFallbackRepo) Update(context.Context, string, UpdatePrivateMessageRequest) error {
	r.markUsed()
	return nil
}
func (r *privateMessageFallbackRepo) Recall(context.Context, string, RecallPrivateMessageRequest) error {
	r.markUsed()
	return nil
}
func (r *privateMessageFallbackRepo) MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) error {
	r.markUsed()
	return nil
}
func (r *privateMessageFallbackRepo) MarkAllRead(context.Context, string) error {
	r.markUsed()
	return nil
}

type privateMessageStubRecord struct {
	MessageID string
	ChatKey   string
	SenderID  string
	Type      string
	Content   string
	ShareCode string
	SentAt    string
}

type privateMessageBackendStub struct {
	items     map[string][]privateMessageStubRecord
	execErrs  map[string]error
	queryErrs map[string]error
}

func (b *privateMessageBackendStub) Ping(context.Context) error { return nil }
func (b *privateMessageBackendStub) Close() error               { return nil }
func (b *privateMessageBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	switch query {
	case queryInsertPrivateChat:
		return nil
	case queryInsertPrivateMessage:
		chatKey := args[1].(string)
		record := privateMessageStubRecord{
			MessageID: args[0].(string),
			ChatKey:   chatKey,
			SenderID:  args[2].(string),
			Type:      args[3].(string),
			Content:   args[4].(string),
			ShareCode: args[5].(string),
			SentAt:    args[6].(string),
		}
		b.items[chatKey] = append(b.items[chatKey], record)
		return nil
	default:
		return nil
	}
}
func (b *privateMessageBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return privateMessageRowValues{err: persistence.ErrNotImplemented}
}
func (b *privateMessageBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if query == queryPrivateMessagesHistory {
		chatKey := args[0].(string)
		records := append([]privateMessageStubRecord(nil), b.items[chatKey]...)
		sort.Slice(records, func(i, j int) bool {
			if records[i].SentAt == records[j].SentAt {
				return records[i].MessageID > records[j].MessageID
			}
			return records[i].SentAt > records[j].SentAt
		})
		rows := make([][]any, 0, len(records))
		for _, record := range records {
			rows = append(rows, []any{record.MessageID, record.Type, record.Content, record.ShareCode, record.SenderID, record.SentAt})
		}
		return &privateMessageRowsValues{rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositorySendHistoryRecentAndLast(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	sent, err := repo.Send(context.Background(), "user_alice", "user_bob", SendPrivateMessageRequest{Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	history, err := repo.History(context.Background(), "user_alice", "user_bob")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history.Items) != 1 || history.Items[0].MessageID != sent.MessageID || history.Items[0].Content != "hello" {
		t.Fatalf("unexpected history: %#v", history)
	}
	recent, err := repo.Recent(context.Background(), "user_alice", "user_bob", 1)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent.Items) != 1 || recent.Items[0].MessageID != sent.MessageID {
		t.Fatalf("unexpected recent: %#v", recent)
	}
	last, err := repo.Last(context.Background(), "user_alice", "user_bob")
	if err != nil {
		t.Fatalf("last: %v", err)
	}
	if last == nil || last.MessageID != sent.MessageID {
		t.Fatalf("unexpected last: %#v", last)
	}
	if fallback.used {
		t.Fatal("expected postgres path without fallback")
	}
}

func TestPostgresRepositoryHistoryReturnsEmptyWithoutFallback(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	history, err := repo.History(context.Background(), "user_alice", "user_bob")
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
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, execErrs: map[string]error{queryInsertPrivateMessage: errors.New("boom")}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.Send(context.Background(), "user_alice", "user_bob", SendPrivateMessageRequest{Type: "text", Content: "hello"})
	if err == nil {
		t.Fatal("expected send error")
	}
	if fallback.used {
		t.Fatal("expected send exec failure without fallback")
	}
}

func TestPostgresRepositoryHistoryReturnsErrorOnQueryFailure(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{queryPrivateMessagesHistory: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.History(context.Background(), "user_alice", "user_bob")
	if err == nil {
		t.Fatal("expected history error")
	}
	if fallback.used {
		t.Fatal("expected query failure without fallback")
	}
}

func TestPostgresRepositoryMutationSupportMatchesCurrentPhase(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &privateMessageFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	if err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_1", Content: "updated"}); err != nil {
		t.Fatalf("expected update to execute on postgres, got %v", err)
	}
	if err := repo.Recall(context.Background(), "user_alice", RecallPrivateMessageRequest{MessageID: "pm_1"}); err != nil {
		t.Fatalf("expected recall to execute on postgres, got %v", err)
	}
	if fallback.used {
		t.Fatal("expected update/recall to execute on postgres without fallback")
	}

	fallback.used = false
	if err := repo.MarkAsRead(context.Background(), "user_alice", MarkPrivateMessagesReadRequest{TargetUserID: "user_bob", UpToMessageID: "pm_1"}); err != nil {
		t.Fatalf("expected mark_as_read fallback behavior, got %v", err)
	}
	if !fallback.used {
		t.Fatal("expected mark_as_read to use fallback")
	}

	fallback.used = false
	if err := repo.MarkAllRead(context.Background(), "user_alice"); err != nil {
		t.Fatalf("expected mark_all_read fallback behavior, got %v", err)
	}
	if !fallback.used {
		t.Fatal("expected mark_all_read to use fallback")
	}
}
