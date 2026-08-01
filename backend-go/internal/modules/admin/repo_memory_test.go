package admin

import (
	"context"
	"testing"
)

// TestInMemoryDeleteRole_ClearsAssignmentsAcrossAdmins verifies the in-memory
// DeleteRole removes the role from every admin it was granted to. The postgres
// path now wraps the assignment clear and role soft-delete in a single
// transaction (so a failure can't leave assignments pointing at a deleted
// role); this guards the tx-transparent in-memory mirror.
func TestInMemoryDeleteRole_ClearsAssignmentsAcrossAdmins(t *testing.T) {
	repo := NewInMemoryRepository()
	role, err := repo.CreateRole(context.Background(), AdminRbacRoleCreateRequest{Name: "Moderator"})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{
		UserID:  "admin_1",
		RoleIDs: []string{role.RoleID},
	}); err != nil {
		t.Fatalf("GrantAdmin failed: %v", err)
	}

	if err := repo.DeleteRole(context.Background(), role.RoleID); err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	page, err := repo.ListAdmins(context.Background(), PaginationParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListAdmins failed: %v", err)
	}
	for _, op := range page.Items {
		for _, r := range op.Roles {
			if r.RoleID == role.RoleID {
				t.Fatalf("deleted role %s still assigned to admin %s", role.RoleID, op.User.UserID)
			}
		}
	}
}

// TestInMemorySetRolePermissions_ReplacesWholesale verifies the in-memory
// SetRolePermissions fully replaces the permission set (the postgres path does
// delete-then-batch-insert inside its transaction for the same effect).
func TestInMemorySetRolePermissions_ReplacesWholesale(t *testing.T) {
	repo := NewInMemoryRepository()
	role, err := repo.CreateRole(context.Background(), AdminRbacRoleCreateRequest{Name: "Editor"})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if err := repo.SetRolePermissions(context.Background(), role.RoleID, []string{"read", "write"}); err != nil {
		t.Fatalf("first SetRolePermissions failed: %v", err)
	}
	if err := repo.SetRolePermissions(context.Background(), role.RoleID, []string{"delete"}); err != nil {
		t.Fatalf("second SetRolePermissions failed: %v", err)
	}
	roles, err := repo.ListRoles(context.Background())
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	for _, r := range roles {
		if r.RoleID == role.RoleID {
			if len(r.Permissions) != 1 || r.Permissions[0] != "delete" {
				t.Fatalf("expected permissions fully replaced to [delete], got %#v", r.Permissions)
			}
			return
		}
	}
	t.Fatalf("role %s not found in ListRoles after SetRolePermissions", role.RoleID)
}
