package admin

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestInMemoryGetRoleDirectLookup 防回归（审查 P2-23）：角色写操作的审计快照
// 改用 GetRole 按 role_id 直查，不再 ListRoles 全量扫描；内存实现须与 PG
// 同口径（未删除、权限排序去重）。
func TestInMemoryGetRoleDirectLookup(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	created, err := repo.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: "Moderator", Description: "mod"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if err := repo.SetRolePermissions(ctx, created.RoleID, []string{"admin.users.read", "admin.groups.read"}); err != nil {
		t.Fatalf("SetRolePermissions: %v", err)
	}
	got, found, err := repo.GetRole(ctx, created.RoleID)
	if err != nil || !found {
		t.Fatalf("GetRole: found=%v err=%v", found, err)
	}
	if got.RoleID != created.RoleID || got.Name != "Moderator" {
		t.Fatalf("unexpected role: %#v", got)
	}
	if len(got.Permissions) != 2 || got.Permissions[0] != "admin.groups.read" || got.Permissions[1] != "admin.users.read" {
		t.Fatalf("permissions must be sorted/unique: %#v", got.Permissions)
	}
	// 不存在 → found=false。
	if _, found, err := repo.GetRole(ctx, "role_missing"); err != nil || found {
		t.Fatalf("missing role: found=%v err=%v", found, err)
	}
	// 删除后 → found=false（与 ListRoles 的 deleted_at IS NULL 过滤一致）。
	if err := repo.DeleteRole(ctx, created.RoleID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if _, found, err := repo.GetRole(ctx, created.RoleID); err != nil || found {
		t.Fatalf("deleted role: found=%v err=%v", found, err)
	}
}

// TestPostgresGetRoleDirectLookup PG 实现：复用 getRoleByID（单行 + 权限批量），
// 已软删记录按不存在处理。
func TestPostgresGetRoleDirectLookup(t *testing.T) {
	backend := &adminFullStub{
		queryRow: map[string][]any{
			queryRoleByID: {"role_1", "Moderator", "mod", aTestTime, aTestTime, false},
		},
		queryRows: map[string][][]any{
			queryRolePermissions: {{"admin.users.read"}, {"admin.groups.read"}},
		},
	}
	repo := newAdminFullRepo(backend)
	got, found, err := repo.GetRole(context.Background(), "role_1")
	if err != nil || !found {
		t.Fatalf("GetRole: found=%v err=%v", found, err)
	}
	if got.RoleID != "role_1" || len(got.Permissions) != 2 || got.Permissions[0] != "admin.groups.read" {
		t.Fatalf("unexpected role: %#v", got)
	}
	// 0 行 → found=false, nil error（handler 走「角色不存在」而不是 10600）。
	missing := &adminFullStub{queryRowErr: map[string]error{queryRoleByID: pgx.ErrNoRows}}
	if _, found, err := newAdminFullRepo(missing).GetRole(context.Background(), "role_missing"); err != nil || found {
		t.Fatalf("missing role: found=%v err=%v", found, err)
	}
	// 已软删 → found=false。
	deleted := &adminFullStub{queryRow: map[string][]any{
		queryRoleByID: {"role_1", "Moderator", "", aTestTime, aTestTime, true},
	}}
	if _, found, err := newAdminFullRepo(deleted).GetRole(context.Background(), "role_1"); err != nil || found {
		t.Fatalf("soft-deleted role: found=%v err=%v", found, err)
	}
}
