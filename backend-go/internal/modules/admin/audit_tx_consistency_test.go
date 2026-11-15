package admin

import (
	"context"
	"errors"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/appconfig"
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

// ── 批次 4：标签屏蔽 / 公告 update+delete / RBAC 写路径的审计同事务 ──────────

// moderationRollbackTxRunner 模拟真实事务回滚：回调出错时把内存审核仓储（标签
// 屏蔽状态与公告表）恢复为调用前快照，用于断言「审计失败 → 业务写回滚」。
func moderationRollbackTxRunner(mods *InMemoryModerationStore) func(context.Context, func(context.Context) error) error {
	return func(ctx context.Context, fn func(context.Context) error) error {
		mods.mu.RLock()
		tags := make(map[string]TagRecord, len(mods.tags))
		for name, record := range mods.tags {
			tags[name] = record
		}
		blocked := make(map[string]struct{}, len(mods.blockedTags))
		for name := range mods.blockedTags {
			blocked[name] = struct{}{}
		}
		announcements := make(map[string]systemmodule.AnnouncementRecord, len(mods.announcements))
		for id, record := range mods.announcements {
			announcements[id] = record
		}
		mods.mu.RUnlock()
		err := fn(ctx)
		if err == nil {
			return nil
		}
		mods.mu.Lock()
		mods.tags = tags
		mods.blockedTags = blocked
		mods.announcements = announcements
		mods.mu.Unlock()
		return err
	}
}

// memoryRepositoryRollbackTxRunner 模拟真实事务回滚：回调出错时恢复内存仓储中
// RBAC（角色/管理员授权）与审计表的调用前快照。
func memoryRepositoryRollbackTxRunner(repo *InMemoryRepository) func(context.Context, func(context.Context) error) error {
	return func(ctx context.Context, fn func(context.Context) error) error {
		repo.mu.RLock()
		roles := make(map[string]roleRecord, len(repo.roles))
		for id, record := range repo.roles {
			roles[id] = record
		}
		admins := make(map[string]adminGrantRecord, len(repo.admins))
		for id, record := range repo.admins {
			admins[id] = record
		}
		auditLogs := make(map[string]AuditLogEntry, len(repo.auditLogs))
		for id, entry := range repo.auditLogs {
			auditLogs[id] = entry
		}
		auditOrder := append([]string(nil), repo.auditOrder...)
		deletedRoles := make(map[string]struct{}, len(repo.deletedRoleIDs))
		for id := range repo.deletedRoleIDs {
			deletedRoles[id] = struct{}{}
		}
		revokedAdmins := make(map[string]struct{}, len(repo.revokedAdmins))
		for id := range repo.revokedAdmins {
			revokedAdmins[id] = struct{}{}
		}
		repo.mu.RUnlock()
		err := fn(ctx)
		if err == nil {
			return nil
		}
		repo.mu.Lock()
		repo.roles = roles
		repo.admins = admins
		repo.auditLogs = auditLogs
		repo.auditOrder = auditOrder
		repo.deletedRoleIDs = deletedRoles
		repo.revokedAdmins = revokedAdmins
		repo.mu.Unlock()
		return err
	}
}

// newAuditTxService 构造带可控审计故障与内存事务回滚的 admin Service。
func newAuditTxService(mods ModerationStore, txRunner func(context.Context, func(context.Context) error) error) (*Service, *covRepo) {
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, nil)
	svc.SetTxRunner(txRunner)
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		panic(err)
	}
	return svc, repo
}

func TestSetTagBanAuditFailureRollsBackBan(t *testing.T) {
	mods := NewInMemoryModerationStore()
	svc, repo := newAuditTxService(mods, moderationRollbackTxRunner(mods))

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	req := AdminSetTagBanRequest{Name: "hotspot", Active: true}
	if apiErr, err := svc.SetTagBan(adminCtx("admin1"), req); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort tag ban: apiErr=%v err=%v", apiErr, err)
	}
	// 审计失败 → 屏蔽写回滚：标签仍未屏蔽、无 tag.set_ban 审计。
	if _, blocked := mods.blockedTags["hotspot"]; blocked {
		t.Fatal("tag ban must roll back on audit failure")
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionTagSetBan, "hotspot"); got != 0 {
		t.Fatalf("tag.set_ban audit count after failure = %d, want 0", got)
	}

	// 撤掉审计故障后可安全重试：屏蔽生效且只落一条审计。
	repo.auditErr = nil
	if apiErr, err := svc.SetTagBan(adminCtx("admin1"), req); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	if _, blocked := mods.blockedTags["hotspot"]; !blocked {
		t.Fatal("retry must block tag")
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionTagSetBan, "hotspot"); got != 1 {
		t.Fatalf("tag.set_ban audit count = %d, want 1", got)
	}
}

