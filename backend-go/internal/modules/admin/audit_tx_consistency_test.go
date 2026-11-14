package admin

import (
	"context"
	"errors"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
)

// 本文件锁定第 81 轮审计的「审计写入与业务写同事务」口径：
// 举报裁决/公告创建的审计 INSERT 必须与业务落库同事务提交——审计失败整体
// 回滚，管理员可安全重试；不再靠 aftercommit best-effort，避免 COMMIT 与回调
// 间崩溃/审计写失败导致永久丢审计，或公告已发布但返回 10600 后同 Key 重试
// 重复发布 + 重复全体广播。
// 回滚语义在真实库上用 IT（audit_tx_it_test.go）覆盖；这里的 fake TxRunner
// 只模拟「失败时撤销内存仓储写入」，用于无库单测断言错误出口与副作用顺序。

// reportRollbackTxRunner 模拟真实事务回滚：回调出错时把内存仓库中的举报记录
// 恢复为调用前快照（admin 内存仓储未实现 persistence.MemoryTx 撤销日志）。
func reportRollbackTxRunner(repo *InMemoryRepository, reportID string) func(context.Context, func(context.Context) error) error {
	return func(ctx context.Context, fn func(context.Context) error) error {
		repo.mu.RLock()
		saved, found := repo.reports[reportID]
		repo.mu.RUnlock()
		err := fn(ctx)
		if err != nil && found {
			repo.mu.Lock()
			repo.reports[reportID] = saved
			repo.mu.Unlock()
		}
		return err
	}
}

func countAuditByAction(repo *InMemoryRepository, action AuditAction, targetID string) int {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	count := 0
	for _, entry := range repo.auditLogs {
		if entry.Action == action && entry.TargetID == targetID {
			count++
		}
	}
	return count
}

func TestHandleReportAuditFailureRollsBackDecision(t *testing.T) {
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	push := &capturePushStub{}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, NewInMemoryModerationStore(), push)
	svc.SetReportAdminStore(testReportAdminStore{store: repo.InMemoryRepository})
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	seedPendingReport(t, repo.InMemoryRepository, "report_tx_audit", "user_reporter", "user", "user_target")
	svc.SetTxRunner(reportRollbackTxRunner(repo.InMemoryRepository, "report_tx_audit"))

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	req := JudgeAdminReportRequest{ReportID: "report_tx_audit", Action: ReportActionResolve, Note: "confirmed"}
	apiErr, err := svc.HandleReport(adminCtx("admin1"), req)
	if !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort decision: apiErr=%v err=%v", apiErr, err)
	}
	// 审计失败 → 裁决事务回滚：举报仍 pending，未向举报人发通知。
	record, found, getErr := repo.GetReportHandleRecord(context.Background(), "report_tx_audit")
	if getErr != nil || !found {
		t.Fatalf("report record missing: found=%v err=%v", found, getErr)
	}
	if record.Item.Status != ReportStatusPending {
		t.Fatalf("report status = %q, want pending (decision must roll back)", record.Item.Status)
	}
	if len(push.resolvedCalls) != 0 || len(push.dismissCalls) != 0 {
		t.Fatalf("audit failure must not notify reporter: resolved=%d dismiss=%d", len(push.resolvedCalls), len(push.dismissCalls))
	}

	// 管理员可安全重试：撤掉审计故障后同一裁决成功，且只落一条 report.judge 审计。
	repo.auditErr = nil
	if apiErr, err := svc.HandleReport(adminCtx("admin1"), req); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	record, found, getErr = repo.GetReportHandleRecord(context.Background(), "report_tx_audit")
	if getErr != nil || !found || record.Item.Status != ReportStatusResolved {
		t.Fatalf("retry must resolve report: found=%v status=%q err=%v", found, record.Item.Status, getErr)
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditAction("report.judge"), "report_tx_audit"); got != 1 {
		t.Fatalf("report.judge audit count = %d, want 1", got)
	}
	if len(push.resolvedCalls) != 1 {
		t.Fatalf("retry must notify reporter once, got %d", len(push.resolvedCalls))
	}
}

func TestCreateAnnouncementAuditFailureRollsBackPublish(t *testing.T) {
	for _, sendPush := range []bool{true, false} {
		sendPush := sendPush
		name := "send_push_false"
		if sendPush {
			name = "send_push_true"
		}
		t.Run(name, func(t *testing.T) {
			mods := NewInMemoryModerationStore()
			push := &stubPushService{}
			repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
			svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, push)
			writer := &captureOutboxWriter{}
			svc.SetEventOutbox(writer)
			// 模拟真实事务回滚：失败时恢复公告表快照（含 SendPush=false 的纯落库路径）。
			svc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
				mods.mu.RLock()
				snapshot := make(map[string]systemmodule.AnnouncementRecord, len(mods.announcements))
				for id, record := range mods.announcements {
					snapshot[id] = record
				}
				mods.mu.RUnlock()
				err := fn(ctx)
				if err != nil {
					mods.mu.Lock()
					mods.announcements = snapshot
					mods.mu.Unlock()
				}
				return err
			})

			auditErr := errors.New("audit insert failed")
			repo.auditErr = auditErr
			req := CreateAnnouncementRequest{Title: "维护通知", Content: "今晚维护", Severity: AnnounceLevelInfo, SendPush: sendPush}
			if _, apiErr, err := svc.CreateAnnouncement(adminCtx("admin"), req); !errors.Is(err, auditErr) || apiErr != nil {
				t.Fatalf("audit failure must abort publish: apiErr=%v err=%v", apiErr, err)
			}
			// 审计失败 → 公告未落库、广播未入队、未同步广播。
			if len(mods.announcements) != 0 {
				t.Fatalf("announcement must not persist on audit failure: %d rows", len(mods.announcements))
			}
			if writer.calls != 0 {
				t.Fatalf("outbox enqueue calls = %d, want 0 on audit failure", writer.calls)
			}
			if push.pushed {
				t.Fatal("sync broadcast must not happen on audit failure")
			}

			// 幂等中间件只缓存成功响应：同 Key（服务层同请求）重试成功且只发布一次。
			repo.auditErr = nil
			if _, apiErr, err := svc.CreateAnnouncement(adminCtx("admin"), req); apiErr != nil || err != nil {
				t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
			}
			if len(mods.announcements) != 1 {
				t.Fatalf("retry must publish exactly one announcement, got %d", len(mods.announcements))
			}
			if push.pushed {
				t.Fatal("outbox path must not broadcast synchronously")
			}
			wantEnqueues := 0
			if sendPush {
				wantEnqueues = 1
			}
			if writer.calls != wantEnqueues {
				t.Fatalf("outbox enqueue calls = %d, want %d", writer.calls, wantEnqueues)
			}
			for announcementID := range mods.announcements {
				if got := countAuditByAction(repo.InMemoryRepository, AuditActionAnnouncementCreate, announcementID); got != 1 {
					t.Fatalf("announcement.create audit count = %d, want 1", got)
				}
			}
		})
	}
}
