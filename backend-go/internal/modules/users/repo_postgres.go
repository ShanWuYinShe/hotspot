package users

import (
	"context"

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
	return queryable.Exec(ctx, queryUpsertUserProfile, profile.UserID, profile.Nickname, profile.AvatarURL, profile.UserType, profile.Bio, profile.Gender, profile.Email, profile.Phone, profile.Location.LocationName, profile.Location.Point.Latitude, profile.Location.Point.Longitude, profile.PresenceStatus, profile.LastOnline, profile.CreatedAt)
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
	_, blocked, remark, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, true, blocked, remark)
}
func (r *PostgresRepository) Unfollow(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Unfollow(ctx, userID, targetUserID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	_, blocked, remark, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, false, blocked, remark)
}
func (r *PostgresRepository) Block(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Block(ctx, userID, targetUserID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, _, remark, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, true, remark)
}
func (r *PostgresRepository) Unblock(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Unblock(ctx, userID, targetUserID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, _, remark, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, false, remark)
}
func (r *PostgresRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.SetRemark(ctx, userID, targetUserID, remark)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, blocked, _, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, blocked, remark)
}
