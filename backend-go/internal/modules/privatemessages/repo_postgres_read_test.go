package privatemessages

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"sort"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type privateMessageNonQueryableDB struct{}

func (privateMessageNonQueryableDB) Ping(context.Context) error { return nil }
func (privateMessageNonQueryableDB) Close() error               { return nil }

type privateMessageStubRecord struct {
	MessageID        string
	ChatKey          string
	SenderID         string
	SenderNickname   string
	SenderAvatar     string
	Type             string
	Content          string
	FileID           string
	ShareCode        string
	ReplyToMessageID string
	EditedAt         string
	RecalledAt       string
	SentAt           string
}

type privateMessageReadState struct {
	MessageID string
	SentAt    string
}

type privateMessageBackendStub struct {
	items      map[string][]privateMessageStubRecord
	readStates map[string]map[string]privateMessageReadState
	execErrs   map[string]error
	queryErrs  map[string]error
}

func (b *privateMessageBackendStub) Ping(context.Context) error { return nil }
func (b *privateMessageBackendStub) Close() error               { return nil }

func normalizeStringArg(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func sentAtString(v any) string {
	if t, ok := v.(time.Time); ok {
		return t.Format(time.RFC3339Nano)
	}
	return v.(string)
}

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
			MessageID:        args[0].(string),
			ChatKey:          chatKey,
			SenderID:         args[2].(string),
			SenderNickname:   args[2].(string),
			Type:             args[3].(string),
			Content:          normalizeStringArg(args[4]),
			FileID:           normalizeStringArg(args[5]),
			ShareCode:        normalizeStringArg(args[6]),
			ReplyToMessageID: normalizeStringArg(args[7]),
			SentAt:           args[8].(string),
		}
		b.items[chatKey] = append(b.items[chatKey], record)
		return nil
	case queryUpdatePrivateMessage:
		messageID := args[0].(string)
		content := args[1].(string)
		for chatKey, items := range b.items {
			for i := range items {
				if items[i].MessageID == messageID && items[i].RecalledAt == "" {
					items[i].EditedAt = time.Now().UTC().Format(time.RFC3339Nano)
					items[i].Content = content
					b.items[chatKey] = items
					return nil
				}
			}
		}
		return nil
	case queryRecallPrivateMessage:
		messageID := args[0].(string)
		for chatKey, items := range b.items {
			for i := range items {
				if items[i].MessageID == messageID && items[i].RecalledAt == "" {
					items[i].RecalledAt = time.Now().UTC().Format(time.RFC3339Nano)
					b.items[chatKey] = items
					return nil
				}
			}
		}
		return nil
	case queryUpsertPrivateMessageReadState:
		chatKey := args[0].(string)
		userID := args[1].(string)
		messageID := args[2].(string)
		sentAt := sentAtString(args[3])
		if b.readStates[chatKey] == nil {
			b.readStates[chatKey] = map[string]privateMessageReadState{}
		}
		current, ok := b.readStates[chatKey][userID]
		if !ok || sentAt > current.SentAt || (sentAt == current.SentAt && messageID > current.MessageID) {
			b.readStates[chatKey][userID] = privateMessageReadState{MessageID: messageID, SentAt: sentAt}
		}
		return nil
	case queryMarkAllPrivateMessagesReadForChat:
		chatKey := args[0].(string)
		userID := args[1].(string)
		records := append([]privateMessageStubRecord(nil), b.items[chatKey]...)
		if len(records) == 0 {
			return nil
		}
		sort.Slice(records, func(i, j int) bool {
			if records[i].SentAt == records[j].SentAt {
				return records[i].MessageID > records[j].MessageID
			}
			return records[i].SentAt > records[j].SentAt
		})
		if b.readStates[chatKey] == nil {
			b.readStates[chatKey] = map[string]privateMessageReadState{}
		}
		latest := records[0]
		current, ok := b.readStates[chatKey][userID]
		if !ok || latest.SentAt > current.SentAt || (latest.SentAt == current.SentAt && latest.MessageID > current.MessageID) {
			b.readStates[chatKey][userID] = privateMessageReadState{MessageID: latest.MessageID, SentAt: latest.SentAt}
		}
		return nil
	case queryMarkAllPrivateMessagesRead:
		userID := args[0].(string)
		for chatKey, records := range b.items {
			ordered := append([]privateMessageStubRecord(nil), records...)
			if len(ordered) == 0 {
				continue
			}
			sort.Slice(ordered, func(i, j int) bool {
				if ordered[i].SentAt == ordered[j].SentAt {
					return ordered[i].MessageID > ordered[j].MessageID
				}
				return ordered[i].SentAt > ordered[j].SentAt
			})
			latest := ordered[0]
			if b.readStates[chatKey] == nil {
				b.readStates[chatKey] = map[string]privateMessageReadState{}
			}
			current, ok := b.readStates[chatKey][userID]
			if !ok || latest.SentAt > current.SentAt || (latest.SentAt == current.SentAt && latest.MessageID > current.MessageID) {
				b.readStates[chatKey][userID] = privateMessageReadState{MessageID: latest.MessageID, SentAt: latest.SentAt}
			}
		}
		return nil
	default:
		return nil
	}
}