func TestUpdateAnnouncementAuditFailureRollsBackUpdate(t *testing.T) {
	mods := NewInMemoryModerationStore()
	svc, repo := newAuditTxService(mods, moderationRollbackTxRunner(mods))
	created, apiErr, err := svc.CreateAnnouncement(adminCtx("admin1"), CreateAnnouncementRequest{Title: "原标题", Content: "内容", Severity: AnnounceLevelInfo})
	if apiErr != nil || err != nil {
		t.Fatalf("create announcement: apiErr=%v err=%v", apiErr, err)
	}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	newTitle := "审计失败的新标题"
	req := UpdateAnnouncementRequest{AnnouncementID: created.AnnouncementID, Title: &newTitle}
	if _, apiErr, err := svc.UpdateAnnouncement(adminCtx("admin1"), req); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort update: apiErr=%v err=%v", apiErr, err)
	}
	current, found, getErr := mods.GetAnnouncement(context.Background(), created.AnnouncementID)
	if getErr != nil || !found {
		t.Fatalf("announcement missing: found=%v err=%v", found, getErr)
	}
	if current.Title != "原标题" {
		t.Fatalf("announcement title after audit failure = %q, want 原标题 (must roll back)", current.Title)
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionAnnouncementUpdate, created.AnnouncementID); got != 0 {
		t.Fatalf("announcement.update audit count after failure = %d, want 0", got)
	}

	// 撤掉故障后重试：更新生效且只落一条审计。
	repo.auditErr = nil
	if _, apiErr, err := svc.UpdateAnnouncement(adminCtx("admin1"), req); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	current, _, getErr = mods.GetAnnouncement(context.Background(), created.AnnouncementID)
	if getErr != nil || current.Title != newTitle {
		t.Fatalf("retry must update title: title=%q err=%v", current.Title, getErr)
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionAnnouncementUpdate, created.AnnouncementID); got != 1 {
		t.Fatalf("announcement.update audit count = %d, want 1", got)
	}
}

func TestDeleteAnnouncementAuditFailureRollsBackDelete(t *testing.T) {
	mods := NewInMemoryModerationStore()
	svc, repo := newAuditTxService(mods, moderationRollbackTxRunner(mods))
	created, apiErr, err := svc.CreateAnnouncement(adminCtx("admin1"), CreateAnnouncementRequest{Title: "待删公告", Content: "内容", Severity: AnnounceLevelInfo})
	if apiErr != nil || err != nil {
		t.Fatalf("create announcement: apiErr=%v err=%v", apiErr, err)
	}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	if apiErr, err := svc.DeleteAnnouncement(adminCtx("admin1"), created.AnnouncementID); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort delete: apiErr=%v err=%v", apiErr, err)
	}
	current, found, getErr := mods.GetAnnouncement(context.Background(), created.AnnouncementID)
	if getErr != nil || !found {
		t.Fatalf("announcement missing: found=%v err=%v", found, getErr)
	}
	if current.IsDeleted() {
		t.Fatal("announcement delete must roll back on audit failure")
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionAnnouncementDelete, created.AnnouncementID); got != 0 {
		t.Fatalf("announcement.delete audit count after failure = %d, want 0", got)
	}

	// 撤掉故障后重试：软删生效且只落一条审计。
	repo.auditErr = nil
	if apiErr, err := svc.DeleteAnnouncement(adminCtx("admin1"), created.AnnouncementID); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	current, _, getErr = mods.GetAnnouncement(context.Background(), created.AnnouncementID)
	if getErr != nil || !current.IsDeleted() {
		t.Fatalf("retry must soft-delete announcement: deleted=%v err=%v", current.IsDeleted(), getErr)
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionAnnouncementDelete, created.AnnouncementID); got != 1 {
		t.Fatalf("announcement.delete audit count = %d, want 1", got)
	}
}

