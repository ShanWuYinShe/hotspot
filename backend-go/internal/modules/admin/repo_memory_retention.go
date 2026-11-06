package admin

import (
	"context"
	"time"
)

// DeleteAuditLogsBefore 物理删除超保留期的审计日志（内存实现，契约测试/单测用）。
func (r *InMemoryRepository) DeleteAuditLogsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var deleted int64
	kept := make([]string, 0, len(r.auditOrder))
	for _, logID := range r.auditOrder {
		entry := r.auditLogs[logID]
		if entry.CreatedAt.Before(cutoff) {
			delete(r.auditLogs, logID)
			deleted++
			continue
		}
		kept = append(kept, logID)
	}
	r.auditOrder = kept
	return deleted, nil
}

// DeleteHandledReportsBefore 物理删除超保留期的终态举报（内存实现）。
func (r *InMemoryRepository) DeleteHandledReportsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var deleted int64
	for reportID, record := range r.reports {
		if record.Item.Status != ReportStatusResolved && record.Item.Status != ReportStatusDismissed {
			continue
		}
		if record.Item.HandledAt == nil {
			continue
		}
		handledAt, err := time.Parse(time.RFC3339Nano, *record.Item.HandledAt)
		if err != nil {
			continue
		}
		if handledAt.Before(cutoff) {
			delete(r.reports, reportID)
			deleted++
		}
	}
	return deleted, nil
}