func (b *privateMessageBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	switch query {
	case queryPrivateMessageReadStateByMessage:
		chatKey := args[0].(string)
		messageID := args[1].(string)
		for _, record := range b.items[chatKey] {
			if record.MessageID == messageID {
				return testdb.RowValues{Values: []any{record.SentAt, record.MessageID}}
			}
		}
	case queryGetPrivateMessageMeta:
		messageID := args[0].(string)
		for _, items := range b.items {
			for _, record := range items {
				if record.MessageID == messageID {
					return testdb.RowValues{Values: []any{record.ChatKey, record.SenderID, record.Type, record.RecalledAt != "", record.SentAt}}
				}
			}
		}
	case queryUpdatePrivateMessage:
		// 模拟 is_recalled = FALSE 守卫 + RETURNING：未撤回的消息更新并返回
		// 一行，已撤回的 0 行（ErrNoRows）。
		messageID := args[0].(string)
		content := args[1].(string)
		for _, items := range b.items {
			for i := range items {
				if items[i].MessageID == messageID && items[i].RecalledAt == "" {
					items[i].Content = content
					items[i].EditedAt = time.Now().UTC().Format(time.RFC3339Nano)
					return testdb.RowValues{Values: []any{messageID}}
				}
			}
		}
		return testdb.RowValues{Err: pgx.ErrNoRows}
	case queryRecallPrivateMessage:
		messageID := args[0].(string)
		for _, items := range b.items {
			for i := range items {
				if items[i].MessageID == messageID && items[i].RecalledAt == "" {
					items[i].RecalledAt = time.Now().UTC().Format(time.RFC3339Nano)
					return testdb.RowValues{Values: []any{messageID}}
				}
			}
		}
		return testdb.RowValues{Err: pgx.ErrNoRows}
	case queryUnreadPrivateMessageCount:
		// 口径与 SQL 一致：对方发来、晚于已读游标、未撤回；deletedAfter 是
		// 可见性下界（仅统计严格晚于它的消息）。
		chatKey := args[0].(string)
		viewerID := args[1].(string)
		var deletedAfter time.Time
		if v, ok := args[2].(time.Time); ok {
			deletedAfter = v
		}
		state := privateMessageReadState{}
		if b.readStates != nil && b.readStates[chatKey] != nil {
			state = b.readStates[chatKey][viewerID]
		}
		count := 0
		for _, record := range b.items[chatKey] {
			if record.SenderID == viewerID || record.RecalledAt != "" {
				continue
			}
			if (!deletedAfter.IsZero()) && record.SentAt <= deletedAfter.Format(time.RFC3339Nano) {
				continue
			}
			if state.SentAt == "" || record.SentAt > state.SentAt {
				count++
			}
		}
		return testdb.RowValues{Values: []any{count}}
	}
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}

