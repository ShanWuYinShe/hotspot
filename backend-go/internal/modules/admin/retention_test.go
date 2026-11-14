package admin

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestPruneRetentionOnceRemovesExpiredRows(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	old := time.Now().UTC().AddDate(0, 0, -400)
	newer := time.Now().UTC().AddDate(0, 0, -10)
	repo.auditLogs = map[string]AuditLogEntry{
		"audit_old": {LogID: "audit_old", CreatedAt: old},
		"audit_new": {LogID: "audit_new", CreatedAt: newer},
	}
	repo.auditOrder = []string{"audit_old", "audit_new"}
	handledOld := old.Format(time.RFC3339Nano)
	repo.reports = map[string]reportHandleRecord{
		"report_resolved_old": {Item: AdminReportItem{ReportID: "report_resolved_old", Status: ReportStatusResolved, HandledAt: &handledOld}},
		"report_pending_old":  {Item: AdminReportItem{ReportID: "report_pending_old", Status: ReportStatusPending}},
	}
	service := NewService(repo, nil, NewInMemoryModerationStore(), nil)
	service.SetReportAdminStore(testReportAdminStore{store: repo})

	service.pruneRetentionOnce(ctx, 365, 365)

	if _, ok := repo.auditLogs["audit_old"]; ok {
		t.Fatal("expired audit log should be deleted")
	}
	if _, ok := repo.auditLogs["audit_new"]; !ok {
		t.Fatal("fresh audit log must be kept")
	}
	if _, ok := repo.reports["report_resolved_old"]; ok {
		t.Fatal("expired resolved report should be deleted")
	}
	if _, ok := repo.reports["report_pending_old"]; !ok {
		t.Fatal("pending report must never be auto-deleted")
	}
}

// overrideRetentionBatchLimits 临时调小分批参数（单测用），用例结束还原。
func overrideRetentionBatchLimits(t *testing.T, batchSize, maxBatches int) {
	t.Helper()
	oldBatch, oldMax := retentionDeleteBatchSize, retentionDeleteMaxBatches
	retentionDeleteBatchSize, retentionDeleteMaxBatches = batchSize, maxBatches
	t.Cleanup(func() {
		retentionDeleteBatchSize, retentionDeleteMaxBatches = oldBatch, oldMax
	})
}

// resetRetentionMemoryRepo 清掉内存仓储的预置数据，避免预置审计日志干扰计数。
func resetRetentionMemoryRepo(repo *InMemoryRepository) {
	repo.auditLogs = make(map[string]AuditLogEntry)
	repo.auditOrder = nil
	repo.reports = make(map[string]reportHandleRecord)
}

// TestPruneRetentionBatchesUntilEmpty 覆盖「按 cutoff 循环分批删除，直到无行」：
// 单批大小远小于过期行数时一轮清理也必须删空，且不误删未超期行。
func TestPruneRetentionBatchesUntilEmpty(t *testing.T) {
	overrideRetentionBatchLimits(t, 2, 100)

	repo := NewInMemoryRepository()
	resetRetentionMemoryRepo(repo)
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -400)
	handledOld := old.Format(time.RFC3339Nano)
	for _, logID := range []string{"audit_b1", "audit_b2", "audit_b3", "audit_b4", "audit_b5"} {
		repo.auditLogs[logID] = AuditLogEntry{LogID: logID, CreatedAt: old}
		repo.auditOrder = append(repo.auditOrder, logID)
	}
	for _, reportID := range []string{"report_b1", "report_b2", "report_b3"} {
		repo.reports[reportID] = reportHandleRecord{Item: AdminReportItem{ReportID: reportID, Status: ReportStatusResolved, HandledAt: &handledOld}}
	}
	// 未超期审计日志必须保留（分批循环不得越过 cutoff 误删）。
	repo.auditLogs["audit_fresh"] = AuditLogEntry{LogID: "audit_fresh", CreatedAt: now}
	repo.auditOrder = append(repo.auditOrder, "audit_fresh")

	service := NewService(repo, nil, NewInMemoryModerationStore(), nil)
	service.SetReportAdminStore(testReportAdminStore{store: repo})
	service.pruneRetentionOnce(context.Background(), 365, 365)

	if len(repo.auditLogs) != 1 {
		t.Fatalf("audit logs after prune = %d, want 1 (only audit_fresh)", len(repo.auditLogs))
	}
	if _, ok := repo.auditLogs["audit_fresh"]; !ok {
		t.Fatal("fresh audit log must be kept by batched prune")
	}
	if len(repo.reports) != 0 {
		t.Fatalf("reports after prune = %d, want 0", len(repo.reports))
	}
}

