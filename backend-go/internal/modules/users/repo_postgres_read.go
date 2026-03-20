package users

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.GetProfile(ctx, userID)
	}
	var profile MyProfile
	var avatarURL, bio, gender, email, phone string
	var locationName string
	var latitude, longitude float64
	var presenceStatus, userType, lastOnline, createdAt string
	if err := queryable.QueryRow(ctx, queryGetUserProfile, userID).Scan(&profile.UserID, &profile.Nickname, &avatarURL, &userType, &bio, &gender, &email, &phone, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &createdAt); err != nil {
		return r.fallback.GetProfile(ctx, userID)
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
	return profile, true
}

func (r *PostgresRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return r.fallback.GetSettings(ctx, userID)
	}
	var settings UserSettings
	var allowPrivateChat string
	if err := queryable.QueryRow(ctx, queryGetUserSettings, userID).Scan(&settings.PrivacyLevel, &settings.LocationVisible, &settings.PresenceVisible, &settings.SocialListsVisible, &settings.GroupsVisible, &settings.LikedMomentsVisible, &settings.StatsVisible, &settings.AllowFollow, &allowPrivateChat, &settings.Searchable, &settings.PhoneSearchable, &settings.NicknameSearchable, &settings.LocationSearchable); err != nil {
		return r.fallback.GetSettings(ctx, userID)
	}
	settings.AllowPrivateChat = PrivateChatPermission(allowPrivateChat)
	return settings, true
}