func (b *privateMessageBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if query == queryPrivateMessagesHistory {
		chatKey := args[0].(string)
		readerUserID := args[1].(string)
		readStateUserID := args[2].(string)
		cursorSentAt, _ := args[3].(string)
		cursorMessageID := args[4].(string)
		limit := args[5].(int)
		records := append([]privateMessageStubRecord(nil), b.items[chatKey]...)
		sort.Slice(records, func(i, j int) bool {
			if records[i].SentAt == records[j].SentAt {
				return records[i].MessageID > records[j].MessageID
			}
			return records[i].SentAt > records[j].SentAt
		})
		if cursorSentAt != "" {
			filtered := make([]privateMessageStubRecord, 0, len(records))
			for _, record := range records {
				if record.SentAt < cursorSentAt || (record.SentAt == cursorSentAt && record.MessageID < cursorMessageID) {
					filtered = append(filtered, record)
				}
			}
			records = filtered
		}
		if limit > 0 && len(records) > limit {
			records = records[:limit]
		}
		state := privateMessageReadState{}
		if b.readStates != nil && b.readStates[chatKey] != nil {
			state = b.readStates[chatKey][readStateUserID]
		}
		rows := make([][]any, 0, len(records))
		for _, record := range records {
			isRead := record.SenderID != readerUserID
			if record.SenderID == readerUserID {
				if state.SentAt == "" {
					isRead = false
				} else if record.SentAt < state.SentAt {
					isRead = true
				} else if record.SentAt > state.SentAt {
					isRead = false
				} else if state.MessageID != "" {
					isRead = record.MessageID == state.MessageID
				}
			}
			reply := privateMessageStubRecord{}
			if record.ReplyToMessageID != "" {
				for _, candidate := range b.items[chatKey] {
					if candidate.MessageID == record.ReplyToMessageID {
						reply = candidate
						break
					}
				}
			}
			rows = append(rows, []any{
				record.MessageID,
				record.Type,
				record.Content,
				record.FileID,
				record.ShareCode,
				record.SenderID,
				record.SenderNickname,
				record.SenderAvatar,
				reply.MessageID,
				reply.Type,
				reply.Content,
				reply.ShareCode,
				reply.SenderID,
				reply.SenderNickname,
				reply.SenderAvatar,
				reply.SentAt,
				record.SentAt,
				record.EditedAt,
				record.RecalledAt,
				isRead,
			})
		}
		return &testdb.RowsValues{Rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryFailsClosedWhenQueryableUnavailable(t *testing.T) {
	repo := NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     privateMessageNonQueryableDB{},
	})

	if _, err := repo.Send(context.Background(), "user_alice", "user_bob", SendPrivateMessageRequest{Type: "text", Content: "hello"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for send, got %v", err)
	}
	if _, err := repo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for history, got %v", err)
	}
	if _, err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_1", Content: "updated"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for update, got %v", err)
	}
	if _, err := repo.Recall(context.Background(), "user_alice", RecallPrivateMessageRequest{MessageID: "pm_1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for recall, got %v", err)
	}
	if _, err := repo.MarkAsRead(context.Background(), "user_alice", MarkPrivateMessagesReadRequest{TargetUserID: "user_bob"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for mark_as_read, got %v", err)
	}
	if err := repo.MarkAllRead(context.Background(), "user_alice"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for mark_all_read, got %v", err)
	}
}

