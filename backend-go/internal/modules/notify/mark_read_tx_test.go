package notify

import (
	"context"
	"errors"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
)

// 本文件锁定第 81 轮审计的「mark_read 两段写同事务」口径：公告虚拟通知已读
// （system 模块 announcement_reads）与普通通知已读（notifications.is_read）
// 必须在同一事务内提交/回滚。真实回滚语义由 mark_read_tx_it_test.go 在真实
// 库上验证；这里的 fake TxRunner 只在无库单测里模拟「失败时撤销第一段写入」，
// 用于断言错误出口与两段写的执行顺序。
func TestPostgresRepositoryMarkAsReadRollsBackBothSegmentsOnPersistedWriteFailure(t *testing.T) {
	repo, backend := newMergedNotifyRepo(t)
	backend.items["user_alice"] = []NotifyItem{{
		NotifyID:  "notif_1",
		Type:      "new_follower",
		CreatedAt: "2026-03-20T00:00:00Z",
		Payload:   map[string]any{"type": "new_follower"},
	}}
	source := &fakeAnnouncementSource{}
	repo.SetAnnouncementNotificationSource(source)
	backend.execErrs[queryMarkNotifysRead] = errors.New("persisted write failed")

	txRuns := 0
	repo.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
		txRuns++
		savedMarks := append([]string(nil), source.marked...)
		err := fn(ctx)
		if err != nil {
			// 模拟真实事务回滚：撤销事务内已完成的第一段（公告已读）写入。
			source.marked = savedMarks
		}
		return err
	})

	ids := []string{systemmodule.AnnouncementNotifyID("ann_1"), "notif_1"}
	if err := repo.MarkAsRead(context.Background(), "user_alice", ids); err == nil {
		t.Fatal("第二段写失败时 mark_read 必须返回错误")
	}
	if txRuns != 1 {
		t.Fatalf("两段写必须在同一事务内执行，txRunner 调用次数 = %d，期望 1", txRuns)
	}
	if len(source.marked) != 0 {
		t.Fatalf("公告已读行必须随事务回滚，实际落库 %v", source.marked)
	}
	if backend.items["user_alice"][0].IsRead {
		t.Fatal("普通通知不应在事务回滚后保持已读")
	}

	// 故障恢复后同一请求成功：两段写都落库，via txRunner 单事务。
	delete(backend.execErrs, queryMarkNotifysRead)
	if err := repo.MarkAsRead(context.Background(), "user_alice", ids); err != nil {
		t.Fatalf("恢复后 mark_read: %v", err)
	}
	if txRuns != 2 {
		t.Fatalf("txRunner 调用次数 = %d，期望 2", txRuns)
	}
	if len(source.marked) != 1 || source.marked[0] != "ann_1" {
		t.Fatalf("公告已读未落库: %v", source.marked)
	}
	if !backend.items["user_alice"][0].IsRead {
		t.Fatal("普通通知应已标记已读")
	}
}

func TestPostgresRepositoryMarkAllReadRollsBackOnPersistedWriteFailure(t *testing.T) {
	repo, backend := newMergedNotifyRepo(t)
	backend.items["user_alice"] = []NotifyItem{{
		NotifyID:  "notif_1",
		Type:      "new_follower",
		CreatedAt: "2026-03-20T00:00:00Z",
		Payload:   map[string]any{"type": "new_follower"},
	}}
	source := &fakeAnnouncementSource{}
	repo.SetAnnouncementNotificationSource(source)
	backend.execErrs[queryMarkAllNotifysRead] = errors.New("persisted write failed")

	txRuns := 0
	repo.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
		txRuns++
		savedAll := source.markedAll
		err := fn(ctx)
		if err != nil {
			source.markedAll = savedAll
		}
		return err
	})

	if err := repo.MarkAllRead(context.Background(), "user_alice"); err == nil {
		t.Fatal("第二段写失败时 mark_all_read 必须返回错误")
	}
	if txRuns != 1 || source.markedAll {
		t.Fatalf("公告全读必须随事务回滚: txRuns=%d markedAll=%v", txRuns, source.markedAll)
	}
	if backend.items["user_alice"][0].IsRead {
		t.Fatal("普通通知不应在事务回滚后保持已读")
	}

	delete(backend.execErrs, queryMarkAllNotifysRead)
	if err := repo.MarkAllRead(context.Background(), "user_alice"); err != nil {
		t.Fatalf("恢复后 mark_all_read: %v", err)
	}
	if !source.markedAll || !backend.items["user_alice"][0].IsRead {
		t.Fatalf("恢复后两段写都应落库: markedAll=%v isRead=%v", source.markedAll, backend.items["user_alice"][0].IsRead)
	}
}

// TestPostgresRepositoryMarkAsReadFallsBackToRunInTransactionWithoutTxRunner
// 锁定默认路径：未注入 txRunner 时仍经 runInTx（RunInTransaction）执行，
// 两条分支（注入替身 / 默认 Runner）都被覆盖。
func TestPostgresRepositoryMarkAsReadFallsBackToRunInTransactionWithoutTxRunner(t *testing.T) {
	repo, backend := newMergedNotifyRepo(t)
	backend.items["user_alice"] = []NotifyItem{{
		NotifyID:  "notif_1",
		Type:      "new_follower",
		CreatedAt: "2026-03-20T00:00:00Z",
		Payload:   map[string]any{"type": "new_follower"},
	}}
	if err := repo.MarkAsRead(context.Background(), "user_alice", []string{"notif_1"}); err != nil {
		t.Fatalf("mark as read without txRunner: %v", err)
	}
	if !backend.items["user_alice"][0].IsRead {
		t.Fatal("普通通知应已标记已读")
	}
}
