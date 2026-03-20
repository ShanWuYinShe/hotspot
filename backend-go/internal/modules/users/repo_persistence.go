package users

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

func (r *PersistenceRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool) {
	return r.impl.GetProfile(ctx, userID)
}
func (r *PersistenceRepository) PutProfile(ctx context.Context, profile MyProfile) {
	r.impl.PutProfile(ctx, profile)
}
func (r *PersistenceRepository) DeleteProfile(ctx context.Context, userID string) {
	r.impl.DeleteProfile(ctx, userID)
}
func (r *PersistenceRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool) {
	return r.impl.GetSettings(ctx, userID)
}
func (r *PersistenceRepository) PutSettings(ctx context.Context, userID string, settings UserSettings) {
	r.impl.PutSettings(ctx, userID, settings)
}
func (r *PersistenceRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) []MyProfile {
	return r.impl.SearchProfiles(ctx, req)
}
func (r *PersistenceRepository) Follow(ctx context.Context, userID, targetUserID string) {
	r.impl.Follow(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Unfollow(ctx context.Context, userID, targetUserID string) {
	r.impl.Unfollow(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Block(ctx context.Context, userID, targetUserID string) {
	r.impl.Block(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Unblock(ctx context.Context, userID, targetUserID string) {
	r.impl.Unblock(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) {
	r.impl.SetRemark(ctx, userID, targetUserID, remark)
}
func (r *PersistenceRepository) ListBlocked(ctx context.Context, userID string) []BlockedUserItem {
	return r.impl.ListBlocked(ctx, userID)
}
func (r *PersistenceRepository) Relationship(ctx context.Context, userID, targetUserID string) Relationship {
	return r.impl.Relationship(ctx, userID, targetUserID)
}
