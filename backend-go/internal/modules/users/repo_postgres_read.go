package users

import (
	"context"
	"errors"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.GetProfile(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MyProfile{}, false, persistence.ErrNotImplemented
	}
	var profile MyProfile
	var avatarURL, bio, gender, email, phone string
	var locationName string
	var latitude, longitude float64
	var presenceStatus, userType, lastOnline, createdAt string
	if err := queryable.QueryRow(ctx, queryGetUserProfile, userID).Scan(&profile.UserID, &profile.Nickname, &avatarURL, &userType, &bio, &gender, &email, &phone, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyProfile{}, false, nil
		}
		return MyProfile{}, false, err
	}
	profile.AvatarURL = avatarURL
	profile.UserType = UserType(userType)
	profile.Bio = bio
	profile.Gender = Gender(gender)
	profile.Email = email
	profile.Phone = phone
	profile.Location = Location{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	profile.PresenceStatus = UserPresenceStatus(presenceStatus)
	profile.LastOnline = lastOnline
	profile.CreatedAt = createdAt
	return profile, true, nil
}

func (r *PostgresRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.GetSettings(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return UserSettings{}, false, persistence.ErrNotImplemented
	}
	var settings UserSettings
	var allowPrivateChat string
	if err := queryable.QueryRow(ctx, queryGetUserSettings, userID).Scan(&settings.PrivacyLevel, &settings.LocationVisible, &settings.PresenceVisible, &settings.SocialListsVisible, &settings.GroupsVisible, &settings.LikedMomentsVisible, &settings.StatsVisible, &settings.AllowFollow, &allowPrivateChat, &settings.Searchable, &settings.PhoneSearchable, &settings.NicknameSearchable, &settings.LocationSearchable); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserSettings{}, false, nil
		}
		return UserSettings{}, false, err
	}
	settings.AllowPrivateChat = PrivateChatPermission(allowPrivateChat)
	return settings, true, nil
}

func (r *PostgresRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) ([]MyProfile, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.SearchProfiles(ctx, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, querySearchUserProfiles, req.Keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MyProfile{}
	for rows.Next() {
		var profile MyProfile
		var avatarURL, bio, gender, email, phone string
		var locationName string
		var latitude, longitude float64
		var presenceStatus, userType, lastOnline, createdAt string
		if err := rows.Scan(&profile.UserID, &profile.Nickname, &avatarURL, &userType, &bio, &gender, &email, &phone, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &createdAt); err != nil {
			return nil, err
		}
		profile.AvatarURL = avatarURL
		profile.UserType = UserType(userType)
		profile.Bio = bio
		profile.Gender = Gender(gender)
		profile.Email = email
		profile.Phone = phone
		profile.Location = Location{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
		profile.PresenceStatus = UserPresenceStatus(presenceStatus)
		profile.LastOnline = lastOnline
		profile.CreatedAt = createdAt
		items = append(items, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) ListBlocked(ctx context.Context, userID string) ([]BlockedUserItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListBlocked(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListBlockedUsers, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BlockedUserItem{}
	for rows.Next() {
		var targetUserID, remark string
		if err := rows.Scan(&targetUserID, &remark); err != nil {
			return nil, err
		}
		items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: targetUserID}, Remark: remark})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) Relationship(ctx context.Context, userID, targetUserID string) (Relationship, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Relationship(ctx, userID, targetUserID)
	}
	following, blocked, _, err := r.getRelationFields(ctx, userID, targetUserID)
	if err != nil {
		return RelationshipNone, err
	}
	if blocked {
		return RelationshipBlocked, nil
	}
	if following {
		return RelationshipFollowing, nil
	}
	return RelationshipNone, nil
}

func (r *PostgresRepository) getRelationFields(ctx context.Context, userID, targetUserID string) (bool, bool, string, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, false, "", persistence.ErrNotImplemented
	}
	var following, blocked bool
	var remark string
	if err := queryable.QueryRow(ctx, queryGetUserRelation, userID, targetUserID).Scan(&following, &blocked, &remark); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, "", nil
		}
		return false, false, "", err
	}
	return following, blocked, remark, nil
}
