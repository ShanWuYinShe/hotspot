package admin

import (
	"context"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Service struct{ appCfg appconfig.Source }

func NewService(appCfg appconfig.Source) *Service { return &Service{appCfg: appCfg} }
func (s *Service) GetMe(ctx context.Context) AdminPermissions {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	_ = principal
	return AdminPermissions{Permissions: []string{"*"}, Roles: []AdminRoleBasic{{RoleID: "role_super", Name: "Super Admin"}}}
}
func (s *Service) GetAppConfig(ctx context.Context) any { return s.appCfg.Get(ctx) }