func TestCreateRoleAuditFailureRollsBackCreate(t *testing.T) {
	mods := NewInMemoryModerationStore()
	repoBase := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: repoBase}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, nil)
	svc.SetTxRunner(memoryRepositoryRollbackTxRunner(repoBase))
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	// 内存仓储构造时预置了演示审计行：用基线差值断言，不能直接要求全表为空。
	repoBase.mu.RLock()
	auditBefore := len(repoBase.auditLogs)
	repoBase.mu.RUnlock()

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	req := AdminRbacRoleCreateRequest{Name: "audit_rollback_role"}
	if _, apiErr, err := svc.CreateRole(adminCtx("admin1"), req); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort role create: apiErr=%v err=%v", apiErr, err)
	}
	if roles, _ := repo.ListRoles(context.Background()); len(roles) != 0 {
		t.Fatalf("role create must roll back on audit failure, got %d roles", len(roles))
	}
	repoBase.mu.RLock()
	auditAfter := len(repoBase.auditLogs)
	repoBase.mu.RUnlock()
	if auditAfter != auditBefore {
		t.Fatalf("role.create audit row must roll back: before=%d after=%d", auditBefore, auditAfter)
	}

	repo.auditErr = nil
	item, apiErr, err := svc.CreateRole(adminCtx("admin1"), req)
	if apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	if got := countAuditByAction(repoBase, AuditActionRoleCreate, item.RoleID); got != 1 {
		t.Fatalf("role.create audit count = %d, want 1", got)
	}
}

func TestUpdateRoleAuditFailureRollsBackUpdate(t *testing.T) {
	mods := NewInMemoryModerationStore()
	repoBase := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: repoBase}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, nil)
	svc.SetTxRunner(memoryRepositoryRollbackTxRunner(repoBase))
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	role, err := repoBase.CreateRole(context.Background(), AdminRbacRoleCreateRequest{Name: "old_name"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	newName := "new_name"
	req := AdminRbacRoleUpdateRequest{RoleID: role.RoleID, Name: &newName}
	if apiErr, err := svc.UpdateRole(adminCtx("admin1"), req); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort role update: apiErr=%v err=%v", apiErr, err)
	}
	after, found, getErr := repoBase.GetRole(context.Background(), role.RoleID)
	if getErr != nil || !found || after.Name != "old_name" {
		t.Fatalf("role update must roll back on audit failure: name=%q found=%v err=%v", after.Name, found, getErr)
	}
	if got := countAuditByAction(repoBase, AuditActionRoleUpdate, role.RoleID); got != 0 {
		t.Fatalf("role.update audit count after failure = %d, want 0", got)
	}

	repo.auditErr = nil
	if apiErr, err := svc.UpdateRole(adminCtx("admin1"), req); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	after, _, _ = repoBase.GetRole(context.Background(), role.RoleID)
	if after.Name != newName {
		t.Fatalf("retry must rename role, got %q", after.Name)
	}
	if got := countAuditByAction(repoBase, AuditActionRoleUpdate, role.RoleID); got != 1 {
		t.Fatalf("role.update audit count = %d, want 1", got)
	}
}

func TestDeleteRoleAuditFailureRollsBackDelete(t *testing.T) {
	mods := NewInMemoryModerationStore()
	repoBase := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: repoBase}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, nil)
	svc.SetTxRunner(memoryRepositoryRollbackTxRunner(repoBase))
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	role, err := repoBase.CreateRole(context.Background(), AdminRbacRoleCreateRequest{Name: "delete_me"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	if apiErr, err := svc.DeleteRole(adminCtx("admin1"), role.RoleID); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort role delete: apiErr=%v err=%v", apiErr, err)
	}
	if _, found, getErr := repoBase.GetRole(context.Background(), role.RoleID); getErr != nil || !found {
		t.Fatalf("role delete must roll back on audit failure: found=%v err=%v", found, getErr)
	}
	if got := countAuditByAction(repoBase, AuditActionRoleDelete, role.RoleID); got != 0 {
		t.Fatalf("role.delete audit count after failure = %d, want 0", got)
	}

	repo.auditErr = nil
	if apiErr, err := svc.DeleteRole(adminCtx("admin1"), role.RoleID); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	if _, found, getErr := repoBase.GetRole(context.Background(), role.RoleID); getErr != nil || found {
		t.Fatalf("retry must delete role: found=%v err=%v", found, getErr)
	}
	if got := countAuditByAction(repoBase, AuditActionRoleDelete, role.RoleID); got != 1 {
		t.Fatalf("role.delete audit count = %d, want 1", got)
	}
}

