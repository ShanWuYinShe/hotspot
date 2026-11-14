package notify

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITNotifyMarkReadTransaction 在真实库上验证 mark_read 的「公告虚拟
// 通知已读 + 普通通知已读」两段写在同一事务内：
//   - 第二段（notifications UPDATE）确定性失败时，第一段（announcement_reads
//     INSERT）随事务回滚，普通通知也保持未读；
//   - 故障解除后同一请求成功，两段写一并提交。
//
// 失败注入不依赖 sleep/取消 ctx（取消 ctx 会让回滚本身拿不到连接、污染连接池）：
// 用另一条连接对目标通知行加 FOR UPDATE 行锁，再让被测连接池的 lock_timeout
// 到期，使第二段 SQL 以锁超时错误失败，事务随后走正常 ctx 回滚。
func TestPostgresITNotifyMarkReadTransaction(t *testing.T) {
	testdb.SkipIfShort(t)
	base := testdb.Open(t)
	dsn := testdb.DSN()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID := "it_notify_markread_tx_user"
	notificationID := "it_notify_markread_tx_notify"
	announcementID := ""

	baseQDB, ok := base.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("testdb factory.DB is not QueryableDB")
	}
	// 本用例提交行（公告 read 行/通知/公告/用户）必须清理：共享库其它包的 IT
	// 依赖计数类断言，残留行会破坏它们。
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if announcementID != "" {
			_ = baseQDB.Exec(cleanupCtx, "DELETE FROM announcement_reads WHERE announcement_id = $1", announcementID)
			_ = baseQDB.Exec(cleanupCtx, "DELETE FROM system_announcements WHERE announcement_id = $1", announcementID)
		}
		_ = baseQDB.Exec(cleanupCtx, "DELETE FROM announcement_reads WHERE user_id = $1", userID)
		_ = baseQDB.Exec(cleanupCtx, "DELETE FROM notifications WHERE user_id = $1", userID)
		_ = baseQDB.Exec(cleanupCtx, "DELETE FROM users WHERE user_id = $1", userID)
	})

	now := time.Now().UTC()
	// 用户先于公告注册（虚拟通知可见性口径：公告创建时已注册用户可见）。
	if err := baseQDB.Exec(ctx,
		"INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, s2_cell_id, presence_status, last_online, created_at, register_ip, register_location, last_login_ip, last_login_location) VALUES ($1,$1,'','normal','','','','','',0,0,0,'offline',$2,$2,'','','','')",
		userID, now.Add(-time.Hour)); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	ann, err := systemmodule.NewPostgresRepository(base).CreateAnnouncement(ctx, systemmodule.AnnouncementCreateInput{
		Actor:    systemmodule.AnnouncementActor{UserID: userID, Nickname: "IT 管理员"},
		Title:    "事务公告",
		Content:  "mark_read 事务验证",
		Severity: systemmodule.AnnounceLevelInfo,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	announcementID = ann.AnnouncementID
	if err := baseQDB.Exec(ctx,
		"INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload) VALUES ($1,$2,'new_follower',FALSE,$3,$4::jsonb)",
		notificationID, userID, now, `{"type":"new_follower"}`); err != nil {
		t.Fatalf("insert notification: %v", err)
	}

	// 被测连接池：lock_timeout 使第二段在行锁冲突时确定性失败。
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	poolConfig.ConnConfig.RuntimeParams["lock_timeout"] = "250ms"
	lockPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create lock-timeout pool: %v", err)
	}
	t.Cleanup(lockPool.Close)
	txDB := persistence.NewPostgresDB(dsn)
	txDB.Pool = lockPool
	txFactory := &persistence.Factory{Config: persistence.Config{PostgresDSN: dsn}, DB: txDB}
	repo := NewPostgresRepository(txFactory)
	repo.SetAnnouncementNotificationSource(systemmodule.NewPostgresRepository(txFactory))

	// 另一条连接持有目标通知行锁，模拟第二段写失败。
	blocker, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect blocker: %v", err)
	}
	blockerTx, err := blocker.Begin(ctx)
	if err != nil {
		_ = blocker.Close(ctx)
		t.Fatalf("begin blocker tx: %v", err)
	}
	var locked int
	if err := blockerTx.QueryRow(ctx,
		"SELECT 1 FROM notifications WHERE notification_id = $1 AND user_id = $2 FOR UPDATE",
		notificationID, userID).Scan(&locked); err != nil {
		_ = blockerTx.Rollback(ctx)
		_ = blocker.Close(ctx)
		t.Fatalf("lock notification row: %v", err)
	}
	if locked != 1 {
		_ = blockerTx.Rollback(ctx)
		_ = blocker.Close(ctx)
		t.Fatalf("lock notification row: got %d", locked)
	}

	markIDs := []string{systemmodule.AnnouncementNotifyID(announcementID), notificationID}
	markCtx, markCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer markCancel()
	if err := repo.MarkAsRead(markCtx, userID, markIDs); err == nil {
		t.Fatal("第二段写失败时 mark_read 必须返回错误")
	}

	var readRows int
	if err := baseQDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM announcement_reads WHERE announcement_id = $1 AND user_id = $2",
		announcementID, userID).Scan(&readRows); err != nil {
		t.Fatalf("count announcement_reads: %v", err)
	}
	if readRows != 0 {
		t.Fatalf("公告已读行必须随事务回滚，实际残留 %d 行", readRows)
	}
	var isRead bool
	if err := baseQDB.QueryRow(ctx,
		"SELECT is_read FROM notifications WHERE notification_id = $1 AND user_id = $2",
		notificationID, userID).Scan(&isRead); err != nil {
		t.Fatalf("read notification: %v", err)
	}
	if isRead {
		t.Fatal("事务回滚后普通通知必须保持未读")
	}

	// 解除行锁后重试成功：两段写一并提交。
	if err := blockerTx.Rollback(ctx); err != nil {
		t.Fatalf("release blocker tx: %v", err)
	}
	if err := blocker.Close(ctx); err != nil {
		t.Fatalf("close blocker: %v", err)
	}
	if err := repo.MarkAsRead(markCtx, userID, markIDs); err != nil {
		t.Fatalf("重试 mark_read: %v", err)
	}
	if err := baseQDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM announcement_reads WHERE announcement_id = $1 AND user_id = $2",
		announcementID, userID).Scan(&readRows); err != nil {
		t.Fatalf("count announcement_reads after retry: %v", err)
	}
	if readRows != 1 {
		t.Fatalf("重试成功后公告已读行 = %d，期望 1", readRows)
	}
	if err := baseQDB.QueryRow(ctx,
		"SELECT is_read FROM notifications WHERE notification_id = $1 AND user_id = $2",
		notificationID, userID).Scan(&isRead); err != nil {
		t.Fatalf("read notification after retry: %v", err)
	}
	if !isRead {
		t.Fatal("重试成功后普通通知应已读")
	}
}
