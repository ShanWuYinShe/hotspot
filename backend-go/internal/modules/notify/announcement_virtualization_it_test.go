package notify

import (
	"context"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITNotifyAnnouncementVirtualization：notify 读取路径把 system 公告
// 虚拟行与 notifications 行按同一全序合并；分页游标、类型/已读过滤、未读计数、
// mark_read 路由（sysann_ 前缀 → system 公告已读表）都在真实库上验证。
func TestPostgresITNotifyAnnouncementVirtualization(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		userID := "it_notify_virtual"
		now := time.Now().UTC()
		if err := qdb.Exec(ctx, "INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, s2_cell_id, presence_status, last_online, created_at, register_ip, register_location, last_login_ip, last_login_location) VALUES ($1,$1,'','normal','','','','', '',0,0,0,'offline',$2,$2,'','','','')", userID, now.Add(-24*time.Hour)); err != nil {
			t.Fatalf("insert user: %v", err)
		}
		systemRepo := systemmodule.NewPostgresRepository(txFactory)
		ann, err := systemRepo.CreateAnnouncement(ctx, systemmodule.AnnouncementCreateInput{
			Actor:    systemmodule.AnnouncementActor{UserID: userID, Nickname: "管理员"},
			Title:    "虚拟公告",
			Content:  "内容",
			Severity: systemmodule.AnnounceLevelInfo,
			SendPush: true,
		})
		if err != nil {
			t.Fatalf("create announcement: %v", err)
		}
		announcementAt := now.Add(-time.Hour)
		if err := qdb.Exec(ctx, "UPDATE system_announcements SET created_at = $2 WHERE announcement_id = $1", ann.AnnouncementID, announcementAt); err != nil {
			t.Fatalf("set announcement time: %v", err)
		}
		// 三条持久化通知：newer/older 夹住公告，用于验证跨来源归并。
		insertNotify := func(id string, createdAt time.Time, isRead bool) {
			payload := `{"type":"new_follower","actor":{"user_id":"u1","nickname":"u1"}}`
			if err := qdb.Exec(ctx, "INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload) VALUES ($1,$2,'new_follower',$3,$4,$5::jsonb)", id, userID, isRead, createdAt, payload); err != nil {
				t.Fatalf("insert notify %s: %v", id, err)
			}
		}
		insertNotify("n_newer", anniversary(now.Add(-time.Minute)), false)
		insertNotify("n_older", anniversary(now.Add(-2*time.Hour)), false)

		repo := NewPostgresRepository(txFactory)
		repo.SetAnnouncementNotificationSource(systemRepo)
		page, err := repo.List(ctx, userID, NotifyListRequest{})
		if err != nil {
			t.Fatalf("list merged: %v", err)
		}
		if len(page.Items) != 3 {
			t.Fatalf("merged items = %d (%#v), want 3", len(page.Items), page.Items)
		}
		wantIDs := []string{"n_newer", systemmodule.AnnouncementNotifyID(ann.AnnouncementID), "n_older"}
		for i, want := range wantIDs {
			if page.Items[i].NotifyID != want {
				t.Fatalf("item[%d] id = %q, want %q (%#v)", i, page.Items[i].NotifyID, want, page.Items)
			}
		}
		if page.Items[1].Type != "system_announcement" || page.Items[1].Payload["title"] != "虚拟公告" {
			t.Fatalf("unexpected virtual item: %#v", page.Items[1])
		}

		// 类型过滤：仅系统公告。
		systemOnly, err := repo.List(ctx, userID, NotifyListRequest{Type: "system_announcement"})
		if err != nil || len(systemOnly.Items) != 1 || systemOnly.Items[0].NotifyID != systemmodule.AnnouncementNotifyID(ann.AnnouncementID) {
			t.Fatalf("system type filter: %#v err=%v", systemOnly, err)
		}

		// 未读摘要：2 条普通 + 1 条虚拟公告。
		unread, err := repo.UnreadCount(ctx, userID)
		if err != nil {
			t.Fatalf("unread count: %v", err)
		}
		if unread.Total != 3 || unread.ByType.System != 1 || unread.ByType.Social != 2 {
			t.Fatalf("unexpected unread summary: %#v", unread)
		}

		// 分页 + 游标：每页 2 条，跨来源不重不漏。
		first, err := repo.List(ctx, userID, NotifyListRequest{Pagination: &PaginationParams{Limit: 2}})
		if err != nil || len(first.Items) != 2 || first.NextCursor == nil {
			t.Fatalf("first page: %#v err=%v", first, err)
		}
		second, err := repo.List(ctx, userID, NotifyListRequest{Pagination: &PaginationParams{Limit: 2, Cursor: *first.NextCursor}})
		if err != nil || len(second.Items) != 1 || second.Items[0].NotifyID != "n_older" {
			t.Fatalf("second page: %#v err=%v", second, err)
		}

		// mark_read：虚拟 ID 路由到公告已读表；普通 ID 更新 notifications；未读清零。
		if err := repo.MarkAsRead(ctx, userID, []string{systemmodule.AnnouncementNotifyID(ann.AnnouncementID), "n_newer"}); err != nil {
			t.Fatalf("mark read: %v", err)
		}
		unread, err = repo.UnreadCount(ctx, userID)
		if err != nil || unread.Total != 1 || unread.ByType.System != 0 || unread.ByType.Social != 1 {
			t.Fatalf("unread after mark_read: %#v err=%v", unread, err)
		}
		if err := repo.MarkAllRead(ctx, userID); err != nil {
			t.Fatalf("mark all: %v", err)
		}
		unread, err = repo.UnreadCount(ctx, userID)
		if err != nil || unread.Total != 0 {
			t.Fatalf("unread after mark_all: %#v err=%v", unread, err)
		}
		allRead, err := repo.List(ctx, userID, NotifyListRequest{IsRead: boolPtr(true)})
		if err != nil || len(allRead.Items) != 3 {
			t.Fatalf("read filter: %#v err=%v", allRead, err)
		}
	})
}

func anniversary(t time.Time) time.Time { return t.Truncate(time.Microsecond) }

func boolPtr(v bool) *bool { return &v }