func TestSetRolePermissionsAuditFailureRollsBackPermissions(t *testing.T) {
	mods := NewInMemoryModerationStore()
	repoBase := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: repoBase}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, nil)
	svc.SetTxRunner(memoryRepositoryRollbackTxRunner(repoBase))
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	role, err := repoBase.CreateRole(context.Background(), AdminRbacRoleCreateRequest{Name: "perm_role"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := repoBase.SetRolePermissions(context.Background(), role.RoleID, []string{permissionTagsRead}); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	req := AdminRbacRoleSetPermissionsRequest{RoleID: role.RoleID, PermissionKeys: []string{permissionTagsWrite}}
	if apiErr, err := svc.SetRolePermissions(adminCtx("admin1"), req); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort set permissions: apiErr=%v err=%v", apiErr, err)
	}
	after, found, getErr := repoBase.GetRole(context.Background(), role.RoleID)
	if getErr != nil || !found {
		t.Fatalf("role missing: found=%v err=%v", found, getErr)
	}
	if len(after.Permissions) != 1 || after.Permissions[0] != permissionTagsRead {
		t.Fatalf("permissions must roll back on audit failure, got %v", after.Permissions)
	}
	if got := countAuditByAction(repoBase, AuditActionRoleSetPermissions, role.RoleID); got != 0 {
		t.Fatalf("role.set_permissions audit count after failure = %d, want 0", got)
	}

	repo.auditErr = nil
	if apiErr, err := svc.SetRolePermissions(adminCtx("admin1"), req); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	after, _, _ = repoBase.GetRole(context.Background(), role.RoleID)
	if len(after.Permissions) != 1 || after.Permissions[0] != permissionTagsWrite {
		t.Fatalf("retry must replace permissions, got %v", after.Permissions)
	}
	if got := countAuditByAction(repoBase, AuditActionRoleSetPermissions, role.RoleID); got != 1 {
		t.Fatalf("role.set_permissions audit count = %d, want 1", got)
	}
}

func TestGrantAdminAuditFailureRollsBackGrant(t *testing.T) {
	mods := NewInMemoryModerationStore()
	repoBase := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: repoBase}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, nil)
	svc.SetTxRunner(memoryRepositoryRollbackTxRunner(repoBase))
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	req := AdminRbacGrantAdminRequest{UserID: "user_bob", AwardAllPermissions: true}
	if apiErr, err := svc.GrantAdmin(adminCtx("admin1"), req); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort grant: apiErr=%v err=%v", apiErr, err)
	}
	if _, found, getErr := repoBase.GetActor(context.Background(), "user_bob"); getErr != nil || found {
		t.Fatalf("grant must roll back on audit failure: found=%v err=%v", found, getErr)
	}
	if got := countAuditByAction(repoBase, AuditActionAdminAward, "user_bob"); got != 0 {
		t.Fatalf("admin.award audit count after failure = %d, want 0", got)
	}

	repo.auditErr = nil
	if apiErr, err := svc.GrantAdmin(adminCtx("admin1"), req); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	if _, found, getErr := repoBase.GetActor(context.Background(), "user_bob"); getErr != nil || !found {
		t.Fatalf("retry must grant admin: found=%v err=%v", found, getErr)
	}
	if got := countAuditByAction(repoBase, AuditActionAdminAward, "user_bob"); got != 1 {
		t.Fatalf("admin.award audit count = %d, want 1", got)
	}
}

