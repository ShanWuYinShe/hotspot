package admin

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	reportsmodule "hotspot/backend-go/internal/modules/reports"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// failingAuditRepository 在真实 Postgres 仓储之上让审计写入失败：用于验证
// 裁决事务的原子性（审计写失败 → 事务内已执行的举报终态 UPDATE 一并回滚）。
type failingAuditRepository struct {
	*PostgresRepository
	err error
}

func (r *failingAuditRepository) AppendAuditLog(context.Context, AuditLogEntry) error {
	return r.err
}

// TestPostgresIT_ReportJudgeAuditInDecisionTx 真实库验证第 81 轮审计修复的
// 原子性（临时库由外部脚本自建自删，测试只读写自己种下的数据）：
//  1. 审计成功时与裁决同事务提交——举报终态与 report.judge 审计同时可见；
//  2. 审计写入失败时裁决事务整体回滚——举报保持 pending（无幻影终态）、
//     审计无落库；撤掉故障后同一裁决重试成功，且只留一条审计。
func TestPostgresIT_ReportJudgeAuditInDecisionTx(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	repo := NewPostgresRepository(factory)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	adminID := testdb.InsertUser(t, factory, "it_audit_admin_"+suffix, "ItAuditAdmin")
	reporterID := testdb.InsertUser(t, factory, "it_audit_reporter_"+suffix, "ItAuditReporter")
	// uq_reports_creator_target 对同人同目标全状态唯一，成功/失败两条举报
	// 必须用不同目标用户。
	targetOKID := testdb.InsertUser(t, factory, "it_audit_target_ok_"+suffix, "ItAuditTargetOK")
	targetFailID := testdb.InsertUser(t, factory, "it_audit_target_fail_"+suffix, "ItAuditTargetFail")
	// 本用例为了验证多事务提交语义必须真实提交数据；共享库上的其它包 IT
	// （如 reports 的保留期删除断言会把「已处理举报」一并计数）会被残留行污染，
	// 故测试结束（含失败路径）清理本用例创建的全部行。
	// admin_audit_logs 无外键，先删；users 级联清理 admin_users / reports 等。
	userIDs := []string{adminID, reporterID, targetOKID, targetFailID}
	reportIDs := []string{"it_audit_ok_" + suffix, "it_audit_fail_" + suffix}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if err := qdb.Exec(cleanupCtx,
			"DELETE FROM admin_audit_logs WHERE target_id = ANY($1) OR operator_id = ANY($2)",
			reportIDs, userIDs); err != nil {
			t.Logf("cleanup it audit logs: %v", err)
		}
		if err := qdb.Exec(cleanupCtx, "DELETE FROM users WHERE user_id = ANY($1)", userIDs); err != nil {
			t.Logf("cleanup it audit users: %v", err)
		}
	})
	if err := repo.GrantAdmin(ctx, AdminGrantUpsert{UserID: adminID, GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	insertReport := func(reportID, targetUserID string) {
		t.Helper()
		if err := qdb.Exec(ctx,
			"INSERT INTO reports (report_id, creator_user_id, target_type, target_id, reason, status, created_at) VALUES ($1, $2, 'user', $3, 'spam', 'pending', $4)",
			reportID, reporterID, targetUserID, time.Now().UTC()); err != nil {
			t.Fatalf("insert report %s: %v", reportID, err)
		}
	}
	reportStatus := func(reportID string) ReportStatus {
		t.Helper()
		var status string
		if err := qdb.QueryRow(ctx, "SELECT status FROM reports WHERE report_id = $1", reportID).Scan(&status); err != nil {
			t.Fatalf("query report %s status: %v", reportID, err)
		}
		return ReportStatus(status)
	}
	judgeAuditCount := func(reportID string) int {
		t.Helper()
		var count int
		if err := qdb.QueryRow(ctx, "SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'report.judge' AND target_id = $1", reportID).Scan(&count); err != nil {
			t.Fatalf("query report %s judge audit: %v", reportID, err)
		}
		return count
	}
	moderationStore := NewPersistenceModerationStore(factory, nil, nil)
	reportAdmin := reportsmodule.NewModule(factory).Service
	newDecider := func(repository Repository) *Service {
		svc := NewService(repository, &stubProvider{config: defaultConfig()}, moderationStore, nil)
		svc.SetTxRunner(persistence.NewTxRunner(factory))
		svc.SetReportAdminStore(reportAdmin)
		return svc
	}

	// 1) 审计成功：终态与审计同事务提交后均可见。
	successID := "it_audit_ok_" + suffix
	insertReport(successID, targetOKID)
	req := JudgeAdminReportRequest{ReportID: successID, Action: ReportActionResolve, Note: "confirmed"}
	if apiErr, err := newDecider(repo).HandleReport(adminCtx(adminID), req); apiErr != nil || err != nil {
		t.Fatalf("handle report success path: apiErr=%v err=%v", apiErr, err)
	}
	if got := reportStatus(successID); got != ReportStatusResolved {
		t.Fatalf("report status = %q, want resolved", got)
	}
	if got := judgeAuditCount(successID); got != 1 {
		t.Fatalf("report.judge audit count = %d, want 1", got)
	}

	// 2) 审计失败：举报终态 UPDATE 已执行但必须随事务回滚，举报仍 pending。
	failID := "it_audit_fail_" + suffix
	insertReport(failID, targetFailID)
	auditErr := errors.New("it: audit insert forced failure")
	failing := &failingAuditRepository{PostgresRepository: repo, err: auditErr}
	failReq := JudgeAdminReportRequest{ReportID: failID, Action: ReportActionResolve, Note: "confirmed"}
	if _, err := newDecider(failing).HandleReport(adminCtx(adminID), failReq); !errors.Is(err, auditErr) {
		t.Fatalf("audit failure must surface: %v", err)
	}
	if got := reportStatus(failID); got != ReportStatusPending {
		t.Fatalf("report status after audit failure = %q, want pending (transaction must roll back)", got)
	}
	if got := judgeAuditCount(failID); got != 0 {
		t.Fatalf("report.judge audit count after failure = %d, want 0", got)
	}

	// 3) 管理员撤掉故障后同一裁决可安全重试：成功且只有一条审计。
	if apiErr, err := newDecider(repo).HandleReport(adminCtx(adminID), failReq); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	if got := reportStatus(failID); got != ReportStatusResolved {
		t.Fatalf("report status after retry = %q, want resolved", got)
	}
	if got := judgeAuditCount(failID); got != 1 {
		t.Fatalf("report.judge audit count after retry = %d, want 1", got)
	}
}
