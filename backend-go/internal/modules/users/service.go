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

func (s *Service) GetMyProfile(ctx context.Context) (MyProfile, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if profile, ok, err := s.repo.GetProfile(ctx, principal.UserID); err != nil {
		return MyProfile{}, err
	} else if ok {
		return profile, nil
	}
	profile := defaultProfile(principal.UserID)
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		return MyProfile{}, err
	}
	return profile, nil
}

func (s *Service) UpdateMyProfile(ctx context.Context, req UpdateProfileRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		return err
	}
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
	return s.repo.PutProfile(ctx, profile)
}

func (s *Service) GetMySettings(ctx context.Context) (UserSettings, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if settings, ok, err := s.repo.GetSettings(ctx, principal.UserID); err != nil {
		return UserSettings{}, err
	} else if ok {
		return settings, nil
	}
	settings := defaultSettings()
	if err := s.repo.PutSettings(ctx, principal.UserID, settings); err != nil {
		return UserSettings{}, err
	}
	return settings, nil
}

func (s *Service) UpdateMySettings(ctx context.Context, req UpdateSettingsRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.PutSettings(ctx, principal.UserID, req.Settings)
}
func (s *Service) Search(ctx context.Context, req SearchUsersRequest) (PaginatedUsers, error) {
	profiles, err := s.repo.SearchProfiles(ctx, req)
	if err != nil {
		return PaginatedUsers{}, err
	}
	items := make([]UserItem, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, userItemFromProfile(profile, RelationshipNone))
	}
	return PaginatedUsers{Items: items}, nil
}
func (s *Service) OpenTheDoor(ctx context.Context, userID string) (UserItem, bool, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return UserItem{}, false, err
	}
	if !ok {
		return UserItem{}, false, nil
	}
	relation, err := s.repo.Relationship(ctx, principal.UserID, userID)
	if err != nil {
		return UserItem{}, false, err
	}
	return userItemFromProfile(profile, relation), true, nil
}
func (s *Service) UpdateLocation(ctx context.Context, req UpdateLocationRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		return err
	}
	if !ok {
		profile = defaultProfile(principal.UserID)
	}
	profile.Location = req.Location
	return s.repo.PutProfile(ctx, profile)
}
func (s *Service) Upgrade(ctx context.Context, req UpgradeAccountRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		return err
	}
	if !ok {
		profile = defaultProfile(principal.UserID)
	}
	profile.UserType = UserTypeNormal
	profile.Email = req.LoginID
	return s.repo.PutProfile(ctx, profile)
}
func (s *Service) DeleteAccount(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.DeleteProfile(ctx, principal.UserID)
}
func (s *Service) Follow(ctx context.Context, targetUserID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.Follow(ctx, principal.UserID, targetUserID); err != nil {
		return err
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: "NewFollower", Payload: map[string]any{"actor": map[string]any{"user_id": principal.UserID, "nickname": principal.UserID}, "followed_at": time.Now().UTC().Format(time.RFC3339)}})
	return nil
}
func (s *Service) Unfollow(ctx context.Context, targetUserID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unfollow(ctx, principal.UserID, targetUserID)
}
func (s *Service) Block(ctx context.Context, targetUserID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Block(ctx, principal.UserID, targetUserID)
}
func (s *Service) Unblock(ctx context.Context, targetUserID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unblock(ctx, principal.UserID, targetUserID)
}
func (s *Service) SetRemark(ctx context.Context, targetUserID, remark string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.SetRemark(ctx, principal.UserID, targetUserID, remark)
}
func (s *Service) ListBlocked(ctx context.Context) (PaginatedBlockedUsers, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items, err := s.repo.ListBlocked(ctx, principal.UserID)
	if err != nil {
		return PaginatedBlockedUsers{}, err
	}
	return PaginatedBlockedUsers{Items: items}, nil
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
