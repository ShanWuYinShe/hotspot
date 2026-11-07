package system

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITAnnouncementNotificationVirtualization：系统公告通知改虚拟模型
// 的真实库行为：
//   - 公告本体不落 notifications 行，读时按「公告创建时已注册用户」可见；
//   - notify_id 稳定为 sysann_<announcement_id>；
//   - 已读状态稀疏存储，MarkRead/MarkAll 幂等，未读计数随之下降。
func TestPostgresITAnnouncementNotificationVirtualization(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		insertUser := func(userID string, createdAt time.Time) {
			if err := qdb.Exec(ctx, "INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, s2_cell_id, presence_status, last_online, created_at, register_ip, register_location, last_login_ip, last_login_location) VALUES ($1,$1,'','normal','','','','', '',0,0,0,'offline',$2,$2,'','','','')", userID, createdAt.UTC()); err != nil {
				t.Fatalf("insert user %s: %v", userID, err)
			}
		}
		olderUser := "it_ann_older"
		newerUser := "it_ann_newer"
		insertUser(olderUser, time.Now().UTC().Add(-time.Hour))
		repo := NewPostgresRepository(txFactory)
		ann, err := repo.CreateAnnouncement(ctx, AnnouncementCreateInput{
			Actor:    AnnouncementActor{UserID: olderUser, Nickname: "管理员"},
			Title:    "虚拟公告",
			Content:  "内容",
			Severity: AnnounceLevelInfo,
			SendPush: true,
		})
		if err != nil {
			t.Fatalf("create announcement: %v", err)
		}
		// 公告创建之后注册的用户不应看到它（与旧 fanout 快照口径一致）。
		insertUser(newerUser, time.Now().UTC().Add(time.Minute))

		var legacyRows int
		if err := qdb.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE type = 'system_announcement'").Scan(&legacyRows); err != nil {
			t.Fatalf("count legacy rows: %v", err)
		}
		if legacyRows != 0 {
			t.Fatalf("virtual model must not fan out notifications rows, got %d", legacyRows)
		}

		items, err := repo.ListAnnouncementNotifications(ctx, olderUser, AnnouncementNotifyCursor{}, nil, 20)
		if err != nil {
			t.Fatalf("list announcements: %v", err)
		}
		if len(items) != 1 || items[0].AnnouncementID != ann.AnnouncementID || items[0].IsRead {
			t.Fatalf("unexpected virtual items: %#v", items)
		}
		if got, want := AnnouncementNotifyID(ann.AnnouncementID), AnnouncementNotifyPrefix+ann.AnnouncementID; got != want {
			t.Fatalf("notify id = %q, want %q", got, want)
		}
		if parsed, ok := ParseAnnouncementNotifyID(AnnouncementNotifyID(ann.AnnouncementID)); !ok || parsed != ann.AnnouncementID {
			t.Fatalf("parse notify id: %q ok=%v", parsed, ok)
		}
		if count, err := repo.UnreadAnnouncementCount(ctx, olderUser); err != nil || count != 1 {
			t.Fatalf("unread count = %d err=%v", count, err)
		}
		if err := repo.MarkAnnouncementRead(ctx, olderUser, []string{ann.AnnouncementID}); err != nil {
			t.Fatalf("mark read: %v", err)
		}
		if err := repo.MarkAnnouncementRead(ctx, olderUser, []string{ann.AnnouncementID}); err != nil {
			t.Fatalf("mark read idempotent: %v", err)
		}
		read := true
		readItems, err := repo.ListAnnouncementNotifications(ctx, olderUser, AnnouncementNotifyCursor{}, &read, 20)
		if err != nil || len(readItems) != 1 || !readItems[0].IsRead {
			t.Fatalf("read filter: items=%#v err=%v", readItems, err)
		}
		unread := false
		unreadItems, err := repo.ListAnnouncementNotifications(ctx, olderUser, AnnouncementNotifyCursor{}, &unread, 20)
		if err != nil || len(unreadItems) != 0 {
			t.Fatalf("unread filter: items=%#v err=%v", unreadItems, err)
		}
		if count, err := repo.UnreadAnnouncementCount(ctx, olderUser); err != nil || count != 0 {
			t.Fatalf("unread count after read = %d err=%v", count, err)
		}
		// 新用户不可见（公告早于其注册时间）。
		if count, err := repo.UnreadAnnouncementCount(ctx, newerUser); err != nil || count != 0 {
			t.Fatalf("newer user unread = %d err=%v", count, err)
		}
		// MarkAll 幂等（对已读用户 no-op）。
		if err := repo.MarkAllAnnouncementsRead(ctx, olderUser); err != nil {
			t.Fatalf("mark all: %v", err)
		}
	})
}
