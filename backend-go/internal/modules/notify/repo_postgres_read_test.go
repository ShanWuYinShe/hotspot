package notify

import (
	"context"
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"sort"
	"strconv"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type notifyNonQueryableDB struct{}

func (notifyNonQueryableDB) Ping(context.Context) error { return nil }
func (notifyNonQueryableDB) Close() error               { return nil }

type notifyBackendStub struct {
	items     map[string][]NotifyItem
	execErrs  map[string]error
	queryErrs map[string]error
}

func newNotifyBackendStub() *notifyBackendStub {
	return &notifyBackendStub{items: make(map[string][]NotifyItem), execErrs: make(map[string]error), queryErrs: make(map[string]error)}
}

func (b *notifyBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	userID := args[0].(string)
	switch query {
	case queryInsertNotify:
		item := NotifyItem{
			NotifyID:  args[0].(string),
			Type:      args[2].(string),
			IsRead:    args[3].(bool),
			CreatedAt: args[4].(string),
		}
		if payloadRaw := args[5].(string); payloadRaw != "" {
			_ = json.Unmarshal([]byte(payloadRaw), &item.Payload)
		}
		b.items[args[1].(string)] = append(b.items[args[1].(string)], item)
	case queryMarkNotifysRead:
		ids := args[1].([]string)
		for i := range b.items[userID] {
			for _, id := range ids {
				if b.items[userID][i].NotifyID == id {
					b.items[userID][i].IsRead = true
				}
			}
		}
	case queryMarkAllNotifysRead:
		for i := range b.items[userID] {
			b.items[userID][i].IsRead = true
		}
	}
	return nil
}

func (b *notifyBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}

func (b *notifyBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	switch query {
	case queryListNotifys:
		userID := args[0].(string)
		typeFilter := ""
		if args[1] != nil {
			typeFilter = args[1].(string)
		}
		var isReadFilter *bool
		if args[2] != nil {
			isReadFilter = args[2].(*bool)
		}
		cursorCreatedAt := ""
		if args[3] != nil {
			cursorCreatedAt = args[3].(string)
		}
		cursorNotifyID := args[4].(string)
		limit := args[5].(int)
		items := append([]NotifyItem(nil), b.items[userID]...)
		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].NotifyID > items[j].NotifyID
			}
			return items[i].CreatedAt > items[j].CreatedAt
		})
		rows := make([][]any, 0)
		for _, item := range items {
			if typeFilter != "" && item.Type != typeFilter {
				continue
			}
			if isReadFilter != nil && item.IsRead != *isReadFilter {
				continue
			}
			if cursorCreatedAt != "" {
				if !(item.CreatedAt < cursorCreatedAt || (item.CreatedAt == cursorCreatedAt && item.NotifyID < cursorNotifyID)) {
					continue
				}
			}
			payloadJSON, _ := json.Marshal(item.Payload)
			rows = append(rows, []any{item.NotifyID, item.Type, item.IsRead, item.CreatedAt, string(payloadJSON)})
			if len(rows) >= limit {
				break
			}
		}
		return &testdb.RowsValues{Rows: rows}, nil
	case queryUnreadNotifyCounts:
		userID := args[0].(string)
		counts := map[string]int{}
		for _, item := range b.items[userID] {
			if item.IsRead {
				continue
			}
			counts[item.Type]++
		}
		rows := make([][]any, 0, len(counts))
		for notifyType, count := range counts {
			rows = append(rows, []any{notifyType, strconv.Itoa(count)})
		}
		return &testdb.RowsValues{Rows: rows}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func TestPersistenceRepositoryWithPostgresDSNDoesNotFallbackSilently(t *testing.T) {
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     notifyNonQueryableDB{},
	}
	repo := NewPostgresRepository(factory)

	if err := repo.Append(context.Background(), "user_alice", NotifyItem{Type: "new_follower"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for append, got %v", err)
	}
	if _, err := repo.List(context.Background(), "user_alice", NotifyListRequest{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented instead of in-memory fallback, got %v", err)
	}
	if _, err := repo.UnreadCount(context.Background(), "user_alice"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for unread count, got %v", err)
	}
	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{"notif_1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for mark_as_read, got %v", err)
	}
	if err := repo.MarkAllRead(context.Background(), "user_alice"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for mark_all_read, got %v", err)
	}
}

