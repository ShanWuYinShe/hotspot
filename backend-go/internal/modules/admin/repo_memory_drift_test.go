package admin

import (
	"context"
	"errors"
	"testing"
)

// TestInMemoryHandleReportPendingGuard 审查 P2-16：PG 实现的
// queryAdminHandleReport 带 WHERE status='pending' 守卫，0 行映射
// errReportAlreadyHandled（并发裁决只有一方命中）。内存实现此前任意状态都
// 可重复处理——降级路径下会双重执行解散/封禁副作用，单测也无法覆盖该并发语义。
func TestInMemoryHandleReportPendingGuard(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()

	first, found, err := repo.HandleReport(ctx, "report_1", ReportActionResolve, "ok", &UserInfo{UserID: "admin_1"})
	if err != nil || !found {
		t.Fatalf("first HandleReport: found=%v err=%v", found, err)
	}
	if first.Status != ReportStatusResolved {
		t.Fatalf("status=%s want resolved", first.Status)
	}

	// 第二次裁决：必须返回 errReportAlreadyHandled，且不得覆盖首次裁决结果。
	_, found, err = repo.HandleReport(ctx, "report_1", ReportActionDismiss, "again", &UserInfo{UserID: "admin_2"})
	if !errors.Is(err, errReportAlreadyHandled) {
		t.Fatalf("second HandleReport err=%v want errReportAlreadyHandled", err)
	}
	if found {
		t.Fatal("second HandleReport must report not-found/handled semantics")
	}
	record := repo.reports["report_1"]
	if record.Item.Status != ReportStatusResolved {
		t.Fatalf("first decision overwritten: status=%s", record.Item.Status)
	}
}
