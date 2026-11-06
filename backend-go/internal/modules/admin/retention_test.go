package admin

import (
	"context"
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
	service := NewService(repo, nil, nil, nil)
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

func TestStartRetentionSweeperDisabledWhenBothRetentionZero(t *testing.T) {
	retentionSweepMu.Lock()
	retentionSweepStarted = false
	retentionSweepMu.Unlock()

	service := NewService(NewInMemoryRepository(), nil, nil, nil)
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

	service := NewService(NewInMemoryRepository(), nil, nil, nil)
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
