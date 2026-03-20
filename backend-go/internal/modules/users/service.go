package users

import (
	"context"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) GetMyProfile(ctx context.Context) MyProfile {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if profile, ok := s.repo.GetProfile(ctx, principal.UserID); ok {
		return profile
	}
	profile := defaultProfile(principal.UserID)
	s.repo.PutProfile(ctx, profile)
	return profile
}

func (s *Service) UpdateMyProfile(ctx context.Context, req UpdateProfileRequest) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok := s.repo.GetProfile(ctx, principal.UserID)
	if !ok {
		profile = defaultProfile(principal.UserID)
	}
	if req.Nickname != "" {
		profile.Nickname = req.Nickname
	}
	if req.AvatarFileID != "" {
		profile.AvatarURL = req.AvatarFileID
	}
	if req.Bio != "" {
		profile.Bio = req.Bio
	}
	if req.Tags != nil {
		profile.Tags = req.Tags
	}
	if req.Gender != "" {
		profile.Gender = req.Gender
	}
	s.repo.PutProfile(ctx, profile)
}

func (s *Service) GetMySettings(ctx context.Context) UserSettings {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if settings, ok := s.repo.GetSettings(ctx, principal.UserID); ok {
		return settings
	}
	settings := defaultSettings()
	s.repo.PutSettings(ctx, principal.UserID, settings)
	return settings
}

func (s *Service) UpdateMySettings(ctx context.Context, req UpdateSettingsRequest) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.PutSettings(ctx, principal.UserID, req.Settings)
}
func (s *Service) Search(ctx context.Context, req SearchUsersRequest) PaginatedUsers {
	profiles := s.repo.SearchProfiles(ctx, req)
	items := make([]UserItem, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, userItemFromProfile(profile, RelationshipNone))
	}
	return PaginatedUsers{Items: items}
}
func (s *Service) OpenTheDoor(ctx context.Context, userID string) (UserItem, bool) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok := s.repo.GetProfile(ctx, userID)
	if !ok {
		profile = defaultProfile(userID)
		s.repo.PutProfile(ctx, profile)
	}
	return userItemFromProfile(profile, s.repo.Relationship(ctx, principal.UserID, userID)), true
}
func (s *Service) UpdateLocation(ctx context.Context, req UpdateLocationRequest) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok := s.repo.GetProfile(ctx, principal.UserID)
	if !ok {
		profile = defaultProfile(principal.UserID)
	}
	profile.Location = req.Location
	s.repo.PutProfile(ctx, profile)
}
func (s *Service) Upgrade(ctx context.Context, req UpgradeAccountRequest) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok := s.repo.GetProfile(ctx, principal.UserID)
	if !ok {
		profile = defaultProfile(principal.UserID)
	}
	profile.UserType = UserTypeNormal
	profile.Email = req.LoginID
	s.repo.PutProfile(ctx, profile)
}
func (s *Service) DeleteAccount(ctx context.Context) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.DeleteProfile(ctx, principal.UserID)
}
func (s *Service) Follow(ctx context.Context, targetUserID string) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.Follow(ctx, principal.UserID, targetUserID)
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: "NewFollower", Payload: map[string]any{"actor": map[string]any{"user_id": principal.UserID, "nickname": principal.UserID}, "followed_at": time.Now().UTC().Format(time.RFC3339)}})
}
func (s *Service) Unfollow(ctx context.Context, targetUserID string) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.Unfollow(ctx, principal.UserID, targetUserID)
}
func (s *Service) Block(ctx context.Context, targetUserID string) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.Block(ctx, principal.UserID, targetUserID)
}
func (s *Service) Unblock(ctx context.Context, targetUserID string) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.Unblock(ctx, principal.UserID, targetUserID)
}
func (s *Service) SetRemark(ctx context.Context, targetUserID, remark string) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.SetRemark(ctx, principal.UserID, targetUserID, remark)
}
func (s *Service) ListBlocked(ctx context.Context) PaginatedBlockedUsers {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return PaginatedBlockedUsers{Items: s.repo.ListBlocked(ctx, principal.UserID)}
}

func defaultProfile(userID string) MyProfile {
	now := time.Now().UTC()
	return MyProfile{UserID: userID, Nickname: "HotSpot User", UserType: UserTypeNormal, Location: Location{Point: GeoPoint{Latitude: 35.154922, Longitude: 112.682524}, LocationName: "南天门"}, PresenceStatus: UserPresenceOnline, LastOnline: now.Format(time.RFC3339), CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339), Stats: UserStats{}}
}

func defaultSettings() UserSettings {
	return UserSettings{PrivacyLevel: "public", LocationVisible: true, PresenceVisible: true, SocialListsVisible: true, GroupsVisible: true, LikedMomentsVisible: true, StatsVisible: true, AllowFollow: true, AllowPrivateChat: PrivateChatEveryone, Searchable: true, PhoneSearchable: false, NicknameSearchable: true, LocationSearchable: true}
}
func userItemFromProfile(profile MyProfile, relation Relationship) UserItem {
	return UserItem{User: UserInfo{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: profile.AvatarURL}, Bio: profile.Bio, Tags: profile.Tags, PresenceStatus: profile.PresenceStatus, LastOnline: profile.LastOnline, Location: &RelativeLocation{Point: profile.Location.Point, LocationName: profile.Location.LocationName}, Stats: &profile.Stats, Relation: relation}
}
