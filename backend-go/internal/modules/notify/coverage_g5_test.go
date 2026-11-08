package notify

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// countG5Notifications 统计指定用户的通知行数（测试专用，供 prune/purge 断言）。
func countG5Notifications(t *testing.T, ctx context.Context, q persistence.Queryable, userID string) int {
	t.Helper()
	rows, err := q.Query(ctx, "SELECT notification_id FROM notifications WHERE user_id = $1", userID)
	if err != nil {
		t.Fatalf("query notifications: %v", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan notification id: %v", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate notifications: %v", err)
	}
	return count
}

// insertG5Notification 插入一条通知（只用于构造 prune/purge 前置数据）。
func insertG5Notification(t *testing.T, ctx context.Context, q persistence.Queryable, notifyID, userID string, isRead bool, createdAt time.Time) {
	t.Helper()
	if err := q.Exec(ctx,
		"INSERT INTO notifications (notification_id, user_id, type, is_read, created_at, payload) VALUES ($1, $2, $3, $4, $5, '{}'::jsonb)",
		notifyID, userID, "system", isRead, createdAt); err != nil {
		t.Fatalf("insert notification %s: %v", notifyID, err)
	}
}

// TestPostgresIT_PruneReadBefore 覆盖后台 prune sweeper 依赖的分批物理删除：
// 只删「已读且 created_at 早于截止时间」的行，未读/较新行必须保留；删空后再执行
// 返回 0（可重复执行）。
func TestPostgresIT_PruneReadBefore(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		userID := testdb.InsertUser(t, txFactory, "u_g5_prune", "PruneUser")
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("expected transaction queryable")
		}

		old := time.Now().UTC().Add(-48 * time.Hour)
		fresh := time.Now().UTC().Add(time.Hour)
		insertG5Notification(t, ctx, q, "notif_g5_prune_old_read", userID, true, old)
		insertG5Notification(t, ctx, q, "notif_g5_prune_old_unread", userID, false, old)
		insertG5Notification(t, ctx, q, "notif_g5_prune_fresh_read", userID, true, fresh)

		pruned, err := repo.PruneReadBefore(ctx, time.Now().UTC())
		if err != nil {
			t.Fatalf("PruneReadBefore: %v", err)
		}
		if pruned != 1 {
			t.Fatalf("PruneReadBefore pruned = %d, want 1", pruned)
		}
		if got := countG5Notifications(t, ctx, q, userID); got != 2 {
			t.Fatalf("notifications after prune = %d, want 2 (unread + fresh read kept)", got)
		}

		// 幂等：没有更多可删行时返回 0 而不是死循环。
		pruned, err = repo.PruneReadBefore(ctx, time.Now().UTC())
		if err != nil {
			t.Fatalf("PruneReadBefore second run: %v", err)
		}
		if pruned != 0 {
			t.Fatalf("PruneReadBefore second run pruned = %d, want 0", pruned)
		}
	})
}

// TestPostgresIT_PurgeUserData 覆盖账户注销级联：仓储与 Service 两个入口都能
// 按 user_id 清空本模块数据，且不误删其他用户的通知。
func TestPostgresIT_PurgeUserData(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		purgedByRepo := testdb.InsertUser(t, txFactory, "u_g5_purge_repo", "PurgeRepoUser")
		purgedBySvc := testdb.InsertUser(t, txFactory, "u_g5_purge_svc", "PurgeSvcUser")
		kept := testdb.InsertUser(t, txFactory, "u_g5_purge_keep", "PurgeKeepUser")
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("expected transaction queryable")
		}
		now := time.Now().UTC()
		insertG5Notification(t, ctx, q, "notif_g5_purge_repo", purgedByRepo, false, now)
		insertG5Notification(t, ctx, q, "notif_g5_purge_svc", purgedBySvc, false, now)
		insertG5Notification(t, ctx, q, "notif_g5_purge_keep", kept, false, now)

		if err := repo.PurgeUserData(ctx, purgedByRepo); err != nil {
			t.Fatalf("repo.PurgeUserData: %v", err)
		}
		svc := NewServiceWithRepo(repo)
		if err := svc.PurgeUserData(ctx, purgedBySvc); err != nil {
			t.Fatalf("svc.PurgeUserData: %v", err)
		}

		if got := countG5Notifications(t, ctx, q, purgedByRepo); got != 0 {
			t.Fatalf("repo purged user notifications = %d, want 0", got)
		}
		if got := countG5Notifications(t, ctx, q, purgedBySvc); got != 0 {
			t.Fatalf("svc purged user notifications = %d, want 0", got)
		}
		if got := countG5Notifications(t, ctx, q, kept); got != 1 {
			t.Fatalf("unrelated user notifications = %d, want 1", got)
		}
	})
}

// TestServicePurgeUserDataWithoutPurger 覆盖 Service 分支：repo 不实现
// userDataPurger（内存仓储）时优雅返回 nil，不报错。
func TestServicePurgeUserDataWithoutPurger(t *testing.T) {
	svc := NewServiceWithRepo(NewInMemoryRepository())
	if err := svc.PurgeUserData(context.Background(), "u_g5_no_purger"); err != nil {
		t.Fatalf("PurgeUserData with memory repo: %v", err)
	}
}
