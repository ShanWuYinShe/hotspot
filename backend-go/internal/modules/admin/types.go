package admin

type AdminRoleBasic struct {
	RoleID string `json:"role_id"`
	Name   string `json:"name"`
}
type AdminPermissions struct {
	Permissions []string         `json:"permissions"`
	Roles       []AdminRoleBasic `json:"roles"`
}