func TestPostgresRepositorySendHistoryRecentAndLast(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, readStates: map[string]map[string]privateMessageReadState{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	sent, err := repo.Send(context.Background(), "user_alice", "user_bob", SendPrivateMessageRequest{Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	history, err := repo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob"})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history.Items) != 1 || history.Items[0].MessageID != sent.MessageID || *history.Items[0].Content != "hello" {
		t.Fatalf("unexpected history: %#v", history)
	}
	recent, err := repo.Recent(context.Background(), "user_alice", "user_bob", 1, nil)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent.Items) != 1 || recent.Items[0].MessageID != sent.MessageID {
		t.Fatalf("unexpected recent: %#v", recent)
	}
	last, err := repo.Last(context.Background(), "user_alice", "user_bob", nil)
	if err != nil {
		t.Fatalf("last: %v", err)
	}
	if last == nil || last.MessageID != sent.MessageID {
		t.Fatalf("unexpected last: %#v", last)
	}
}

func TestPostgresRepositoryHistorySupportsMediaReplyAndPagination(t *testing.T) {
	chatKey := conversationKey("user_alice", "user_bob")
	backend := &privateMessageBackendStub{
		items: map[string][]privateMessageStubRecord{
			chatKey: {
				{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_alice", SenderNickname: "alice", Type: "text", Content: "hello", SentAt: "2026-03-30T10:00:00Z"},
				{MessageID: "pm_2", ChatKey: chatKey, SenderID: "user_bob", SenderNickname: "bob", Type: "image", FileID: "file_1", ReplyToMessageID: "pm_1", SentAt: "2026-03-30T10:01:00Z"},
				{MessageID: "pm_3", ChatKey: chatKey, SenderID: "user_alice", SenderNickname: "alice", Type: "share", ShareCode: "share_1", SentAt: "2026-03-30T10:02:00Z"},
			},
		},
		readStates: map[string]map[string]privateMessageReadState{},
		execErrs:   map[string]error{},
		queryErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	history, err := repo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob", Pagination: &PaginationParams{Limit: 2}})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history.Items) != 2 || history.NextCursor == nil {
		t.Fatalf("unexpected paginated history: %#v", history)
	}
	if history.Items[1].Media == nil || history.Items[1].Media.URL != "file_1" {
		t.Fatalf("expected media payload, got %#v", history.Items[1])
	}
	if history.Items[1].ReplyTo == nil || history.Items[1].ReplyTo.MessageID != "pm_1" {
		t.Fatalf("expected reply_to payload, got %#v", history.Items[1])
	}

	next, err := repo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob", Pagination: &PaginationParams{Cursor: *history.NextCursor, Limit: 2}})
	if err != nil {
		t.Fatalf("history next page: %v", err)
	}
	if len(next.Items) != 1 || next.Items[0].MessageID != "pm_1" {
		t.Fatalf("unexpected next page: %#v", next)
	}
}

func TestPostgresRepositoryHistoryReturnsEmptyWithoutFallback(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, readStates: map[string]map[string]privateMessageReadState{}, execErrs: map[string]error{}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	history, err := repo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob"})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history.Items) != 0 {
		t.Fatalf("expected empty history, got %#v", history)
	}
}

func TestPostgresRepositorySendReturnsErrorOnExecFailure(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, readStates: map[string]map[string]privateMessageReadState{}, execErrs: map[string]error{queryInsertPrivateMessage: errors.New("boom")}, queryErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Send(context.Background(), "user_alice", "user_bob", SendPrivateMessageRequest{Type: "text", Content: "hello"})
	if err == nil {
		t.Fatal("expected send error")
	}
}

func TestPostgresRepositoryHistoryReturnsErrorOnQueryFailure(t *testing.T) {
	backend := &privateMessageBackendStub{items: map[string][]privateMessageStubRecord{}, readStates: map[string]map[string]privateMessageReadState{}, execErrs: map[string]error{}, queryErrs: map[string]error{queryPrivateMessagesHistory: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob"})
	if err == nil {
		t.Fatal("expected history error")
	}
}

func TestPostgresRepositoryMutationSupportMatchesCurrentPhase(t *testing.T) {
	chatKey := conversationKey("user_alice", "user_bob")
	backend := &privateMessageBackendStub{
		items: map[string][]privateMessageStubRecord{
			chatKey: {
				{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_alice", SenderNickname: "alice", Type: "text", Content: "first", SentAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)},
				{MessageID: "pm_2", ChatKey: chatKey, SenderID: "user_alice", SenderNickname: "alice", Type: "text", Content: "second", SentAt: time.Now().UTC().Format(time.RFC3339Nano)},
			},
		},
		readStates: map[string]map[string]privateMessageReadState{},
		execErrs:   map[string]error{},
		queryErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	if _, err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_1", Content: "updated"}); err != nil {
		t.Fatalf("expected update to execute on postgres, got %v", err)
	}
	if backend.items[chatKey][0].Content != "updated" || backend.items[chatKey][0].EditedAt == "" {
		t.Fatalf("expected update to persist content/edit timestamp, got %#v", backend.items[chatKey][0])
	}

	if _, err := repo.Recall(context.Background(), "user_alice", RecallPrivateMessageRequest{MessageID: "pm_2"}); err != nil {
		t.Fatalf("expected recall to execute on postgres, got %v", err)
	}
	if backend.items[chatKey][1].RecalledAt == "" {
		t.Fatalf("expected recall timestamp, got %#v", backend.items[chatKey][1])
	}

	if _, err := repo.MarkAsRead(context.Background(), "user_alice", MarkPrivateMessagesReadRequest{TargetUserID: "user_bob", UpToMessageID: "pm_1"}); err != nil {
		t.Fatalf("expected mark_as_read postgres behavior, got %v", err)
	}
	state := backend.readStates[chatKey]["user_alice"]
	if state.MessageID != "pm_1" {
		t.Fatalf("expected mark_as_read to persist cursor, got %#v", state)
	}

	if _, err := repo.MarkAsRead(context.Background(), "user_alice", MarkPrivateMessagesReadRequest{TargetUserID: "user_bob", UpToMessageID: "pm_missing"}); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound, got %v", err)
	}

	if err := repo.MarkAllRead(context.Background(), "user_alice"); err != nil {
		t.Fatalf("expected mark_all_read postgres behavior, got %v", err)
	}
	state = backend.readStates[chatKey]["user_alice"]
	if state.MessageID != "pm_2" {
		t.Fatalf("expected mark_all_read to advance cursor, got %#v", state)
	}
}

func TestPostgresRepositoryHistoryReflectsReadStateSemantics(t *testing.T) {
	chatKey := conversationKey("user_alice", "user_bob")
	backend := &privateMessageBackendStub{
		items: map[string][]privateMessageStubRecord{
			chatKey: {
				{MessageID: "pm_3", ChatKey: chatKey, SenderID: "user_bob", SenderNickname: "bob", Type: "text", Content: "reply", SentAt: "2026-03-30T10:02:00Z"},
				{MessageID: "pm_2", ChatKey: chatKey, SenderID: "user_alice", SenderNickname: "alice", Type: "text", Content: "second", SentAt: "2026-03-30T10:01:00Z"},
				{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_alice", SenderNickname: "alice", Type: "text", Content: "first", SentAt: "2026-03-30T10:00:00Z"},
			},
		},
		readStates: map[string]map[string]privateMessageReadState{
			chatKey: {
				"user_bob": {MessageID: "pm_1", SentAt: "2026-03-30T10:00:00Z"},
			},
		},
		execErrs:  map[string]error{},
		queryErrs: map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	history, err := repo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob"})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history.Items) != 3 {
		t.Fatalf("expected 3 messages, got %#v", history)
	}
	if !history.Items[0].IsRead {
		t.Fatal("expected incoming message to always be read for current viewer")
	}
	if history.Items[1].IsRead {
		t.Fatal("expected newer outgoing message to remain unread before peer cursor")
	}
	if !history.Items[2].IsRead {
		t.Fatal("expected outgoing message at peer cursor to be read")
	}
}

func TestPostgresRepositoryUnreadCountUsesViewerReadState(t *testing.T) {
	chatKey := conversationKey("user_alice", "user_bob")
	backend := &privateMessageBackendStub{
		items: map[string][]privateMessageStubRecord{
			chatKey: {
				{MessageID: "pm_3", ChatKey: chatKey, SenderID: "user_bob", SenderNickname: "bob", Type: "text", Content: "new", SentAt: "2026-03-30T10:02:00Z"},
				{MessageID: "pm_2", ChatKey: chatKey, SenderID: "user_bob", SenderNickname: "bob", Type: "text", Content: "old", SentAt: "2026-03-30T10:01:00Z"},
				{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_alice", SenderNickname: "alice", Type: "text", Content: "mine", SentAt: "2026-03-30T10:00:00Z"},
			},
		},
		readStates: map[string]map[string]privateMessageReadState{
			chatKey: {
				"user_alice": {MessageID: "pm_2", SentAt: "2026-03-30T10:01:00Z"},
			},
		},
		execErrs:  map[string]error{},
		queryErrs: map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	count, err := repo.UnreadCount(context.Background(), "user_alice", "user_bob", nil)
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unread count 1, got %d", count)
	}
}
