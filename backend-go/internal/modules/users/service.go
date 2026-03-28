package users

import (
	"context"
	"time"

	momenttypes "hotspot/backend-go/internal/modules/moments"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) GetMyProfile(ctx context.Context) (MyProfile, error) {
	startedAt := time.Now()
	observability.L().Info("users.get_my_profile started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if profile, ok, err := s.repo.GetProfile(ctx, principal.UserID); err != nil {
		observability.L().Error("users.get_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return MyProfile{}, err
	} else if ok {
		observability.L().Info("users.get_my_profile completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return profile, nil
	}
	profile := defaultProfile(principal.UserID)
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.get_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return MyProfile{}, err
	}
	observability.L().Info("users.get_my_profile completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return profile, nil
}

func (s *Service) UpdateMyProfile(ctx context.Context, req UpdateProfileRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.update_my_profile started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.update_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
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
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.update_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.update_my_profile completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) GetMySettings(ctx context.Context) (UserSettings, error) {
	startedAt := time.Now()
	observability.L().Info("users.get_my_settings started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if settings, ok, err := s.repo.GetSettings(ctx, principal.UserID); err != nil {
		observability.L().Error("users.get_my_settings failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserSettings{}, err
	} else if ok {
		observability.L().Info("users.get_my_settings completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return settings, nil
	}
	settings := defaultSettings()
	if err := s.repo.PutSettings(ctx, principal.UserID, settings); err != nil {
		observability.L().Error("users.get_my_settings failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserSettings{}, err
	}
	observability.L().Info("users.get_my_settings completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return settings, nil
}

func (s *Service) UpdateMySettings(ctx context.Context, req UpdateSettingsRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.update_my_settings started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.PutSettings(ctx, principal.UserID, req.Settings); err != nil {
		observability.L().Error("users.update_my_settings failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.update_my_settings completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) Search(ctx context.Context, req SearchUsersRequest) (PaginatedUsers, error) {
	startedAt := time.Now()
	observability.L().Info("users.search started")
	profiles, err := s.repo.SearchProfiles(ctx, req)
	if err != nil {
		observability.L().Error("users.search failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return PaginatedUsers{}, err
	}
	items := make([]UserItem, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, userItemFromProfile(profile, RelationshipNone))
	}
	observability.L().Info("users.search completed", "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return PaginatedUsers{Items: items}, nil
}
func (s *Service) OpenTheDoor(ctx context.Context, userID string) (UserItem, bool, error) {
	startedAt := time.Now()
	observability.L().Info("users.open_the_door started", "user_id", userID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		observability.L().Error("users.open_the_door failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserItem{}, false, err
	}
	if !ok {
		observability.L().Info("users.open_the_door not found", "user_id", userID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserItem{}, false, nil
	}
	relation, err := s.repo.Relationship(ctx, principal.UserID, userID)
	if err != nil {
		observability.L().Error("users.open_the_door failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserItem{}, false, err
	}
	observability.L().Info("users.open_the_door completed", "user_id", userID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return userItemFromProfile(profile, relation), true, nil
}
func (s *Service) UpdateLocation(ctx context.Context, req UpdateLocationRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.update_location started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.update_location failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		profile = defaultProfile(principal.UserID)
	}
	profile.Location = req.Location
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.update_location failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.update_location completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) Upgrade(ctx context.Context, req UpgradeAccountRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.upgrade started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		profile = defaultProfile(principal.UserID)
	}
	profile.UserType = UserTypeNormal
	profile.Email = req.LoginID
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.upgrade completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) DeleteAccount(ctx context.Context) error {
	startedAt := time.Now()
	observability.L().Info("users.delete_account started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.DeleteProfile(ctx, principal.UserID); err != nil {
		observability.L().Error("users.delete_account failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.delete_account completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) Follow(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.follow started", "target_user_id", targetUserID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.Follow(ctx, principal.UserID, targetUserID); err != nil {
		observability.L().Error("users.follow failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: "NewFollower", Payload: map[string]any{"actor": map[string]any{"user_id": principal.UserID, "nickname": principal.UserID}, "followed_at": time.Now().UTC().Format(time.RFC3339)}})
	observability.L().Info("users.follow completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) Unfollow(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.unfollow started", "target_user_id", targetUserID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.Unfollow(ctx, principal.UserID, targetUserID); err != nil {
		observability.L().Error("users.unfollow failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.unfollow completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) Block(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.block started", "target_user_id", targetUserID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.Block(ctx, principal.UserID, targetUserID); err != nil {
		observability.L().Error("users.block failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.block completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) Unblock(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.unblock started", "target_user_id", targetUserID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.Unblock(ctx, principal.UserID, targetUserID); err != nil {
		observability.L().Error("users.unblock failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.unblock completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) SetRemark(ctx context.Context, targetUserID, remark string) error {
	startedAt := time.Now()
	observability.L().Info("users.set_remark started", "target_user_id", targetUserID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.SetRemark(ctx, principal.UserID, targetUserID, remark); err != nil {
		observability.L().Error("users.set_remark failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.set_remark completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}
func (s *Service) ListBlocked(ctx context.Context) (PaginatedBlockedUsers, error) {
	startedAt := time.Now()
	observability.L().Info("users.list_blocked started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items, err := s.repo.ListBlocked(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.list_blocked failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return PaginatedBlockedUsers{}, err
	}
	observability.L().Info("users.list_blocked completed", "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return PaginatedBlockedUsers{Items: items}, nil
}

func (s *Service) ListLikedMoments(ctx context.Context, req UserLikedMomentsRequest) (momenttypes.PaginatedMoments, error) {
	startedAt := time.Now()
	observability.L().Info("users.list_liked_moments started", "user_id", req.UserID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	settings, ok, err := s.repo.GetSettings(ctx, req.UserID)
	if err != nil {
		observability.L().Error("users.list_liked_moments failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return momenttypes.PaginatedMoments{}, err
	}
	if !ok {
		settings = defaultSettings()
	}
	if !settings.LikedMomentsVisible && principal.UserID != req.UserID {
		observability.L().Info("users.list_liked_moments hidden by privacy", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return momenttypes.PaginatedMoments{Items: []momenttypes.MomentItem{}}, nil
	}
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > 100 {
			limit = 100
		}
		cursor = req.Pagination.Cursor
	}
	result, err := s.repo.ListLikedMoments(ctx, req.UserID, limit, cursor)
	if err != nil {
		observability.L().Error("users.list_liked_moments failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return momenttypes.PaginatedMoments{}, err
	}
	observability.L().Info("users.list_liked_moments completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return result, nil
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
