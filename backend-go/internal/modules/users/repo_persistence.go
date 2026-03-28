package users

import (
	"context"

	momenttypes "hotspot/backend-go/internal/modules/moments"
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

func (r *PersistenceRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool, error) {
	return r.impl.GetProfile(ctx, userID)
}
func (r *PersistenceRepository) PutProfile(ctx context.Context, profile MyProfile) error {
	return r.impl.PutProfile(ctx, profile)
}
func (r *PersistenceRepository) DeleteProfile(ctx context.Context, userID string) error {
	return r.impl.DeleteProfile(ctx, userID)
}
func (r *PersistenceRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool, error) {
	return r.impl.GetSettings(ctx, userID)
}
func (r *PersistenceRepository) PutSettings(ctx context.Context, userID string, settings UserSettings) error {
	return r.impl.PutSettings(ctx, userID, settings)
}
func (r *PersistenceRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) ([]MyProfile, error) {
	return r.impl.SearchProfiles(ctx, req)
}
func (r *PersistenceRepository) Follow(ctx context.Context, userID, targetUserID string) error {
	return r.impl.Follow(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Unfollow(ctx context.Context, userID, targetUserID string) error {
	return r.impl.Unfollow(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Block(ctx context.Context, userID, targetUserID string) error {
	return r.impl.Block(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Unblock(ctx context.Context, userID, targetUserID string) error {
	return r.impl.Unblock(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) error {
	return r.impl.SetRemark(ctx, userID, targetUserID, remark)
}
func (r *PersistenceRepository) ListBlocked(ctx context.Context, userID string) ([]BlockedUserItem, error) {
	return r.impl.ListBlocked(ctx, userID)
}

func (r *PersistenceRepository) ListLikedMoments(ctx context.Context, userID string, limit int, cursor string) (momenttypes.PaginatedMoments, error) {
	return r.impl.ListLikedMoments(ctx, userID, limit, cursor)
}

func (r *PersistenceRepository) Relationship(ctx context.Context, userID, targetUserID string) (Relationship, error) {
	return r.impl.Relationship(ctx, userID, targetUserID)
}
