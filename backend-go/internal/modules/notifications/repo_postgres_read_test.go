package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type notificationRowValues struct {
	values []any
	err    error
}

func (r notificationRowValues) Scan(dest ...any) error {
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

type notificationRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *notificationRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *notificationRowsValues) Scan(dest ...any) error {
	row := notificationRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *notificationRowsValues) Close() error { return nil }
func (r *notificationRowsValues) Err() error   { return r.err }

type notificationFallbackRepo struct{ used bool }

type notificationNonQueryableDB struct{}

func (notificationNonQueryableDB) Ping(context.Context) error { return nil }
func (notificationNonQueryableDB) Close() error               { return nil }

func (r *notificationFallbackRepo) List(context.Context, string, NotificationListRequest) (PaginatedNotifications, error) {
	r.used = true
	return PaginatedNotifications{}, nil
}
func (r *notificationFallbackRepo) UnreadCount(context.Context, string) (UnreadCountResult, error) {
	r.used = true
	return UnreadCountResult{}, nil
}
func (r *notificationFallbackRepo) MarkAsRead(context.Context, string, []string) error {
	r.used = true
	return nil
}
func (r *notificationFallbackRepo) MarkAllRead(context.Context, string) error {
	r.used = true
	return nil
}

type notificationBackendStub struct {
	items     map[string][]NotificationItem
	execErrs  map[string]error
	queryErrs map[string]error
}

func newNotificationBackendStub() *notificationBackendStub {
	return &notificationBackendStub{items: make(map[string][]NotificationItem), execErrs: make(map[string]error), queryErrs: make(map[string]error)}
}

func (b *notificationBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	userID := args[0].(string)
	switch query {
	case queryMarkNotificationsRead:
		ids := args[1].([]string)
		for i := range b.items[userID] {
			for _, id := range ids {
				if b.items[userID][i].NotificationID == id {
					b.items[userID][i].IsRead = true
				}
			}
		}
	case queryMarkAllNotificationsRead:
		for i := range b.items[userID] {
			b.items[userID][i].IsRead = true
		}
	}
	return nil
}

func (b *notificationBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return notificationRowValues{err: persistence.ErrNotImplemented}
}

func (b *notificationBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	switch query {
	case queryListNotifications:
		userID := args[0].(string)
		typeFilter := args[1].(string)
		cursorCreatedAt := args[2].(string)
		cursorNotificationID := args[3].(string)
		limit := args[4].(int)
		items := append([]NotificationItem(nil), b.items[userID]...)
		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].NotificationID > items[j].NotificationID
			}
			return items[i].CreatedAt > items[j].CreatedAt
		})
		rows := make([][]any, 0)
		for _, item := range items {
			if typeFilter != "" && item.Type != typeFilter {
				continue
			}
			if cursorCreatedAt != "" {
				if !(item.CreatedAt < cursorCreatedAt || (item.CreatedAt == cursorCreatedAt && item.NotificationID < cursorNotificationID)) {
					continue
				}
			}
			payloadJSON, _ := json.Marshal(item.Payload)
			rows = append(rows, []any{item.NotificationID, item.Type, item.IsRead, item.CreatedAt, string(payloadJSON)})
			if len(rows) >= limit {
				break
			}
		}
		return &notificationRowsValues{rows: rows}, nil
	case queryUnreadNotificationCounts:
		userID := args[0].(string)
		counts := map[string]int{}
		for _, item := range b.items[userID] {
			if item.IsRead {
				continue
			}
			counts[item.Type]++
		}
		rows := make([][]any, 0, len(counts))
		for notificationType, count := range counts {
			rows = append(rows, []any{notificationType, strconv.Itoa(count)})
		}
		return &notificationRowsValues{rows: rows}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func TestNewPersistenceRepositoryUsesPostgresWhenDSNConfigured(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	repo := NewPersistenceRepository(factory)
	if _, ok := repo.impl.(*PostgresRepository); !ok {
		t.Fatalf("expected postgres impl, got %T", repo.impl)
	}
}

func TestPersistenceRepositoryUsesMemoryOnlyWithoutPostgresDSN(t *testing.T) {
	repo := NewPersistenceRepository(nil)
	if _, err := repo.List(context.Background(), "user_alice", NotificationListRequest{}); err != nil {
		t.Fatalf("expected in-memory list without postgres dsn, got %v", err)
	}
}