func TestRevokeAdminAuditFailureRollsBackRevoke(t *testing.T) {
	mods := NewInMemoryModerationStore()
	repoBase := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: repoBase}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, nil)
	svc.SetTxRunner(memoryRepositoryRollbackTxRunner(repoBase))
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "user_bob", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant bob: %v", err)
	}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	if apiErr, err := svc.RevokeAdmin(adminCtx("admin1"), "user_bob"); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort revoke: apiErr=%v err=%v", apiErr, err)
	}
	if _, found, getErr := repoBase.GetActor(context.Background(), "user_bob"); getErr != nil || !found {
		t.Fatalf("revoke must roll back on audit failure: found=%v err=%v", found, getErr)
	}
	if got := countAuditByAction(repoBase, AuditActionAdminRevoke, "user_bob"); got != 0 {
		t.Fatalf("admin.revoke audit count after failure = %d, want 0", got)
	}

	repo.auditErr = nil
	if apiErr, err := svc.RevokeAdmin(adminCtx("admin1"), "user_bob"); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	if _, found, getErr := repoBase.GetActor(context.Background(), "user_bob"); getErr != nil || found {
		t.Fatalf("retry must revoke admin: found=%v err=%v", found, getErr)
	}
	if got := countAuditByAction(repoBase, AuditActionAdminRevoke, "user_bob"); got != 1 {
		t.Fatalf("admin.revoke audit count = %d, want 1", got)
	}
}

// roleReadSequenceRepo 让第二次 GetRole（UpdateRole 更新后的回读）返回错误或
// not found：覆盖「同事务内更新后回读异常必须回滚、不写不一致审计」两条分支。
type roleReadSequenceRepo struct {
	*InMemoryRepository
	calls      int
	failOnCall int
	hideOnCall int
}

func (r *roleReadSequenceRepo) GetRole(ctx context.Context, roleID string) (AdminRoleItem, bool, error) {
	r.calls++
	if r.failOnCall > 0 && r.calls == r.failOnCall {
		return AdminRoleItem{}, false, errors.New("injected get role failure")
	}
	if r.hideOnCall > 0 && r.calls == r.hideOnCall {
		return AdminRoleItem{}, false, nil
	}
	return r.InMemoryRepository.GetRole(ctx, roleID)
}

func TestUpdateRolePostReadFailureAbortsWithoutAudit(t *testing.T) {
	for _, tc := range []struct {
		name string
		repo *roleReadSequenceRepo
	}{
		{name: "read_error", repo: &roleReadSequenceRepo{failOnCall: 2}},
		{name: "vanished", repo: &roleReadSequenceRepo{hideOnCall: 2}},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.repo.InMemoryRepository = NewInMemoryRepository()
			svc := NewService(tc.repo, &stubProvider{config: defaultConfig()}, NewInMemoryModerationStore(), nil)
			if err := tc.repo.InMemoryRepository.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
				t.Fatalf("grant admin: %v", err)
			}
			role, err := tc.repo.InMemoryRepository.CreateRole(context.Background(), AdminRbacRoleCreateRequest{Name: "seq_role"})
			if err != nil {
				t.Fatalf("create role: %v", err)
			}
			name := "seq_role_new"
			if apiErr, err := svc.UpdateRole(adminCtx("admin1"), AdminRbacRoleUpdateRequest{RoleID: role.RoleID, Name: &name}); err == nil || apiErr != nil {
				t.Fatalf("post-update read anomaly must abort: apiErr=%v err=%v", apiErr, err)
			}
			if got := countAuditByAction(tc.repo.InMemoryRepository, AuditActionRoleUpdate, role.RoleID); got != 0 {
				t.Fatalf("role.update audit count after post-read anomaly = %d, want 0", got)
			}
		})
	}
}

// ── 批次 5：config.update 审计同事务 ─────────────────────────────────────────
//
// 第 81 轮审计批次 4 残留项：UpdateAppConfig 此前「先写库、事务外写审计」，
// 审计失败时配置已生效。现在「读前态 → 写配置 → config.update 审计」包进同一
// 事务（platform/appconfig 的 Update 已按 ctx 复用事务连接）；审计写失败整体
// 回滚，管理员可重试。真实事务回滚语义由真实库 IT 覆盖（audit_tx_it_test.go），
// 这里用快照 TxRunner 模拟回滚，锁定错误出口与重试语义。

// appConfigSnapshotSource 是可控 MutableSource 替身：记录当前配置，供单测
// 模拟事务回滚时恢复。
type appConfigSnapshotSource struct {
	cfg appconfig.AppConfig
}

func (s *appConfigSnapshotSource) Get(context.Context) appconfig.AppConfig { return s.cfg }

