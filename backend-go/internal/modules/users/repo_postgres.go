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

func (r *PostgresRepository) PutProfile(ctx context.Context, profile MyProfile) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.PutProfile(ctx, profile)
		return
	}
	_ = queryable.Exec(ctx, queryUpsertUserProfile, profile.UserID, profile.Nickname, profile.AvatarURL, profile.UserType, profile.Bio, profile.Gender, profile.Email, profile.Phone, profile.Location.LocationName, profile.Location.Point.Latitude, profile.Location.Point.Longitude, profile.PresenceStatus, profile.LastOnline, profile.CreatedAt)
}
func (r *PostgresRepository) DeleteProfile(ctx context.Context, userID string) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.DeleteProfile(ctx, userID)
		return
	}
	_ = queryable.Exec(ctx, queryDeleteUserProfile, userID)
}
func (r *PostgresRepository) PutSettings(ctx context.Context, userID string, settings UserSettings) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.PutSettings(ctx, userID, settings)
		return
	}
	_ = queryable.Exec(ctx, queryUpsertUserSettings, userID, settings.PrivacyLevel, settings.LocationVisible, settings.PresenceVisible, settings.SocialListsVisible, settings.GroupsVisible, settings.LikedMomentsVisible, settings.StatsVisible, settings.AllowFollow, settings.AllowPrivateChat, settings.Searchable, settings.PhoneSearchable, settings.NicknameSearchable, settings.LocationSearchable)
}
func (r *PostgresRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) []MyProfile {
	return r.fallback.SearchProfiles(ctx, req)
}
func (r *PostgresRepository) Follow(ctx context.Context, userID, targetUserID string) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.Follow(ctx, userID, targetUserID)
		return
	}
	_, blocked, remark := r.getRelationFields(ctx, userID, targetUserID)
	_ = queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, true, blocked, remark)
}
func (r *PostgresRepository) Unfollow(ctx context.Context, userID, targetUserID string) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.Unfollow(ctx, userID, targetUserID)
		return
	}
	_, blocked, remark := r.getRelationFields(ctx, userID, targetUserID)
	_ = queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, false, blocked, remark)
}
func (r *PostgresRepository) Block(ctx context.Context, userID, targetUserID string) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.Block(ctx, userID, targetUserID)
		return
	}
	following, _, remark := r.getRelationFields(ctx, userID, targetUserID)
	_ = queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, true, remark)
}
func (r *PostgresRepository) Unblock(ctx context.Context, userID, targetUserID string) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.Unblock(ctx, userID, targetUserID)
		return
	}
	following, _, remark := r.getRelationFields(ctx, userID, targetUserID)
	_ = queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, false, remark)
}
func (r *PostgresRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		r.fallback.SetRemark(ctx, userID, targetUserID, remark)
		return
	}
	following, blocked, _ := r.getRelationFields(ctx, userID, targetUserID)
	_ = queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, blocked, remark)
}
func (r *PostgresRepository) ListBlocked(ctx context.Context, userID string) []BlockedUserItem {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.ListBlocked(ctx, userID)
	}
	rows, err := queryable.Query(ctx, queryListBlockedUsers, userID)
	if err != nil {
		return r.fallback.ListBlocked(ctx, userID)
	}
	defer rows.Close()
	items := []BlockedUserItem{}
	for rows.Next() {
		var targetUserID, remark string
		if err := rows.Scan(&targetUserID, &remark); err != nil {
			return r.fallback.ListBlocked(ctx, userID)
		}
		items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: targetUserID}, Remark: remark})
	}
	return items
}
func (r *PostgresRepository) Relationship(ctx context.Context, userID, targetUserID string) Relationship {
	following, blocked, _ := r.getRelationFields(ctx, userID, targetUserID)
	if blocked {
		return RelationshipBlocked
	}
	if following {
		return RelationshipFollowing
	}
	return RelationshipNone
}

func (r *PostgresRepository) getRelationFields(ctx context.Context, userID, targetUserID string) (bool, bool, string) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		rel := r.fallback.Relationship(ctx, userID, targetUserID)
		return rel == RelationshipFollowing || rel == RelationshipFriend, rel == RelationshipBlocked, ""
	}
	var following, blocked bool
	var remark string
	if err := queryable.QueryRow(ctx, queryGetUserRelation, userID, targetUserID).Scan(&following, &blocked, &remark); err != nil {
		return false, false, ""
	}
	return following, blocked, remark
}
