package users

import (
	"context"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) PutProfile(ctx context.Context, profile MyProfile) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.PutProfile(ctx, profile)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryUpsertUserProfile, profile.UserID, profile.Nickname, profile.AvatarURL, profile.UserType, profile.Bio, profile.Gender, profile.Email, profile.Phone, profile.Location.LocationName, profile.Location.Point.Latitude, profile.Location.Point.Longitude, profile.PresenceStatus, profile.LastOnline, profile.CreatedAt); err != nil {
		return err
	}
	if err := queryable.Exec(ctx, queryReplaceUserProfileTags, profile.UserID); err != nil {
		return err
	}
	for _, tag := range profile.Tags {
		if err := queryable.Exec(ctx, queryInsertUserProfileTag, profile.UserID, tag); err != nil {
			return err
		}
	}
	return nil
}
func (r *PostgresRepository) DeleteProfile(ctx context.Context, userID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.DeleteProfile(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteUserProfile, userID)
}
func (r *PostgresRepository) PutSettings(ctx context.Context, userID string, settings UserSettings) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.PutSettings(ctx, userID, settings)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpsertUserSettings, userID, settings.PrivacyLevel, settings.LocationVisible, settings.PresenceVisible, settings.SocialListsVisible, settings.GroupsVisible, settings.LikedMomentsVisible, settings.StatsVisible, settings.AllowFollow, settings.AllowPrivateChat, settings.Searchable, settings.PhoneSearchable, settings.NicknameSearchable, settings.LocationSearchable)
}
func (r *PostgresRepository) Follow(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Follow(ctx, userID, targetUserID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	_, blocked, remark, _, blockedAt, blockReason, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, true, blocked, remark, nowRFC3339(), blockedAt, blockReason)
}
func (r *PostgresRepository) Unfollow(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Unfollow(ctx, userID, targetUserID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	_, blocked, remark, _, blockedAt, blockReason, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, false, blocked, remark, nil, blockedAt, blockReason)
}
func (r *PostgresRepository) Block(ctx context.Context, userID, targetUserID, reason string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Block(ctx, userID, targetUserID, reason)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, _, remark, followedAt, _, _, _ := r.getRelationFields(ctx, userID, targetUserID)
	blockedAt := nowRFC3339()
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, true, remark, followedAt, blockedAt, reason)
}
func (r *PostgresRepository) Unblock(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Unblock(ctx, userID, targetUserID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, _, remark, followedAt, _, _, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, false, remark, followedAt, nil, "")
}
func (r *PostgresRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.SetRemark(ctx, userID, targetUserID, remark)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, blocked, _, followedAt, blockedAt, blockReason, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, blocked, remark, followedAt, blockedAt, blockReason)
}

func (r *PostgresRepository) ListBlocked(ctx context.Context, userID string, limit int, cursor string) (PaginatedBlockedUsers, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListBlocked(ctx, userID, limit, cursor)
	}
	return listBlockedUsersPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListLikedMoments(ctx context.Context, userID string, limit int, cursor, viewerUserID string) (momenttypes.PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListLikedMoments(ctx, userID, limit, cursor, viewerUserID)
	}
	return listLikedMomentsPostgres(ctx, r.factory, userID, limit, cursor, viewerUserID)
}

func (r *PostgresRepository) ListFriends(ctx context.Context, userID string, limit int, cursor string) (PaginatedFriends, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListFriends(ctx, userID, limit, cursor)
	}
	return listFriendsPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListFollowing(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowing, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListFollowing(ctx, userID, limit, cursor)
	}
	return listFollowingPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListFollowers(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowers, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListFollowers(ctx, userID, limit, cursor)
	}
	return listFollowersPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListGroups(ctx context.Context, userID string, limit int, cursor string) (grouptypes.PaginatedGroups, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListGroups(ctx, userID, limit, cursor)
	}
	return listUserGroupsPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListMoments(ctx context.Context, userID string, limit int, cursor string, includePrivate bool) (momenttypes.PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMoments(ctx, userID, limit, cursor, includePrivate)
	}
	return listUserMomentsPostgres(ctx, r.factory, userID, limit, cursor, includePrivate)
}

func (r *PostgresRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.UserExists(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var one int
	if err := queryable.QueryRow(ctx, queryUserExists, userID).Scan(&one); err != nil {
		return false, nil
	}
	return true, nil
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
