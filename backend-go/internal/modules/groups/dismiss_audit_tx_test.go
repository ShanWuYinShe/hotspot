package groups

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/testdb"

	"github.com/jackc/pgx/v5"
)

// 本文件锁定第 81 轮审计批次 4 残留项的修复口径：
//   - group.dismiss 审计与解散业务写同一事务提交；审计写失败整体回滚并上抛
//     错误（owner 自助解散与平台审核解散两条链路），重试成功后只留一条审计；
//   - 回滚时不产生幻影 GroupDismissed 广播；
//   - 真实库 IT 验证提交可见性与审计失败回滚（临时库自建自删，提交型数据
//     t.Cleanup 清理）。

// dismissStateRepo 记录「解散业务写是否已执行」的仓储替身：由 fake TxRunner
// 在回调出错时恢复标记，模拟真实事务回滚。
type dismissStateRepo struct {
	*stubGroupsRepo
	dismissed bool
}

func (r *dismissStateRepo) Dismiss(context.Context, string) error {
	if r.dismissErr != nil {
		return r.dismissErr
	}
	r.dismissed = true
	return nil
}

func (r *dismissStateRepo) DismissAsAdmin(context.Context, string) (bool, error) {
	if r.dismissErr != nil {
		return false, r.dismissErr
	}
	r.dismissed = true
	return r.dismissAdminHit, nil
}

// dismissRollbackTxRunner 模拟真实事务回滚：回调出错时恢复群解散标记。
func dismissRollbackTxRunner(repo *dismissStateRepo) func(context.Context, func(context.Context) error) error {
	return func(ctx context.Context, fn func(context.Context) error) error {
		saved := repo.dismissed
		if err := fn(ctx); err != nil {
			repo.dismissed = saved
			return err
		}
		return nil
	}
}

// dismissAuditRecorder 记录审计调用并可注入失败；succeeded 只统计成功的审计
// 写入（等价真实事务里「审计 INSERT 随回滚消失」）。
type dismissAuditRecorder struct {
	err       error
	calls     int
	succeeded int
}

func (r *dismissAuditRecorder) AppendGroupDismissAudit(context.Context, string, string, string, string) error {
	r.calls++
	if r.err != nil {
		return r.err
	}
	r.succeeded++
	return nil
}

