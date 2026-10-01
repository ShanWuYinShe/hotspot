package rbac

import (
	"context"
	"testing"
)

type mockRoleResolver struct {
	role  GroupRole
	found bool
	err   error
}

func (m mockRoleResolver) RoleOf(_ context.Context, _ string) (GroupRole, bool, error) {
	return m.role, m.found, m.err
}

func TestGroupRole_IsAdminRole(t *testing.T) {
	tests := []struct {
		role GroupRole
		want bool
	}{
		{role: GroupRoleOwner, want: true},
		{role: GroupRoleAdmin, want: true},
		{role: GroupRoleMember, want: false},
		{role: GroupRole("guest"), want: false},
		{role: GroupRole(""), want: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsAdminRole(); got != tt.want {
				t.Fatalf("GroupRole(%q).IsAdminRole() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestGroupRole_IsOwner(t *testing.T) {
	tests := []struct {
		role GroupRole
		want bool
	}{
		{role: GroupRoleOwner, want: true},
		{role: GroupRoleAdmin, want: false},
		{role: GroupRoleMember, want: false},
		{role: GroupRole("guest"), want: false},
		{role: GroupRole(""), want: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsOwner(); got != tt.want {
				t.Fatalf("GroupRole(%q).IsOwner() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestGroupRoleResolverInterface(t *testing.T) {
	var resolver GroupRoleResolver = mockRoleResolver{
		role:  GroupRoleOwner,
		found: true,
		err:   nil,
	}

	role, found, err := resolver.RoleOf(context.Background(), "grp-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected found to be true")
	}
	if role != GroupRoleOwner {
		t.Fatalf("expected GroupRoleOwner, got %v", role)
	}
}

func TestPlatformPermissionConstants(t *testing.T) {
	// 验证关键权限常量定义符合 spec/api_spec_authz.md 规范
	expectedPerms := map[string]string{
		"PermAdminRbacManage":         PermAdminRbacManage,
		"PermAdminReportsRead":        PermAdminReportsRead,
		"PermAdminReportsWrite":       PermAdminReportsWrite,
		"PermAdminAppConfigRead":      PermAdminAppConfigRead,
		"PermAdminAppConfigWrite":     PermAdminAppConfigWrite,
		"PermAdminTagsRead":           PermAdminTagsRead,
		"PermAdminTagsWrite":          PermAdminTagsWrite,
		"PermAdminAnnouncementsRead":  PermAdminAnnouncementsRead,
		"PermAdminAnnouncementsWrite": PermAdminAnnouncementsWrite,
		"PermAdminUsersRead":          PermAdminUsersRead,
		"PermAdminUsersReadIP":        PermAdminUsersReadIP,
		"PermAdminUsersWrite":         PermAdminUsersWrite,
		"PermAdminGroupsRead":         PermAdminGroupsRead,
		"PermAdminGroupsWrite":        PermAdminGroupsWrite,
		"PermAdminAuditRead":          PermAdminAuditRead,
		"PermAdminAuditReadIP":        PermAdminAuditReadIP,
		"PermAdminStatsRead":          PermAdminStatsRead,
		"PermAdminSuperadmin":         PermAdminSuperadmin,
	}

	for name, perm := range expectedPerms {
		if perm == "" {
			t.Errorf("permission %s is empty", name)
		}
	}
	if PermAdminSuperadmin != "*" {
		t.Errorf("PermAdminSuperadmin expected '*', got %q", PermAdminSuperadmin)
	}
}
