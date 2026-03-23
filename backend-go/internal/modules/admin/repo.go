package admin

import "context"

type Repository interface {
	GetActor(context.Context, string) (AdminActor, bool, error)
	ListPermissions(context.Context) ([]AdminPermissionItem, error)
	ListRoles(context.Context) ([]AdminRoleItem, error)
	CreateRole(context.Context, AdminRbacRoleCreateRequest) (AdminRoleItem, error)
	UpdateRole(context.Context, AdminRbacRoleUpdateRequest) error
	DeleteRole(context.Context, string) error
	SetRolePermissions(context.Context, string, []string) error
	ListAdmins(context.Context, PaginationParams) (adminListPage, error)
	GrantAdmin(context.Context, AdminGrantUpsert) error
	RevokeAdmin(context.Context, string) error
}