func TestPersistenceRepositoryWithPostgresDSNDoesNotFallbackSilently(t *testing.T) {
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     notificationNonQueryableDB{},
	}
	repo := &PersistenceRepository{impl: NewPostgresRepository(factory, nil)}

	if _, err := repo.List(context.Background(), "user_alice", NotificationListRequest{}); !errors.Is(err, persistence.ErrNotImplemented) {
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
	backend := newNotificationBackendStub()
	backend.items["user_alice"] = []NotificationItem{
		{NotificationID: "notif_3", Type: "system_announcement", IsRead: false, CreatedAt: "2026-03-20T00:00:00Z", Payload: map[string]any{"type": "system_announcement", "announcement_id": "ann_1"}},
		{NotificationID: "notif_2", Type: "new_follower", IsRead: false, CreatedAt: "2026-03-20T00:00:00Z", Payload: map[string]any{"type": "new_follower"}},
		{NotificationID: "notif_1", Type: "group_invite_received", IsRead: true, CreatedAt: "2026-03-19T00:00:00Z", Payload: map[string]any{"type": "group_invite_received"}},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &notificationFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	page1, err := repo.List(context.Background(), "user_alice", NotificationListRequest{Pagination: &PaginationParams{Limit: 2}})
	if err != nil || len(page1.Items) != 2 || page1.Items[0].NotificationID != "notif_3" || page1.Items[1].NotificationID != "notif_2" || page1.NextCursor == "" {
		t.Fatalf("unexpected page1: %#v err=%v", page1, err)
	}
	if page1.Items[0].Payload["announcement_id"] != "ann_1" {
		t.Fatalf("unexpected payload: %#v", page1.Items[0])
	}

	page2, err := repo.List(context.Background(), "user_alice", NotificationListRequest{Pagination: &PaginationParams{Limit: 2, Cursor: page1.NextCursor}})
	if err != nil || len(page2.Items) != 1 || page2.Items[0].NotificationID != "notif_1" || page2.NextCursor != "" {
		t.Fatalf("unexpected page2: %#v err=%v", page2, err)
	}

	filtered, err := repo.List(context.Background(), "user_alice", NotificationListRequest{Type: "new_follower", Pagination: &PaginationParams{Limit: 20}})
	if err != nil || len(filtered.Items) != 1 || filtered.Items[0].NotificationID != "notif_2" {
		t.Fatalf("unexpected filtered list: %#v err=%v", filtered, err)
	}

	count, err := repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 2 || count.ByType["system"] != 1 || count.ByType["social"] != 1 {
		t.Fatalf("unexpected unread count: %#v err=%v", count, err)
	}

	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{"notif_3"}); err != nil {
		t.Fatalf("mark as read: %v", err)
	}
	count, err = repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 1 || count.ByType["system"] != 0 {
		t.Fatalf("unexpected unread count after mark: %#v err=%v", count, err)
	}

	if err := repo.MarkAllRead(context.Background(), "user_alice"); err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	count, err = repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 0 {
		t.Fatalf("unexpected unread count after mark all: %#v err=%v", count, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryReturnsEmptyListWithoutFallback(t *testing.T) {
	backend := newNotificationBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &notificationFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	list, err := repo.List(context.Background(), "user_alice", NotificationListRequest{})
	if err != nil || len(list.Items) != 0 {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
	count, err := repo.UnreadCount(context.Background(), "user_alice")
	if err != nil || count.Total != 0 {
		t.Fatalf("unexpected count: %#v err=%v", count, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryMarkAsReadReturnsErrorOnExecFailure(t *testing.T) {
	backend := newNotificationBackendStub()
	backend.execErrs[queryMarkNotificationsRead] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &notificationFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{"notif_1"}); err == nil {
		t.Fatal("expected mark as read error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryListReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newNotificationBackendStub()
	backend.queryErrs[queryListNotifications] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &notificationFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.List(context.Background(), "user_alice", NotificationListRequest{})
	if err == nil {
		t.Fatal("expected list error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryUnreadCountReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newNotificationBackendStub()
	backend.queryErrs[queryUnreadNotificationCounts] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &notificationFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.UnreadCount(context.Background(), "user_alice")
	if err == nil {
		t.Fatal("expected unread count error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}