func TestServiceDismissAuditFailureRollsBackDismiss(t *testing.T) {
	repo := &dismissStateRepo{stubGroupsRepo: &stubGroupsRepo{getResult: ownerGroup(), getFound: true}}
	svc := NewServiceWithRepo(repo)
	svc.SetTxRunner(dismissRollbackTxRunner(repo))
	audit := &dismissAuditRecorder{err: errors.New("audit down")}
	svc.SetGroupDismissAuditWriter(audit)
	hub := withTestHub(t)
	sink := &sinkStub{}
	hub.SetGroupSubscriptions([]string{"g1"}, sink)

	if err := svc.Dismiss(grpCtx("u1"), DismissGroupRequest{GroupID: "g1"}); !errors.Is(err, audit.err) {
		t.Fatalf("audit failure must propagate: %v", err)
	}
	if repo.dismissed {
		t.Fatal("owner dismiss must roll back on audit failure")
	}
	if len(sink.events) != 0 {
		t.Fatalf("rolled-back dismiss must not broadcast, got %#v", sink.events)
	}
	if audit.calls != 1 || audit.succeeded != 0 {
		t.Fatalf("audit calls=%d succeeded=%d, want 1/0", audit.calls, audit.succeeded)
	}

	// 撤掉审计故障后重试成功：群已解散、只落一条审计、广播一次。
	audit.err = nil
	if err := svc.Dismiss(grpCtx("u1"), DismissGroupRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("retry after audit recovery: %v", err)
	}
	if !repo.dismissed {
		t.Fatal("retry must dismiss group")
	}
	if audit.calls != 2 || audit.succeeded != 1 {
		t.Fatalf("audit calls=%d succeeded=%d, want 2/1 (failed attempt must not append)", audit.calls, audit.succeeded)
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeGroupDismissed {
		t.Fatalf("expected one GroupDismissed after retry, got %#v", sink.events)
	}
}

func TestServiceDismissAsAdminAuditFailureRollsBackDismiss(t *testing.T) {
	repo := &dismissStateRepo{stubGroupsRepo: &stubGroupsRepo{getResult: ownerGroup(), getFound: true, dismissAdminHit: true}}
	svc := NewServiceWithRepo(repo)
	svc.SetTxRunner(dismissRollbackTxRunner(repo))
	audit := &dismissAuditRecorder{err: errors.New("audit down")}
	svc.SetGroupDismissAuditWriter(audit)

	if _, err := svc.DismissAsAdmin(grpCtx("admin1"), "admin1", "g1", "policy"); !errors.Is(err, audit.err) {
		t.Fatalf("audit failure must propagate: %v", err)
	}
	if repo.dismissed {
		t.Fatal("platform dismiss must roll back on audit failure")
	}
	if audit.calls != 1 || audit.succeeded != 0 {
		t.Fatalf("audit calls=%d succeeded=%d, want 1/0", audit.calls, audit.succeeded)
	}

	audit.err = nil
	if _, err := svc.DismissAsAdmin(grpCtx("admin1"), "admin1", "g1", "policy"); err != nil {
		t.Fatalf("retry after audit recovery: %v", err)
	}
	if !repo.dismissed {
		t.Fatal("retry must dismiss group")
	}
	if audit.calls != 2 || audit.succeeded != 1 {
		t.Fatalf("audit calls=%d succeeded=%d, want 2/1", audit.calls, audit.succeeded)
	}
}

// itDismissAuditWriter 与生产 admin 的 AppendGroupDismissAudit 同口径：经 ctx
// 事务连接写 admin_audit_logs（ctx 内已有事务时加入同一事务），err 用于注入
// 审计写失败。
type itDismissAuditWriter struct {
	factory *persistence.Factory
	err     error
}

func (w *itDismissAuditWriter) AppendGroupDismissAudit(ctx context.Context, operatorID, groupID, groupName, reason string) error {
	if w.err != nil {
		return w.err
	}
	queryable, ok := persistence.RepoQueryable(ctx, w.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	details := map[string]any{"type": "group.dismiss", "group_id": groupID}
	if reason != "" {
		details["reason"] = reason
	}
	detailsJSON, marshalErr := json.Marshal(details)
	if marshalErr != nil {
		return marshalErr
	}
	return queryable.Exec(ctx,
		`INSERT INTO admin_audit_logs (log_id, action, operator_id, operator_name, target_type, target_id, target_name, details_json, created_at)
VALUES ($1, 'group.dismiss', $2, 'ItDismissOperator', 'group', $3, $4, $5::jsonb, NOW())`,
		idgen.MustWithPrefix("audit_it_"), operatorID, groupID, groupName, string(detailsJSON))
}

// TestPostgresIT_DismissAuditInTx 真实库验证解散写与 group.dismiss 审计的原子性
// （临时库自建自删；提交型数据 t.Cleanup 清理，不污染共享库其它包 IT）：
//  1. owner 自助解散：审计失败 → 硬删回滚（群仍在）、无审计；撤掉故障重试成功
//     → 群已删且只留一条审计；
//  2. 平台审核解散：审计失败 → 软删与 pending 作废回滚（deleted_at 仍为 NULL）、
//     无审计；重试成功 → 软删生效且只留一条审计。
func TestPostgresIT_DismissAuditInTx(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()
	repo := NewPostgresRepository(factory)
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	ownerID := testdb.InsertUser(t, factory, "it_dismiss_owner_"+suffix, "ItDismissOwner")
	adminID := testdb.InsertUser(t, factory, "it_dismiss_admin_"+suffix, "ItDismissAdmin")

	createGroup := func() string {
		t.Helper()
		group, err := repo.Create(ctx, ownerID, CreateGroupRequest{
			Name:       "it dismiss " + suffix,
			JoinMethod: GroupJoinMethodDirect,
			Location:   Location{LocationName: "Xuzhou"},
		})
		if err != nil {
			t.Fatalf("create group: %v", err)
		}
		return group.Group.GroupID
	}
	// owner 解散为硬删（groups 行消失）、平台解散为软删（deleted_at 置位）：
	// 失败路径断言「行仍在且未软删」即等价「解散事务已回滚」。
	groupState := func(groupID string) (exists, softDeleted bool) {
		t.Helper()
		var deletedAt *time.Time
		err := qdb.QueryRow(ctx, "SELECT deleted_at FROM groups WHERE group_id = $1", groupID).Scan(&deletedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false
		}
		if err != nil {
			t.Fatalf("query group state: %v", err)
		}
		return true, deletedAt != nil
	}
	auditCount := func(groupID string) int {
		t.Helper()
		var count int
		if err := qdb.QueryRow(ctx,
			"SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'group.dismiss' AND target_id = $1", groupID,
		).Scan(&count); err != nil {
			t.Fatalf("query dismiss audit: %v", err)
		}
		return count
	}

	ownerGroupID := createGroup()
	adminGroupID := createGroup()
	groupIDs := []string{ownerGroupID, adminGroupID}
	userIDs := []string{ownerID, adminID}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if err := qdb.Exec(cleanupCtx,
			"DELETE FROM admin_audit_logs WHERE target_id = ANY($1) OR operator_id = ANY($2)", groupIDs, userIDs); err != nil {
			t.Logf("cleanup it dismiss audit logs: %v", err)
		}
		if err := qdb.Exec(cleanupCtx, "DELETE FROM groups WHERE group_id = ANY($1)", groupIDs); err != nil {
			t.Logf("cleanup it dismiss groups: %v", err)
		}
		if err := qdb.Exec(cleanupCtx, "DELETE FROM users WHERE user_id = ANY($1)", userIDs); err != nil {
			t.Logf("cleanup it dismiss users: %v", err)
		}
	})

	writer := &itDismissAuditWriter{factory: factory}
	svc := NewServiceWithDeps(repo, nil, nil)
	svc.SetTxRunner(persistence.NewTxRunner(factory))
	svc.SetGroupDismissAuditWriter(writer)
	auditErr := errors.New("it: dismiss audit insert forced failure")

	// 1) owner 自助解散：审计失败 → 回滚。
	writer.err = auditErr
	ownerCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: ownerID})
	if err := svc.Dismiss(ownerCtx, DismissGroupRequest{GroupID: ownerGroupID}); !errors.Is(err, auditErr) {
		t.Fatalf("owner dismiss audit failure must surface: %v", err)
	}
	if exists, softDeleted := groupState(ownerGroupID); !exists || softDeleted {
		t.Fatalf("owner dismiss must roll back on audit failure: exists=%v soft_deleted=%v", exists, softDeleted)
	}
	if got := auditCount(ownerGroupID); got != 0 {
		t.Fatalf("owner dismiss audit count after failure = %d, want 0", got)
	}
	// 重试成功：群硬删、审计一条。
	writer.err = nil
	if err := svc.Dismiss(ownerCtx, DismissGroupRequest{GroupID: ownerGroupID}); err != nil {
		t.Fatalf("owner dismiss retry: %v", err)
	}
	if exists, _ := groupState(ownerGroupID); exists {
		t.Fatal("owner dismiss retry must delete group row")
	}
	if got := auditCount(ownerGroupID); got != 1 {
		t.Fatalf("owner dismiss audit count after retry = %d, want 1", got)
	}

	// 2) 平台审核解散：审计失败 → 软删与 pending 作废回滚。
	writer.err = auditErr
	if _, err := svc.DismissAsAdmin(context.Background(), adminID, adminGroupID, "policy"); !errors.Is(err, auditErr) {
		t.Fatalf("platform dismiss audit failure must surface: %v", err)
	}
	if exists, softDeleted := groupState(adminGroupID); !exists || softDeleted {
		t.Fatalf("platform dismiss must roll back on audit failure: exists=%v soft_deleted=%v", exists, softDeleted)
	}
	if got := auditCount(adminGroupID); got != 0 {
		t.Fatalf("platform dismiss audit count after failure = %d, want 0", got)
	}
	// 重试成功：软删生效、审计一条。
	writer.err = nil
	if _, err := svc.DismissAsAdmin(context.Background(), adminID, adminGroupID, "policy"); err != nil {
		t.Fatalf("platform dismiss retry: %v", err)
	}
	if exists, softDeleted := groupState(adminGroupID); !exists || !softDeleted {
		t.Fatalf("platform dismiss retry must soft-delete group: exists=%v soft_deleted=%v", exists, softDeleted)
	}
	if got := auditCount(adminGroupID); got != 1 {
		t.Fatalf("platform dismiss audit count after retry = %d, want 1", got)
	}
}