func (s *appConfigSnapshotSource) Update(_ context.Context, apply func(appconfig.AppConfig) appconfig.AppConfig) (appconfig.AppConfig, error) {
	s.cfg = apply(s.cfg)
	return s.cfg, nil
}

// appConfigRollbackTxRunner 模拟真实事务回滚：回调出错时把配置源恢复为调用前
// 快照（appconfig 的 PersistenceSource 在真实事务里由数据库回滚 + aftercommit
// 延迟快照共同保证同一语义）。
func appConfigRollbackTxRunner(source *appConfigSnapshotSource) func(context.Context, func(context.Context) error) error {
	return func(ctx context.Context, fn func(context.Context) error) error {
		saved := source.cfg
		if err := fn(ctx); err != nil {
			source.cfg = saved
			return err
		}
		return nil
	}
}

func TestUpdateAppConfigAuditFailureRollsBackConfig(t *testing.T) {
	source := &appConfigSnapshotSource{}
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := NewService(repo, NewAppConfigProvider(source), NewInMemoryModerationStore(), nil)
	svc.SetTxRunner(appConfigRollbackTxRunner(source))
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	ctx := adminCtx("admin1")
	oldLimit := source.Get(context.Background()).Limits.MaxGroupMembers
	newLimit := oldLimit + 5
	req := UpdateAppConfigRequest{Limits: &PartialAppLimits{MaxGroupMembers: intPtr(newLimit)}}

	auditErr := errors.New("audit insert failed")
	repo.auditErr = auditErr
	if apiErr, err := svc.UpdateAppConfig(ctx, req); !errors.Is(err, auditErr) || apiErr != nil {
		t.Fatalf("audit failure must abort config update: apiErr=%v err=%v", apiErr, err)
	}
	// 审计失败 → 配置写回滚（库内实际生效值未变），且无 config.update 审计。
	if got := source.Get(context.Background()).Limits.MaxGroupMembers; got != oldLimit {
		t.Fatalf("config must roll back on audit failure: max_group_members=%d, want %d", got, oldLimit)
	}
	if cfg, apiErr, err := svc.GetAppConfig(ctx); err != nil || apiErr != nil || cfg.Limits.MaxGroupMembers != oldLimit {
		t.Fatalf("GetAppConfig after audit failure = %#v, apiErr=%v err=%v", cfg.Limits, apiErr, err)
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionConfigUpdate, "app_config"); got != 0 {
		t.Fatalf("config.update audit count after failure = %d, want 0", got)
	}

	// 撤掉审计故障后重试成功：配置生效，且只落一条 config.update 审计。
	repo.auditErr = nil
	if apiErr, err := svc.UpdateAppConfig(ctx, req); apiErr != nil || err != nil {
		t.Fatalf("retry after audit recovery: apiErr=%v err=%v", apiErr, err)
	}
	if got := source.Get(context.Background()).Limits.MaxGroupMembers; got != newLimit {
		t.Fatalf("retry must apply config: max_group_members=%d, want %d", got, newLimit)
	}
	if got := countAuditByAction(repo.InMemoryRepository, AuditActionConfigUpdate, "app_config"); got != 1 {
		t.Fatalf("config.update audit count = %d, want 1", got)
	}
	// before/after 口径保持：审计 details 的 old_value/new_value 是生效前后整份配置。
	repo.mu.RLock()
	var entry AuditLogEntry
	for _, candidate := range repo.auditLogs {
		if candidate.Action == AuditActionConfigUpdate && candidate.TargetID == "app_config" {
			entry = candidate
		}
	}
	repo.mu.RUnlock()
	oldValue, oldOK := entry.Details["old_value"].(AdminAppConfig)
	newValue, newOK := entry.Details["new_value"].(AdminAppConfig)
	if !oldOK || !newOK {
		t.Fatalf("audit details old/new value types = %T/%T", entry.Details["old_value"], entry.Details["new_value"])
	}
	if oldValue.Limits.MaxGroupMembers != oldLimit || newValue.Limits.MaxGroupMembers != newLimit {
		t.Fatalf("audit details limits old/new = %d/%d, want %d/%d",
			oldValue.Limits.MaxGroupMembers, newValue.Limits.MaxGroupMembers, oldLimit, newLimit)
	}
}
