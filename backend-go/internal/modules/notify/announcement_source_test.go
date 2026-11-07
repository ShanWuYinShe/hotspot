package notify

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// fakeAnnouncementSource 模拟 system 模块公告来源（只覆盖 notify 消费的接口）。
type fakeAnnouncementSource struct {
	items       []systemmodule.AnnouncementNotification
	unread      int
	listErr     error
	unreadErr   error
	markErr     error
	markAllErr  error
	marked      []string
	markedAll   bool
	listCursors []systemmodule.AnnouncementNotifyCursor
}

func (f *fakeAnnouncementSource) ListAnnouncementNotifications(_ context.Context, _ string, cursor systemmodule.AnnouncementNotifyCursor, isRead *bool, limit int) ([]systemmodule.AnnouncementNotification, error) {
	f.listCursors = append(f.listCursors, cursor)
	if f.listErr != nil {
		return nil, f.listErr
	}
	items := append([]systemmodule.AnnouncementNotification(nil), f.items...)
	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return systemmodule.AnnouncementNotifyID(items[i].AnnouncementID) > systemmodule.AnnouncementNotifyID(items[j].AnnouncementID)
	})
	out := make([]systemmodule.AnnouncementNotification, 0, limit)
	for _, item := range items {
		if isRead != nil && item.IsRead != *isRead {
			continue
		}
		notifyID := systemmodule.AnnouncementNotifyID(item.AnnouncementID)
		if !cursor.CreatedAt.IsZero() {
			if item.CreatedAt.After(cursor.CreatedAt) {
				continue
			}
			if item.CreatedAt.Equal(cursor.CreatedAt) && notifyID >= cursor.NotifyID {
				continue
			}
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (f *fakeAnnouncementSource) UnreadAnnouncementCount(context.Context, string) (int, error) {
	return f.unread, f.unreadErr
}

func (f *fakeAnnouncementSource) MarkAnnouncementRead(_ context.Context, _ string, announcementIDs []string) error {
	f.marked = append(f.marked, announcementIDs...)
	return f.markErr
}

func (f *fakeAnnouncementSource) MarkAllAnnouncementsRead(context.Context, string) error {
	f.markedAll = true
	return f.markAllErr
}

func newMergedNotifyRepo(t *testing.T) (*PostgresRepository, *notifyBackendStub) {
	t.Helper()
	backend := newNotifyBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return NewPostgresRepository(factory), backend
}

func TestPostgresRepositoryListMergesAnnouncementVirtualRows(t *testing.T) {
	repo, backend := newMergedNotifyRepo(t)
	backend.items["user_alice"] = []NotifyItem{
		{NotifyID: "notif_newer", Type: "new_follower", CreatedAt: "2026-03-20T02:00:00Z", Payload: map[string]any{"type": "new_follower"}},
		{NotifyID: "notif_older", Type: "new_follower", CreatedAt: "2026-03-20T00:00:00Z", Payload: map[string]any{"type": "new_follower"}},
	}
	source := &fakeAnnouncementSource{items: []systemmodule.AnnouncementNotification{
		{AnnouncementID: "ann_1", Title: "公告", Severity: systemmodule.AnnounceLevelInfo, CreatedAt: mustNotifyTime(t, "2026-03-20T01:00:00Z")},
	}}
	repo.SetAnnouncementNotificationSource(source)
	page, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Pagination: &PaginationParams{Limit: 2}})
	if err != nil {
		t.Fatalf("list merged: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("unexpected page: %#v", page)
	}
	want := []string{"notif_newer", systemmodule.AnnouncementNotifyID("ann_1")}
	for i := range want {
		if page.Items[i].NotifyID != want[i] {
			t.Fatalf("item[%d] = %q, want %q", i, page.Items[i].NotifyID, want[i])
		}
	}
	if page.Items[1].Type != "system_announcement" || page.Items[1].Payload["announcement_id"] != "ann_1" {
		t.Fatalf("unexpected virtual item: %#v", page.Items[1])
	}
	next, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Pagination: &PaginationParams{Limit: 2, Cursor: *page.NextCursor}})
	if err != nil || len(next.Items) != 1 || next.Items[0].NotifyID != "notif_older" {
		t.Fatalf("unexpected next page: %#v err=%v", next, err)
	}
	// 游标应透传给公告来源（同一全序）。
	if len(source.listCursors) == 0 || source.listCursors[len(source.listCursors)-1].NotifyID != systemmodule.AnnouncementNotifyID("ann_1") {
		t.Fatalf("announcement source did not receive cursor: %#v", source.listCursors)
	}
}