// TestPruneRetentionStopsAtMaxBatches 覆盖单轮批数上限：积压超过上限时本轮只删
// 上限内的行，剩余由后续轮次追平（避免一轮 DELETE 长时间占用连接）。
func TestPruneRetentionStopsAtMaxBatches(t *testing.T) {
	overrideRetentionBatchLimits(t, 2, 2)

	repo := NewInMemoryRepository()
	resetRetentionMemoryRepo(repo)
	old := time.Now().UTC().AddDate(0, 0, -400)
	for i := 0; i < 10; i++ {
		logID := fmt.Sprintf("audit_cap_%02d", i)
		repo.auditLogs[logID] = AuditLogEntry{LogID: logID, CreatedAt: old}
		repo.auditOrder = append(repo.auditOrder, logID)
	}
	service := NewService(repo, nil, NewInMemoryModerationStore(), nil)
	for round, want := range []int{6, 2, 0} {
		service.pruneRetentionOnce(context.Background(), 365, 0)
		if got := len(repo.auditLogs); got != want {
			t.Fatalf("round %d: audit logs left = %d, want %d（单轮最多 %d 批 × %d 行）", round, got, want, retentionDeleteMaxBatches, retentionDeleteBatchSize)
		}
	}
}

// errAuditPruneRepo 在内存仓储上覆盖审计清理的错误出口（其余方法沿用内存实现）。
type errAuditPruneRepo struct {
	*InMemoryRepository
	err error
}

func (r errAuditPruneRepo) DeleteAuditLogsBefore(context.Context, time.Time, int) (int64, error) {
	return 3, r.err
}

// errReportPruneStore 覆盖举报清理的错误出口。
type errReportPruneStore struct {
	deleted int64
	err     error
}

func (s errReportPruneStore) MarkHandledAsAdmin(context.Context, string, string, time.Time, *string, *string) (bool, error) {
	return false, nil
}

func (s errReportPruneStore) DeleteHandledBefore(context.Context, time.Time, int) (int64, error) {
	return s.deleted, s.err
}

// TestDeleteRetentionBatchesPreservesCumulativeCountOnError 覆盖分批循环的错误
// 出口：中途失败时返回已完成部分的累计删除数（日志据此定位），并支持 ctx 取消。
func TestDeleteRetentionBatchesPreservesCumulativeCountOnError(t *testing.T) {
	ctx := context.Background()
	cutoff := time.Now().UTC()
	service := NewService(errAuditPruneRepo{InMemoryRepository: NewInMemoryRepository(), err: errCovAdmin}, nil, NewInMemoryModerationStore(), nil)
	deleted, err := service.deleteAuditLogsInBatches(ctx, cutoff)
	if !errors.Is(err, errCovAdmin) {
		t.Fatalf("audit prune err = %v, want errCovAdmin", err)
	}
	if deleted != 3 {
		t.Fatalf("audit prune deleted = %d, want 3（已完成部分必须累计返回）", deleted)
	}

	service.SetReportAdminStore(errReportPruneStore{deleted: 2, err: errCovAdmin})
	deleted, err = service.deleteHandledReportsInBatches(ctx, cutoff)
	if !errors.Is(err, errCovAdmin) {
		t.Fatalf("report prune err = %v, want errCovAdmin", err)
	}
	if deleted != 2 {
		t.Fatalf("report prune deleted = %d, want 2", deleted)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.deleteAuditLogsInBatches(cancelled, cutoff); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled ctx err = %v, want context.Canceled（清理应随停机立即退出）", err)
	}
}

func TestStartRetentionSweeperDisabledWhenBothRetentionZero(t *testing.T) {
	retentionSweepMu.Lock()
	retentionSweepStarted = false
	retentionSweepMu.Unlock()

	service := NewService(NewInMemoryRepository(), nil, NewInMemoryModerationStore(), nil)
	service.StartRetentionSweeper(context.Background(), 0, 0)
	if retentionSweepStarted {
		t.Fatal("sweeper must not start when both retention windows are disabled")
	}
}

func TestStartRetentionSweeperIdempotentAndStopsOnCancel(t *testing.T) {
	retentionSweepMu.Lock()
	retentionSweepStarted = false
	retentionSweepMu.Unlock()
	oldInterval := retentionSweepInterval
	retentionSweepInterval = time.Hour
	t.Cleanup(func() { retentionSweepInterval = oldInterval })

	service := NewService(NewInMemoryRepository(), nil, NewInMemoryModerationStore(), nil)
	sweepCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.StartRetentionSweeper(sweepCtx, 365, 365)
	service.StartRetentionSweeper(sweepCtx, 365, 365)
	retentionSweepMu.Lock()
	started := retentionSweepStarted
	retentionSweepMu.Unlock()
	if !started {
		t.Fatal("expected retention sweeper started flag to be set")
	}
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		retentionSweepMu.Lock()
		started = retentionSweepStarted
		retentionSweepMu.Unlock()
		if !started {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected retention sweeper flag to reset after ctx cancel")
}
