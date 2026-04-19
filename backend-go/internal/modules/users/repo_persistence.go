package users

import (
	"context"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
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
func (r *PersistenceRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) (PaginatedProfiles, error) {
	return r.impl.SearchProfiles(ctx, req)
}
func (r *PersistenceRepository) Follow(ctx context.Context, userID, targetUserID string, source FollowSource, sourceMomentID string) error {
	return r.impl.Follow(ctx, userID, targetUserID, source, sourceMomentID)
}
func (r *PersistenceRepository) Unfollow(ctx context.Context, userID, targetUserID string) error {
	return r.impl.Unfollow(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) Block(ctx context.Context, userID, targetUserID, reason string) error {
	return r.impl.Block(ctx, userID, targetUserID, reason)
}
func (r *PersistenceRepository) Unblock(ctx context.Context, userID, targetUserID string) error {
	return r.impl.Unblock(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) error {
	return r.impl.SetRemark(ctx, userID, targetUserID, remark)
}
func (r *PersistenceRepository) ListBlocked(ctx context.Context, userID string, limit int, cursor string) (PaginatedBlockedUsers, error) {
	return r.impl.ListBlocked(ctx, userID, limit, cursor)
}
func (r *PersistenceRepository) ListLikedMoments(ctx context.Context, userID string, limit int, cursor, viewerUserID string) (momenttypes.PaginatedMoments, error) {
	return r.impl.ListLikedMoments(ctx, userID, limit, cursor, viewerUserID)
}
func (r *PersistenceRepository) Relationship(ctx context.Context, userID, targetUserID string) (Relationship, error) {
	return r.impl.Relationship(ctx, userID, targetUserID)
}
func (r *PersistenceRepository) ListFriends(ctx context.Context, userID string, limit int, cursor string) (PaginatedFriends, error) {
	return r.impl.ListFriends(ctx, userID, limit, cursor)
}
func (r *PersistenceRepository) ListFollowing(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowing, error) {
	return r.impl.ListFollowing(ctx, userID, limit, cursor)
}
func (r *PersistenceRepository) ListFollowers(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowers, error) {
	return r.impl.ListFollowers(ctx, userID, limit, cursor)
}
func (r *PersistenceRepository) ListGroups(ctx context.Context, userID string, limit int, cursor string) (grouptypes.PaginatedGroups, error) {
	return r.impl.ListGroups(ctx, userID, limit, cursor)
}
func (r *PersistenceRepository) ListMoments(ctx context.Context, userID string, limit int, cursor string, includePrivate bool) (momenttypes.PaginatedMoments, error) {
	return r.impl.ListMoments(ctx, userID, limit, cursor, includePrivate)
}
func (r *PersistenceRepository) CommonGroupCount(ctx context.Context, viewerUserID, targetUserID string) (int, error) {
	return r.impl.CommonGroupCount(ctx, viewerUserID, targetUserID)
}
func (r *PersistenceRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	return r.impl.UserExists(ctx, userID)
}
