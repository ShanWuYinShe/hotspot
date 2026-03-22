package admin

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Store interface {
	GetMe(context.Context) AdminPermissions
}

type InMemoryStore struct{}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{} }

func (s *InMemoryStore) GetMe(ctx context.Context) AdminPermissions {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	_ = principal
	return AdminPermissions{
		Permissions: []string{"*"},
		Roles:       []AdminRoleBasic{{RoleID: "role_super", Name: "Super Admin"}},
	}
}
