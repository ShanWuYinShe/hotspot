package admin

import (
	"context"
	"testing"
	"time"

	reportsmodule "hotspot/backend-go/internal/modules/reports"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresIT_DeleteHandledReportsBefore 保留期只清终态举报：pending 与
// 「终态但未超期」的举报必须保留，SQL 口径与内存实现一致。
func TestPostgresIT_DeleteHandledReportsBefore(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		reportRepo := reportsmodule.NewPostgresRepository(txFactory)
		reporterID := testdb.InsertUser(t, txFactory, "retention_reporter", "Reporter")
		handlerID := testdb.InsertUser(t, txFactory, "retention_handler", "Handler")

		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		insert := func(reportID, status string, createdAt, handledAt time.Time) {
			t.Helper()
			var handled any = handledAt
			if handledAt.IsZero() {
				handled = nil
			}
			if err := qdb.Exec(ctx,
				"INSERT INTO reports (report_id, creator_user_id, target_type, target_id, reason, status, created_at, handled_at, handled_by_user_id) VALUES ($1, $2, 'user', $1, 'spam', $3, $4, $5, $6)",
				reportID, reporterID, status, createdAt, handled, handlerID); err != nil {
				t.Fatalf("insert report %s: %v", reportID, err)
			}
		}
		now := time.Now().UTC()
		insert("report_pending_old", "pending", now.AddDate(0, 0, -400), time.Time{})
		insert("report_resolved_old", "resolved", now.AddDate(0, 0, -400), now.AddDate(0, 0, -400))
		insert("report_dismissed_recent", "dismissed", now.AddDate(0, 0, -30), now.AddDate(0, 0, -30))

		deleted, err := reportRepo.DeleteHandledBefore(ctx, now.AddDate(0, 0, -365), retentionDeleteBatchSize)
		if err != nil {
			t.Fatalf("delete handled reports: %v", err)
		}
		if deleted != 1 {
			t.Fatalf("deleted = %d, want 1", deleted)
		}
		for reportID, wantFound := range map[string]bool{
			"report_pending_old":      true,
			"report_resolved_old":     false,
			"report_dismissed_recent": true,
		} {
			_, found, err := repo.GetReportHandleRecord(ctx, reportID)
			if err != nil {
				t.Fatalf("get %s: %v", reportID, err)
			}
			if found != wantFound {
				t.Fatalf("%s found = %v, want %v", reportID, found, wantFound)
			}
		}
	})
}

// TestPostgresIT_DeleteAuditLogsBeforeBatches 真实库覆盖审计日志分批删除语义：
// 单批最多 limit 行、循环直至删空，未超期行保留。TxScope 回滚，不污染共享库。
func TestPostgresIT_DeleteAuditLogsBeforeBatches(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		now := time.Now().UTC()
		old := now.AddDate(0, 0, -400)
		insert := func(logID string, createdAt time.Time) {
			t.Helper()
			if err := qdb.Exec(ctx,
				"INSERT INTO admin_audit_logs (log_id, action, operator_id, operator_name, target_type, target_id, created_at) VALUES ($1, 'announcement.create', 'op1', 'op1', 'announcement', 'a1', $2)",
				logID, createdAt); err != nil {
				t.Fatalf("insert audit log %s: %v", logID, err)
			}
		}
		insert("audit_batch_it_1", old)
		insert("audit_batch_it_2", old)
		insert("audit_batch_it_3", old)
		insert("audit_batch_it_fresh", now)

		cutoff := now.AddDate(0, 0, -365)
		for i, want := range []int64{2, 1, 0} {
			deleted, err := repo.DeleteAuditLogsBefore(ctx, cutoff, 2)
			if err != nil {
				t.Fatalf("batch %d: %v", i, err)
			}
			if deleted != want {
				t.Fatalf("batch %d deleted = %d, want %d", i, deleted, want)
			}
		}
		var remaining int
		if err := qdb.QueryRow(ctx, "SELECT COUNT(*) FROM admin_audit_logs WHERE log_id LIKE 'audit_batch_it_%'").Scan(&remaining); err != nil {
			t.Fatalf("count remaining audit logs: %v", err)
		}
		if remaining != 1 {
			t.Fatalf("remaining audit logs = %d, want 1（仅未超期行保留）", remaining)
		}
	})
}
