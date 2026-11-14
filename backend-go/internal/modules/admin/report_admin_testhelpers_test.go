package admin

import (
	"context"
	"time"
)

// testReportAdminStore 把 reports 模块管理端能力落到 admin 内存仓储（测试专用；
// 生产由 bootstrap 注入 reports.Service，report 表归属 reports 模块，P0-4）。
type testReportAdminStore struct{ store *InMemoryRepository }

func (s testReportAdminStore) MarkHandledAsAdmin(ctx context.Context, reportID, status string, handledAt time.Time, handledByID, note *string) (bool, error) {
	action := ReportActionResolve
	if status == string(ReportStatusDismissed) {
		action = ReportActionDismiss
	}
	var handledBy *UserInfo
	if handledByID != nil {
		handledBy = &UserInfo{UserID: *handledByID, Nickname: *handledByID}
	}
	noteStr := ""
	if note != nil {
		noteStr = *note
	}
	_, ok, err := s.store.HandleReport(ctx, reportID, action, noteStr, handledBy)
	return ok, err
}

func (s testReportAdminStore) DeleteHandledBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	return s.store.DeleteHandledReportsBefore(ctx, cutoff, limit)
}