func TestPostgresRepositoryListSupportsFilterPaginationAndUnreadCount(t *testing.T) {
	backend := newNotifyBackendStub()
	backend.items["user_alice"] = []NotifyItem{
		{NotifyID: "notif_3", Type: "system_announcement", IsRead: false, CreatedAt: "2026-03-20T00:00:00Z", Payload: map[string]any{"type": "system_announcement", "announcement_id": "ann_1"}},
		{NotifyID: "notif_2", Type: "new_follower", IsRead: false, CreatedAt: "2026-03-20T00:00:00Z", Payload: map[string]any{"type": "new_follower"}},
		{NotifyID: "notif_1", Type: "group_invite_received", IsRead: true, CreatedAt: "2026-03-19T00:00:00Z", Payload: map[string]any{"type": "group_invite_received"}},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	page1, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Pagination: &PaginationParams{Limit: 2}})
	if err != nil || len(page1.Items) != 2 || page1.Items[0].NotifyID != "notif_3" || page1.Items[1].NotifyID != "notif_2" || page1.NextCursor == nil {
		t.Fatalf("unexpected page1: %#v err=%v", page1, err)
	}
	if page1.Items[0].Payload["announcement_id"] != "ann_1" {
		t.Fatalf("unexpected payload: %#v", page1.Items[0])
	}

	page2, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Pagination: &PaginationParams{Limit: 2, Cursor: *page1.NextCursor}})
	if err != nil || len(page2.Items) != 1 || page2.Items[0].NotifyID != "notif_1" || page2.NextCursor != nil {
		t.Fatalf("unexpected page2: %#v err=%v", page2, err)
	}

	filtered, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Type: "new_follower", Pagination: &PaginationParams{Limit: 20}})
	if err != nil || len(filtered.Items) != 1 || filtered.Items[0].NotifyID != "notif_2" {
		t.Fatalf("unexpected filtered list: %#v err=%v", filtered, err)
	}

	count, err := repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 2 || count.ByType.System != 1 || count.ByType.Social != 1 {
		t.Fatalf("unexpected unread count: %#v err=%v", count, err)
	}

	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{"notif_3"}); err != nil {
		t.Fatalf("mark as read: %v", err)
	}
	count, err = repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 1 || count.ByType.System != 0 {
		t.Fatalf("unexpected unread count after mark: %#v err=%v", count, err)
	}

	if err := repo.MarkAllRead(context.Background(), "user_alice"); err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	count, err = repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 0 {
		t.Fatalf("unexpected unread count after mark all: %#v err=%v", count, err)
	}
}

func TestPostgresRepositoryAppendWritesAndAffectsUnreadCount(t *testing.T) {
	backend := newNotifyBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	if err := repo.Append(context.Background(), "user_alice", NotifyItem{Type: "new_follower", Payload: map[string]any{"actor": map[string]any{"user_id": "user_bob"}}}); err != nil {
		t.Fatalf("append: %v", err)
	}
	list, err := repo.List(context.Background(), "user_alice", NotifyListRequest{})
	if err != nil || len(list.Items) != 1 || list.Items[0].Type != "new_follower" {
		t.Fatalf("unexpected list after append: %#v err=%v", list, err)
	}
	count, err := repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 1 || count.ByType.Social != 1 {
		t.Fatalf("unexpected unread count after append: %#v err=%v", count, err)
	}
}

func TestPostgresRepositoryReturnsEmptyListWithoutFallback(t *testing.T) {
	backend := newNotifyBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	list, err := repo.List(context.Background(), "user_alice", NotifyListRequest{})
	if err != nil || len(list.Items) != 0 {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
	count, err := repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 0 {
		t.Fatalf("unexpected count: %#v err=%v", count, err)
	}
}

func TestPostgresRepositoryMarkAsReadReturnsErrorOnExecFailure(t *testing.T) {
	backend := newNotifyBackendStub()
	backend.execErrs[queryMarkNotifysRead] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{"notif_1"}); err == nil {
		t.Fatal("expected mark as read error")
	}
}

func TestPostgresRepositoryListReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newNotifyBackendStub()
	backend.queryErrs[queryListNotifys] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.List(context.Background(), "user_alice", NotifyListRequest{})
	if err == nil {
		t.Fatal("expected list error")
	}
}

func TestPostgresRepositoryUnreadCountReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newNotifyBackendStub()
	backend.queryErrs[queryUnreadNotifyCounts] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.UnreadCount(context.Background(), "user_alice")
	if err == nil {
		t.Fatal("expected unread count error")
	}
}
