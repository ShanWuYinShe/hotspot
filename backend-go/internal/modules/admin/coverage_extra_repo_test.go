package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

// 本文件覆盖 PostgresRepository 在「未接数据库」时的优雅降级：所有方法必须
// 返回 persistence.ErrNotImplemented，让模块回落到内存实现，而不是 panic 或
// 误报业务错误。这是 DB 未配置场景（单测/无库部署）的契约。

func TestPostgresRepositoryNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(nil)
	ctx := context.Background()
	expect := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, persistence.ErrNotImplemented) {
			t.Fatalf("%s = %v, want ErrNotImplemented", name, err)
		}
	}

	_, _, err := repo.GetActor(ctx, "user_x")
	expect("GetActor", err)
	_, err = repo.IsAdmin(ctx, "user_x")
	expect("IsAdmin", err)
	_, err = repo.ListPermissions(ctx)
	expect("ListPermissions", err)
	_, err = repo.ListRoles(ctx)
	expect("ListRoles", err)
	_, _, err = repo.GetRole(ctx, "role_x")
	expect("GetRole", err)
	_, err = repo.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: "r"})
	expect("CreateRole", err)
	err = repo.UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: "role_x"})
	expect("UpdateRole", err)
	err = repo.DeleteRole(ctx, "role_x")
	expect("DeleteRole", err)
	err = repo.SetRolePermissions(ctx, "role_x", nil)
	expect("SetRolePermissions", err)
	_, err = repo.ListAdmins(ctx, PaginationParams{})
	expect("ListAdmins", err)
	err = repo.GrantAdmin(ctx, AdminGrantUpsert{UserID: "user_x"})
	expect("GrantAdmin", err)
	err = repo.RevokeAdmin(ctx, "user_x")
	expect("RevokeAdmin", err)
	_, err = repo.SearchUsers(ctx, AdminUserSearchRequest{})
	expect("SearchUsers", err)
	_, _, err = repo.GetUser(ctx, "user_x")
	expect("GetUser", err)
	_, _, err = repo.GetUserBanState(ctx, "user_x")
	expect("GetUserBanState", err)
	_, err = repo.IsBanned(ctx, "user_x")
	expect("IsBanned", err)
	_, err = repo.ListUserIDsAfter(ctx, "", 10)
	expect("ListUserIDsAfter", err)
	_, err = repo.SearchGroups(ctx, AdminGroupSearchRequest{})
	expect("SearchGroups", err)
	_, _, err = repo.GetGroup(ctx, "group_x")
	expect("GetGroup", err)
	_, err = repo.ListGroupMemberIDs(ctx, "group_x")
	expect("ListGroupMemberIDs", err)
	_, err = repo.SearchReports(ctx, AdminReportListRequest{})
	expect("SearchReports", err)
	_, _, err = repo.GetReportHandleRecord(ctx, "report_x")
	expect("GetReportHandleRecord", err)
	_, err = repo.ListAuditLogs(ctx, AuditLogFilter{}, false)
	expect("ListAuditLogs", err)
	err = repo.AppendAuditLog(ctx, AuditLogEntry{Action: AuditActionRoleCreate})
	expect("AppendAuditLog", err)
	_, err = repo.DeleteAuditLogsBefore(ctx, time.Now().UTC(), retentionDeleteBatchSize)
	expect("DeleteAuditLogsBefore", err)
	_, err = repo.StatsGeo(ctx, geoStatsQuery{})
	expect("StatsGeo", err)
	_, err = repo.StatsGeoDrilldown(ctx, geoDrilldownQuery{})
	expect("StatsGeoDrilldown", err)
}

func TestPostgresRepositoryTxQueryableFallsBackToContext(t *testing.T) {
	repo := NewPostgresRepository(nil)
	queryable := &adminFullStub{}
	ctx := persistence.WithTx(context.Background(), queryable, nil)

	got, ok := repo.queryable(ctx)
	if !ok || got == nil {
		t.Fatalf("queryable(ctx) = (%#v, %v), want tx queryable", got, ok)
	}
	txGot, txOK := repo.txQueryable(ctx)
	if !txOK || txGot == nil {
		t.Fatalf("txQueryable(ctx) = (%#v, %v), want tx queryable", txGot, txOK)
	}
}