func TestPostgresRepositoryAnnouncementTypeFilterAndReadFilter(t *testing.T) {
	repo, backend := newMergedNotifyRepo(t)
	backend.items["user_alice"] = []NotifyItem{{NotifyID: "notif_1", Type: "new_follower", CreatedAt: "2026-03-20T00:00:00Z", Payload: map[string]any{"type": "new_follower"}}}
	source := &fakeAnnouncementSource{items: []systemmodule.AnnouncementNotification{
		{AnnouncementID: "ann_read", Title: "已读", Severity: systemmodule.AnnounceLevelInfo, IsRead: true, CreatedAt: mustNotifyTime(t, "2026-03-20T01:00:00Z")},
		{AnnouncementID: "ann_unread", Title: "未读", Severity: systemmodule.AnnounceLevelWarning, IsRead: false, CreatedAt: mustNotifyTime(t, "2026-03-20T02:00:00Z")},
	}}
	repo.SetAnnouncementNotificationSource(source)
	systemPage, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Type: "system_announcement"})
	if err != nil || len(systemPage.Items) != 2 {
		t.Fatalf("system type filter: %#v err=%v", systemPage, err)
	}
	unread := false
	unreadPage, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Type: "system_announcement", IsRead: &unread})
	if err != nil || len(unreadPage.Items) != 1 || unreadPage.Items[0].Payload["announcement_id"] != "ann_unread" {
		t.Fatalf("system unread filter: %#v err=%v", unreadPage, err)
	}
	followPage, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Type: "new_follower"})
	if err != nil || len(followPage.Items) != 1 || followPage.Items[0].NotifyID != "notif_1" {
		t.Fatalf("persisted type filter: %#v err=%v", followPage, err)
	}
}

func TestPostgresRepositoryAnnouncementUnreadAndMarkReadRouting(t *testing.T) {
	repo, backend := newMergedNotifyRepo(t)
	backend.items["user_alice"] = []NotifyItem{{NotifyID: "notif_1", Type: "new_follower", CreatedAt: "2026-03-20T00:00:00Z", Payload: map[string]any{"type": "new_follower"}}}
	source := &fakeAnnouncementSource{unread: 2}
	repo.SetAnnouncementNotificationSource(source)

	unread, err := repo.UnreadCount(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if unread.Total != 3 || unread.ByType.System != 2 || unread.ByType.Social != 1 {
		t.Fatalf("unexpected unread: %#v", unread)
	}
	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{systemmodule.AnnouncementNotifyID("ann_1"), "notif_1"}); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if len(source.marked) != 1 || source.marked[0] != "ann_1" {
		t.Fatalf("announcement ids not routed to source: %#v", source.marked)
	}
	if !backend.items["user_alice"][0].IsRead {
		t.Fatalf("persisted notification not marked read")
	}
	if err := repo.MarkAllRead(context.Background(), "user_alice"); err != nil {
		t.Fatalf("mark all: %v", err)
	}
	if !source.markedAll {
		t.Fatalf("mark all not forwarded to announcement source")
	}
}

func TestPostgresRepositoryAnnouncementSourceErrorsPropagate(t *testing.T) {
	repo, _ := newMergedNotifyRepo(t)
	source := &fakeAnnouncementSource{listErr: errors.New("boom"), unreadErr: errors.New("unread boom"), markErr: errors.New("mark boom"), markAllErr: errors.New("markall boom")}
	repo.SetAnnouncementNotificationSource(source)
	if _, err := repo.List(context.Background(), "user_alice", NotifyListRequest{}); err == nil {
		t.Fatal("list must propagate announcement source error")
	}
	if _, err := repo.UnreadCount(context.Background(), "user_alice"); err == nil {
		t.Fatal("unread count must propagate announcement source error")
	}
	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{systemmodule.AnnouncementNotifyID("ann_1")}); err == nil {
		t.Fatal("mark read must propagate announcement source error")
	}
	if err := repo.MarkAllRead(context.Background(), "user_alice"); err == nil {
		t.Fatal("mark all must propagate announcement source error")
	}
}

func TestPostgresRepositoryAnnouncementInvalidCursorRejected(t *testing.T) {
	repo, _ := newMergedNotifyRepo(t)
	repo.SetAnnouncementNotificationSource(&fakeAnnouncementSource{})
	if _, err := repo.List(context.Background(), "user_alice", NotifyListRequest{Pagination: &PaginationParams{Cursor: encodeNotifyCursor(notifyCursor{CreatedAt: "not-a-time", NotifyID: "n1"})}}); !errors.Is(err, ErrInvalidNotifyRequest) {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func mustNotifyTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
