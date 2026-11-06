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

		deleted, err := reportRepo.DeleteHandledBefore(ctx, now.AddDate(0, 0, -365))
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
