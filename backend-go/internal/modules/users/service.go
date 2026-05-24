package users

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct {
	repo          Repository
	notifications userNotificationWriter
	appConfig     appconfig.Source
}

type userNotificationWriter interface {
	AppendNewFollower(ctx context.Context, targetUserID string, actor map[string]any, source string, sourceMomentID string) error
	AppendMutualFollow(ctx context.Context, targetUserID string, actor map[string]any) error
}

func NewService() *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), nil, nil)
}

func NewServiceWithRepo(repo Repository) *Service { return NewServiceWithDeps(repo, nil, nil) }

func NewServiceWithDeps(repo Repository, notificationWriter userNotificationWriter, appConfig appconfig.Source) *Service {
	return &Service{repo: repo, notifications: notificationWriter, appConfig: appConfig}
}

func (s *Service) GetMyProfile(ctx context.Context) (MyProfile, error) {
	startedAt := time.Now()
	observability.L().Info("users.get_my_profile started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return MyProfile{}, err
	}
	if profile, ok, err := s.repo.GetProfile(ctx, principal.UserID); err != nil {
		observability.L().Error("users.get_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return MyProfile{}, err
	} else if ok {
		observability.L().Info("users.get_my_profile completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return profile, nil
	}
	profile := s.defaultProfile(ctx, principal.UserID)
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
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.update_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		profile = s.defaultProfile(ctx, principal.UserID)
	}
	if req.Nickname != nil {
		profile.Nickname = *req.Nickname
	}
	if req.AvatarFileID != nil {
		profile.AvatarURL = *req.AvatarFileID
	}
	if req.Bio != nil {
		profile.Bio = *req.Bio
	}
	if req.Tags != nil {
		profile.Tags = *req.Tags
	}
	if req.Gender != nil {
		profile.Gender = *req.Gender
	}
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.update_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	event := realtime.Event{
		Type: realtime.EventTypeUserProfileUpdated,
		Payload: map[string]any{
			"user_id":    principal.UserID,
			"updated_at": now,
		},
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	realtime.DefaultHub().BroadcastToUserSubscribers(principal.UserID, event)
	observability.L().Info("users.update_my_profile completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) RequestEmailChange(ctx context.Context, req RequestEmailChangeRequest) (VerificationSent, error) {
	if err := s.contactChangeUnavailable(ctx, "users.request_email_change unavailable"); err != nil {
		return VerificationSent{}, err
	}
	return VerificationSent{}, nil
}

func (s *Service) ConfirmEmailChange(ctx context.Context, req ConfirmEmailChangeRequest) error {
	return s.contactChangeUnavailable(ctx, "users.confirm_email_change unavailable")
}

func (s *Service) RequestPhoneChange(ctx context.Context, req RequestPhoneChangeRequest) (VerificationSent, error) {
	if err := s.contactChangeUnavailable(ctx, "users.request_phone_change unavailable"); err != nil {
		return VerificationSent{}, err
	}
	return VerificationSent{}, nil
}

func (s *Service) ConfirmPhoneChange(ctx context.Context, req ConfirmPhoneChangeRequest) error {
	return s.contactChangeUnavailable(ctx, "users.confirm_phone_change unavailable")
}

func (s *Service) contactChangeUnavailable(ctx context.Context, message string) error {
	if _, err := requirePrincipal(ctx); err != nil {
		return err
	}
	observability.L().Warn(message)
	return ErrContactChangeUnavailable
}

func (s *Service) Search(ctx context.Context, req SearchUsersRequest) (PaginatedUsers, error) {
	startedAt := time.Now()
	observability.L().Info("users.search started")
	profiles, err := s.repo.SearchProfiles(ctx, req)
	if err != nil {
		observability.L().Error("users.search failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return PaginatedUsers{}, err
	}
	if profiles.InvalidCursor {
		return PaginatedUsers{}, ErrInvalidCursor
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	items := make([]UserItem, 0, len(profiles.Items))
	for _, profile := range profiles.Items {
		settings, ok, err := s.repo.GetSettings(ctx, profile.UserID)
		if err != nil {
			return PaginatedUsers{}, err
		}
		if !ok {
			settings = defaultSettings()
		}
		isSelf := hasPrincipal && principal.UserID == profile.UserID
		item := userItemFromProfile(profile, nil, isSelf || settings.PresenceVisible, isSelf || settings.LocationVisible, false)
		if item.Location != nil {
			item.Location.Distance, item.Location.BearingDegrees = computeRelativeDistance(req.ReferencePoint, profile.Location.Point)
		}
		if hasPrincipal && principal.UserID != "" && !isSelf {
			relation, err := s.repo.Relationship(ctx, principal.UserID, profile.UserID)
			if err != nil {
				return PaginatedUsers{}, err
			}
			item.Relation = &relation
		}
		items = append(items, item)
	}
	observability.L().Info("users.search completed", "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return PaginatedUsers{Items: items, NextCursor: profiles.NextCursor}, nil
}

func (s *Service) OpenTheDoor(ctx context.Context, userID string) (UserItem, bool, error) {
	startedAt := time.Now()
	observability.L().Info("users.open_the_door started", "user_id", userID)
	profile, ok, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		observability.L().Error("users.open_the_door failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserItem{}, false, err
	}
	if !ok {
		observability.L().Info("users.open_the_door not found", "user_id", userID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserItem{}, false, nil
	}
	settings, ok, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return UserItem{}, false, err
	}
	if !ok {
		settings = defaultSettings()
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	isSelf := hasPrincipal && principal.UserID == userID
	var relation *Relationship
	if hasPrincipal && principal.UserID != "" && !isSelf {
		rel, err := s.repo.Relationship(ctx, principal.UserID, userID)
		if err != nil {
			return UserItem{}, false, err
		}
		relation = &rel
	}
	item := userItemFromProfile(profile, relation, isSelf || settings.PresenceVisible, isSelf || settings.LocationVisible, isSelf || settings.StatsVisible)
	if relation == nil && hasPrincipal && isSelf {
		selfRelation := Relationship{}
		item.Relation = &selfRelation
	}
	if hasPrincipal && principal.UserID != "" && !isSelf {
		socialContext, err := s.buildSocialContext(ctx, principal.UserID, profile)
		if err != nil {
			return UserItem{}, false, err
		}
		item.SocialContext = socialContext
	}
	observability.L().Info("users.open_the_door completed", "user_id", userID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return item, true, nil
}

func (s *Service) UpdateLocation(ctx context.Context, req UpdateLocationRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.update_location started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.update_location failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		profile = s.defaultProfile(ctx, principal.UserID)
	}
	profile.Location = req.Location
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.update_location failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	event := realtime.Event{
		Type: realtime.EventTypeUserProfileUpdated,
		Payload: map[string]any{
			"user_id":    principal.UserID,
			"updated_at": now,
		},
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	realtime.DefaultHub().BroadcastToUserSubscribers(principal.UserID, event)
	observability.L().Info("users.update_location completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) Upgrade(ctx context.Context, req UpgradeAccountRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.upgrade started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		profile = s.defaultProfile(ctx, principal.UserID)
	}
	if profile.UserType == UserTypeNormal {
		return ErrConflict
	}
	profile.UserType = UserTypeNormal
	if strings.Contains(req.LoginID, "@") {
		profile.Email = req.LoginID
	} else {
		profile.Phone = req.LoginID
	}
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.upgrade completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) DeleteAccount(ctx context.Context, req DeleteAccountRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.delete_account started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if req.Confirmation != "DELETE_MY_ACCOUNT" {
		return ErrConflict
	}
	if err := s.repo.DeleteProfile(ctx, principal.UserID); err != nil {
		observability.L().Error("users.delete_account failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.delete_account completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) GetActorInfo(ctx context.Context, userID string) (actor.Info, error) {
	profile, ok, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return actor.Info{UserID: userID, Nickname: userID}, err
	}
	if !ok {
		return actor.Info{UserID: userID, Nickname: userID}, nil
	}
	return actor.Info{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: profile.AvatarURL}, nil
}

func requirePrincipal(ctx context.Context) (platformauth.Principal, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return platformauth.Principal{}, ErrUnauthorized
	}
	return principal, nil
}

func ensureUserExists(ctx context.Context, repo Repository, userID string) error {
	exists, err := repo.UserExists(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return nil
}

func (s *Service) defaultProfile(ctx context.Context, userID string) MyProfile {
	now := time.Now().UTC().Format(time.RFC3339)
	lat := 35.154922
	lng := 112.682524
	locName := "默认位置"
	if s.appConfig != nil {
		defaults := s.appConfig.Get(ctx).Defaults
		lat = defaults.DefaultLocation.Point.Latitude
		lng = defaults.DefaultLocation.Point.Longitude
		locName = defaults.DefaultLocation.LocationName
	}
	return MyProfile{UserID: userID, Nickname: userID, UserType: UserTypeTemp, Location: Location{Point: GeoPoint{Latitude: lat, Longitude: lng}, LocationName: locName}, PresenceStatus: UserPresenceOffline, LastOnline: now, CreatedAt: now, Stats: UserStats{}}
}

func defaultSettings() UserSettings {
	return UserSettings{PrivacyLevel: "public", LocationVisible: true, PresenceVisible: true, SocialListsVisible: true, GroupsVisible: true, LikedMomentsVisible: true, StatsVisible: true, AllowFollow: true, AllowPrivateChat: PrivateChatEveryone, Searchable: true, PhoneSearchable: false, NicknameSearchable: true, LocationSearchable: true}
}

func userItemFromProfile(profile MyProfile, relation *Relationship, includePresence bool, includeLocation bool, includeStats bool) UserItem {
	item := UserItem{User: UserInfo{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: profile.AvatarURL}, Bio: profile.Bio, Tags: profile.Tags, Relation: relation}
	if includePresence {
		item.PresenceStatus = profile.PresenceStatus
		item.LastOnline = profile.LastOnline
	}
	if includeLocation {
		item.Location = &RelativeLocation{Point: profile.Location.Point, LocationName: profile.Location.LocationName}
	}
	if includeStats {
		stats := profile.Stats
		item.Stats = &stats
	}
	return item
}

func computeRelativeDistance(reference GeoPoint, candidate GeoPoint) (int, int) {
	point := geo.Point{Latitude: reference.Latitude, Longitude: reference.Longitude}
	other := geo.Point{Latitude: candidate.Latitude, Longitude: candidate.Longitude}
	return geo.DistanceMeters(point, other), geo.BearingDegrees(point, other)
}
