package users

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct {
	factory *persistence.Factory
	memory  *InMemoryRepository
}

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	if factory == nil {
		factory = &persistence.Factory{}
	}
	return &PersistenceRepository{factory: factory, memory: NewInMemoryRepository()}
}

func (r *PersistenceRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool) {
	return r.memory.GetProfile(ctx, userID)
}
func (r *PersistenceRepository) PutProfile(ctx context.Context, profile MyProfile) {
	r.memory.PutProfile(ctx, profile)
}
func (r *PersistenceRepository) DeleteProfile(ctx context.Context, userID string) {
	r.memory.DeleteProfile(ctx, userID)
}
func (r *PersistenceRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool) {
	return r.memory.GetSettings(ctx, userID)
}
func (r *PersistenceRepository) PutSettings(ctx context.Context, userID string, settings UserSettings) {
	r.memory.PutSettings(ctx, userID, settings)
}
func (r *PersistenceRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) []MyProfile {
	return r.memory.SearchProfiles(ctx, req)
}
func (r *PersistenceRepository) Follow(ctx context.Context, userID, targetUserID string) {
	r.memory.Follow(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Unfollow(ctx context.Context, userID, targetUserID string) {
	r.memory.Unfollow(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Block(ctx context.Context, userID, targetUserID string) {
	r.memory.Block(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Unblock(ctx context.Context, userID, targetUserID string) {
	r.memory.Unblock(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) {
	r.memory.SetRemark(ctx, userID, targetUserID, remark)
}
func (r *PersistenceRepository) ListBlocked(ctx context.Context, userID string) []BlockedUserItem {
	return r.memory.ListBlocked(ctx, userID)
}
func (r *PersistenceRepository) Relationship(ctx context.Context, userID, targetUserID string) Relationship {
	return r.memory.Relationship(ctx, userID, targetUserID)
}
