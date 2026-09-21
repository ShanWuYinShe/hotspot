// Package rbac 提供契约级的守卫模型基础设施（spec/api_spec_authz.md）：
// 群角色层级、群管理守卫中间件工厂与平台权限键常量。
//
// 契约要点（spec/api_spec_authz.md）：
//   - /api/v1 客户端：认证档位 + 资源约束（owner/participant/group.member/self）
//   - /api/v1/groups/admin 守卫子树：统一守卫 = 调用者是目标群 owner/admin
//   - /admin/api/v1 节点：required + admin.* 权限键
//   - 平台管理员与群组管理员是完全不同的两套体系，按节点分别判定、互不传递
package rbac

import (
	"context"
)

// GroupRole 群组角色（仅表达群组内角色；与平台管理员是两套体系）。
type GroupRole string

const (
	GroupRoleOwner  GroupRole = "owner"
	GroupRoleAdmin  GroupRole = "admin"
	GroupRoleMember GroupRole = "member"
)

// IsAdminRole 判定角色是否具备群管理能力（owner ⊃ admin ⊃ member 层级守卫）。
func (r GroupRole) IsAdminRole() bool {
	return r == GroupRoleOwner || r == GroupRoleAdmin
}

// IsOwner 判定是否为群主。
func (r GroupRole) IsOwner() bool {
	return r == GroupRoleOwner
}

// GroupRoleResolver 解析当前调用者在目标群的角色。
// 由 groups 模块注入实现（惰性查成员表）；未找到返回 (GroupRole(""), false, nil)。
type GroupRoleResolver interface {
	RoleOf(ctx context.Context, groupID string) (GroupRole, bool, error)
}

// 平台权限键（权威清单见 spec/api_spec_authz.md「平台权限键」；PII 键需显式持有，superadmin 不隐含）。
const (
	PermAdminRbacManage         = "admin.rbac.manage"
	PermAdminReportsRead        = "admin.reports.read"
	PermAdminReportsWrite       = "admin.reports.write"
	PermAdminAppConfigRead      = "admin.app_config.read"
	PermAdminAppConfigWrite     = "admin.app_config.write"
	PermAdminTagsRead           = "admin.tags.read"
	PermAdminTagsWrite          = "admin.tags.write"
	PermAdminAnnouncementsRead  = "admin.announcements.read"
	PermAdminAnnouncementsWrite = "admin.announcements.write"
	PermAdminUsersRead          = "admin.users.read"
	PermAdminUsersReadIP        = "admin.users.read_ip"
	PermAdminUsersWrite         = "admin.users.write"
	PermAdminGroupsRead         = "admin.groups.read"
	PermAdminGroupsWrite        = "admin.groups.write"
	PermAdminAuditRead          = "admin.audit.read"
	PermAdminAuditReadIP        = "admin.audit.read_ip"
	// PermAdminSuperadmin 表示 superadmin（'*'）。
	PermAdminSuperadmin = "*"
)
