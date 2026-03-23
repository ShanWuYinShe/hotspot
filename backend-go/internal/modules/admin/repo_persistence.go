package admin

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	memory := NewInMemoryRepository()
	if factory == nil || factory.Config.PostgresDSN == "" {
		return &PersistenceRepository{impl: memory}
	}
	return &PersistenceRepository{impl: NewPostgresRepository(factory, memory)}
}

func (r *PersistenceRepository) GetActor(ctx context.Context, userID string) (AdminActor, bool, error) {
	return r.impl.GetActor(ctx, userID)
}

func (r *PersistenceRepository) ListPermissions(ctx context.Context) ([]AdminPermissionItem, error) {
	return r.impl.ListPermissions(ctx)
}

func (r *PersistenceRepository) ListRoles(ctx context.Context) ([]AdminRoleItem, error) {
	return r.impl.ListRoles(ctx)
}

func (r *PersistenceRepository) CreateRole(ctx context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, error) {
	return r.impl.CreateRole(ctx, req)
}

func (r *PersistenceRepository) UpdateRole(ctx context.Context, req AdminRbacRoleUpdateRequest) error {
	return r.impl.UpdateRole(ctx, req)
}

func (r *PersistenceRepository) DeleteRole(ctx context.Context, roleID string) error {
	return r.impl.DeleteRole(ctx, roleID)
}

func (r *PersistenceRepository) SetRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error {
	return r.impl.SetRolePermissions(ctx, roleID, permissionKeys)
}

func (r *PersistenceRepository) ListAdmins(ctx context.Context, pagination PaginationParams) (adminListPage, error) {
	return r.impl.ListAdmins(ctx, pagination)
}

func (r *PersistenceRepository) GrantAdmin(ctx context.Context, grant AdminGrantUpsert) error {
	return r.impl.GrantAdmin(ctx, grant)
}

func (r *PersistenceRepository) RevokeAdmin(ctx context.Context, userID string) error {
	return r.impl.RevokeAdmin(ctx, userID)
}
