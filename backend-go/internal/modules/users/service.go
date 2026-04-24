package users

import (
	"context"
	"strings"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/notifications"
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
	return NewServiceWithDeps(NewInMemoryRepository(), notifications.NewService(), nil)
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
	profile := s.defaultProfile(ctx,principal.UserID)
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
		profile = s.defaultProfile(ctx,principal.UserID)
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
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{
		Type:    "UserProfileUpdated",
		Payload: map[string]any{"user_id": principal.UserID},
	})
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

func (s *Service) GetMySettings(ctx context.Context) (UserSettings, error) {
	startedAt := time.Now()
	observability.L().Info("users.get_my_settings started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return UserSettings{}, err
	}
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
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	settings, ok, err := s.repo.GetSettings(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.update_my_settings failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		settings = defaultSettings()
	}
	applySettingsPatch(&settings, req.Settings)
	if err := s.repo.PutSettings(ctx, principal.UserID, settings); err != nil {
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
		profile = s.defaultProfile(ctx,principal.UserID)
	}
	profile.Location = req.Location
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.update_location failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{
		Type:    "UserProfileUpdated",
		Payload: map[string]any{"user_id": principal.UserID},
	})
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
		profile = s.defaultProfile(ctx,principal.UserID)
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

func (s *Service) Follow(ctx context.Context, targetUserID string, source FollowSource, sourceMomentID string) error {
	startedAt := time.Now()
	observability.L().Info("users.follow started", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	settings, ok, err := s.repo.GetSettings(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !ok {
		settings = defaultSettings()
	}
	if !settings.AllowFollow {
		return ErrPermissionDenied
	}
	if err := s.repo.Follow(ctx, principal.UserID, targetUserID, source, sourceMomentID); err != nil {
		observability.L().Error("users.follow failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	actor := map[string]any{"user_id": principal.UserID, "nickname": principal.UserID}
	followedAt := time.Now().UTC().Format(time.RFC3339)
	sourceValue := strings.TrimSpace(string(source))
	sourceMomentID = strings.TrimSpace(sourceMomentID)
	if sourceValue != string(FollowSourceMoment) {
		sourceMomentID = ""
	}
	if s.notifications != nil {
		if err := s.notifications.AppendNewFollower(ctx, targetUserID, actor, sourceValue, sourceMomentID); err != nil {
			observability.L().Error("users.follow notification append failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
			return err
		}
	}
	newFollowerPayload := map[string]any{"actor": actor, "followed_at": followedAt}
	if sourceValue != "" {
		newFollowerPayload["source"] = sourceValue
	}
	if sourceMomentID != "" {
		newFollowerPayload["source_moment_id"] = sourceMomentID
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: "NewFollower", Payload: newFollowerPayload})
	relation, err := s.repo.Relationship(ctx, targetUserID, principal.UserID)
	if err != nil {
		observability.L().Error("users.follow relationship check failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if relation.IsFollowing {
		mutualAt := time.Now().UTC().Format(time.RFC3339)
		if s.notifications != nil {
			if err := s.notifications.AppendMutualFollow(ctx, principal.UserID, actor); err != nil {
				observability.L().Error("users.follow mutual notification append failed", "error", err, "target_user_id", principal.UserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
				return err
			}
			if err := s.notifications.AppendMutualFollow(ctx, targetUserID, actor); err != nil {
				observability.L().Error("users.follow mutual notification append failed", "error", err, "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
				return err
			}
		}
		mutualPayload := map[string]any{"actor": actor, "mutual_at": mutualAt}
		realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: "MutualFollow", Payload: mutualPayload})
		realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: "MutualFollow", Payload: mutualPayload})
	}
	observability.L().Info("users.follow completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) Unfollow(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.unfollow started", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	if err := s.repo.Unfollow(ctx, principal.UserID, targetUserID); err != nil {
		observability.L().Error("users.unfollow failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.unfollow completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) Block(ctx context.Context, targetUserID, reason string) error {
	startedAt := time.Now()
	observability.L().Info("users.block started", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	if err := s.repo.Block(ctx, principal.UserID, targetUserID, reason); err != nil {
		observability.L().Error("users.block failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.block completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) Unblock(ctx context.Context, targetUserID string) error {
	startedAt := time.Now()
	observability.L().Info("users.unblock started", "target_user_id", targetUserID)
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if targetUserID == principal.UserID {
		return ErrConflict
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
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
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return err
	}
	relation, err := s.repo.Relationship(ctx, principal.UserID, targetUserID)
	if err != nil {
		return err
	}
	if !relation.IsFollowing {
		return ErrConflict
	}
	if err := s.repo.SetRemark(ctx, principal.UserID, targetUserID, remark); err != nil {
		observability.L().Error("users.set_remark failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.set_remark completed", "target_user_id", targetUserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) ListBlocked(ctx context.Context, req MyBlockedRequest) (PaginatedBlockedUsers, error) {
	startedAt := time.Now()
	observability.L().Info("users.list_blocked started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return PaginatedBlockedUsers{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	items, err := s.repo.ListBlocked(ctx, principal.UserID, limit, cursor)
	if err != nil {
		observability.L().Error("users.list_blocked failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return PaginatedBlockedUsers{}, err
	}
	observability.L().Info("users.list_blocked completed", "count", len(items.Items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return items, nil
}

func (s *Service) ListLikedMoments(ctx context.Context, req UserLikedMomentsRequest) (momenttypes.PaginatedMoments, error) {
	startedAt := time.Now()
	observability.L().Info("users.list_liked_moments started", "user_id", req.UserID)
	if err := ensureUserExists(ctx, s.repo, req.UserID); err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	settings, ok, err := s.repo.GetSettings(ctx, req.UserID)
	if err != nil {
		observability.L().Error("users.list_liked_moments failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return momenttypes.PaginatedMoments{}, err
	}
	if !ok {
		settings = defaultSettings()
	}
	viewerUserID := ""
	if hasPrincipal {
		viewerUserID = principal.UserID
	}
	if !settings.LikedMomentsVisible && viewerUserID != req.UserID {
		return momenttypes.PaginatedMoments{}, ErrPermissionDenied
	}
	limit, cursor := normalizeMomentPagination(req.Pagination)
	result, err := s.repo.ListLikedMoments(ctx, req.UserID, limit, cursor, viewerUserID)
	if err != nil {
		observability.L().Error("users.list_liked_moments failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return momenttypes.PaginatedMoments{}, err
	}
	for i := range result.Items {
		if result.Items[i].User.UserID != viewerUserID {
			result.Items[i].PrivacyLevel = ""
		}
	}
	observability.L().Info("users.list_liked_moments completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return result, nil
}

func (s *Service) ListFriends(ctx context.Context, req ListFriendsRequest) (PaginatedFriends, error) {
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	targetUserID := req.UserID
	if targetUserID == "" {
		if !hasPrincipal || principal.UserID == "" {
			return PaginatedFriends{}, ErrUnauthorized
		}
		targetUserID = principal.UserID
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return PaginatedFriends{}, err
	}
	if err := ensureSocialListsVisible(ctx, s.repo, targetUserID, hasPrincipal && principal.UserID == targetUserID); err != nil {
		return PaginatedFriends{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListFriends(ctx, targetUserID, limit, cursor)
	if err != nil {
		return PaginatedFriends{}, err
	}
	applyPresenceVisibilityToFriends(ctx, s.repo, result.Items)
	if !(hasPrincipal && principal.UserID == targetUserID) {
		for i := range result.Items {
			result.Items[i].Remark = ""
		}
	}
	return result, nil
}

func (s *Service) ListFollowing(ctx context.Context, req ListFollowingRequest) (PaginatedFollowing, error) {
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	targetUserID := req.UserID
	if targetUserID == "" {
		if !hasPrincipal || principal.UserID == "" {
			return PaginatedFollowing{}, ErrUnauthorized
		}
		targetUserID = principal.UserID
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return PaginatedFollowing{}, err
	}
	if err := ensureSocialListsVisible(ctx, s.repo, targetUserID, hasPrincipal && principal.UserID == targetUserID); err != nil {
		return PaginatedFollowing{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListFollowing(ctx, targetUserID, limit, cursor)
	if err != nil {
		return PaginatedFollowing{}, err
	}
	applyPresenceVisibilityToFollowing(ctx, s.repo, result.Items)
	return result, nil
}

func (s *Service) ListFollowers(ctx context.Context, req ListFollowersRequest) (PaginatedFollowers, error) {
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	targetUserID := req.UserID
	if targetUserID == "" {
		if !hasPrincipal || principal.UserID == "" {
			return PaginatedFollowers{}, ErrUnauthorized
		}
		targetUserID = principal.UserID
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return PaginatedFollowers{}, err
	}
	if err := ensureSocialListsVisible(ctx, s.repo, targetUserID, hasPrincipal && principal.UserID == targetUserID); err != nil {
		return PaginatedFollowers{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListFollowers(ctx, targetUserID, limit, cursor)
	if err != nil {
		return PaginatedFollowers{}, err
	}
	applyPresenceVisibilityToFollowers(ctx, s.repo, result.Items)
	return result, nil
}

func (s *Service) ListGroups(ctx context.Context, req UserGroupsRequest) (grouptypes.PaginatedGroups, error) {
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	targetUserID := req.UserID
	if targetUserID == "" {
		if !hasPrincipal || principal.UserID == "" {
			return grouptypes.PaginatedGroups{}, ErrUnauthorized
		}
		targetUserID = principal.UserID
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return grouptypes.PaginatedGroups{}, err
	}
	settings, ok, err := s.repo.GetSettings(ctx, targetUserID)
	if err != nil {
		return grouptypes.PaginatedGroups{}, err
	}
	if !ok {
		settings = defaultSettings()
	}
	isSelf := hasPrincipal && principal.UserID == targetUserID
	if !isSelf && !settings.GroupsVisible {
		return grouptypes.PaginatedGroups{}, ErrPermissionDenied
	}
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListGroups(ctx, targetUserID, limit, cursor)
	if err != nil {
		return grouptypes.PaginatedGroups{}, err
	}
	for i := range result.Items {
		result.Items[i].MyState = nil
		result.Items[i].JoinMethod = ""
		result.Items[i].IsSearchable = false
		result.Items[i].MaxMembers = 0
	}
	return result, nil
}

func (s *Service) ListMoments(ctx context.Context, req UserMomentsRequest) (momenttypes.PaginatedMoments, error) {
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	targetUserID := req.UserID
	if targetUserID == "" {
		if !hasPrincipal || principal.UserID == "" {
			return momenttypes.PaginatedMoments{}, ErrUnauthorized
		}
		targetUserID = principal.UserID
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	isSelf := hasPrincipal && principal.UserID == targetUserID
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListMoments(ctx, targetUserID, limit, cursor, isSelf)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	for i := range result.Items {
		if !isSelf {
			result.Items[i].PrivacyLevel = ""
		}
	}
	return result, nil
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

func ensureSocialListsVisible(ctx context.Context, repo Repository, userID string, isSelf bool) error {
	if isSelf {
		return nil
	}
	settings, ok, err := repo.GetSettings(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		settings = defaultSettings()
	}
	if !settings.SocialListsVisible {
		return ErrPermissionDenied
	}
	return nil
}

func normalizePagination(p *PaginationParams) (int, string) {
	limit := 20
	cursor := ""
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if limit > 100 {
			limit = 100
		}
		cursor = p.Cursor
	}
	return limit, cursor
}

func normalizeMomentPagination(p *momenttypes.PaginationParams) (int, string) {
	limit := 20
	cursor := ""
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if limit > 100 {
			limit = 100
		}
		cursor = p.Cursor
	}
	return limit, cursor
}

func applySettingsPatch(settings *UserSettings, patch PartialUserSettings) {
	if patch.PrivacyLevel != nil {
		settings.PrivacyLevel = *patch.PrivacyLevel
	}
	if patch.LocationVisible != nil {
		settings.LocationVisible = *patch.LocationVisible
	}
	if patch.PresenceVisible != nil {
		settings.PresenceVisible = *patch.PresenceVisible
	}
	if patch.SocialListsVisible != nil {
		settings.SocialListsVisible = *patch.SocialListsVisible
	}
	if patch.GroupsVisible != nil {
		settings.GroupsVisible = *patch.GroupsVisible
	}
	if patch.LikedMomentsVisible != nil {
		settings.LikedMomentsVisible = *patch.LikedMomentsVisible
	}
	if patch.StatsVisible != nil {
		settings.StatsVisible = *patch.StatsVisible
	}
	if patch.AllowFollow != nil {
		settings.AllowFollow = *patch.AllowFollow
	}
	if patch.AllowPrivateChat != nil {
		settings.AllowPrivateChat = *patch.AllowPrivateChat
	}
	if patch.Searchable != nil {
		settings.Searchable = *patch.Searchable
	}
	if patch.PhoneSearchable != nil {
		settings.PhoneSearchable = *patch.PhoneSearchable
	}
	if patch.NicknameSearchable != nil {
		settings.NicknameSearchable = *patch.NicknameSearchable
	}
	if patch.LocationSearchable != nil {
		settings.LocationSearchable = *patch.LocationSearchable
	}
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

func applyPresenceVisibilityToFriends(ctx context.Context, repo Repository, items []FriendItem) {
	for i := range items {
		if !isPresenceVisible(ctx, repo, items[i].User.UserID) {
			items[i].PresenceStatus = nil
			items[i].LastOnline = nil
		}
	}
}

func applyPresenceVisibilityToFollowing(ctx context.Context, repo Repository, items []FollowingItem) {
	for i := range items {
		if !isPresenceVisible(ctx, repo, items[i].User.UserID) {
			items[i].PresenceStatus = nil
			items[i].LastOnline = nil
		}
	}
}

func applyPresenceVisibilityToFollowers(ctx context.Context, repo Repository, items []FollowerItem) {
	for i := range items {
		if !isPresenceVisible(ctx, repo, items[i].User.UserID) {
			items[i].PresenceStatus = nil
			items[i].LastOnline = nil
		}
	}
}

func isPresenceVisible(ctx context.Context, repo Repository, userID string) bool {
	settings, ok, err := repo.GetSettings(ctx, userID)
	if err != nil {
		return true
	}
	if !ok {
		settings = defaultSettings()
	}
	return settings.PresenceVisible
}

func computeRelativeDistance(reference GeoPoint, candidate GeoPoint) (int, int) {
	point := geo.Point{Latitude: reference.Latitude, Longitude: reference.Longitude}
	other := geo.Point{Latitude: candidate.Latitude, Longitude: candidate.Longitude}
	return geo.DistanceMeters(point, other), geo.BearingDegrees(point, other)
}

func (s *Service) buildSocialContext(ctx context.Context, viewerUserID string, targetProfile MyProfile) (*SocialContext, error) {
	viewerProfile, ok, err := s.repo.GetProfile(ctx, viewerUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		viewerProfile = s.defaultProfile(ctx,viewerUserID)
	}
	mutualFriends, err := collectMutualFriends(ctx, s.repo, viewerUserID, targetProfile.UserID)
	if err != nil {
		return nil, err
	}
	context := &SocialContext{MutualFriendsCount: len(mutualFriends)}
	if len(mutualFriends) > 0 {
		preview := make([]UserInfo, 0, minInt(len(mutualFriends), 3))
		for _, friend := range mutualFriends {
			preview = append(preview, friend.User)
			if len(preview) == 3 {
				break
			}
		}
		context.MutualFriendsPreview = preview
	}
	targetSettings, ok, err := s.repo.GetSettings(ctx, targetProfile.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		targetSettings = defaultSettings()
	}
	if targetSettings.GroupsVisible {
		commonGroupsCount, err := s.repo.CommonGroupCount(ctx, viewerUserID, targetProfile.UserID)
		if err != nil {
			return nil, err
		}
		context.CommonGroupsCount = &commonGroupsCount
	}
	commonTags := intersectTags(viewerProfile.Tags, targetProfile.Tags)
	if len(commonTags) > 0 {
		context.CommonTags = commonTags
	}
	return context, nil
}

func collectMutualFriends(ctx context.Context, repo Repository, viewerUserID, targetUserID string) ([]FriendItem, error) {
	viewerFriends, err := collectFriendsByUser(ctx, repo, viewerUserID)
	if err != nil {
		return nil, err
	}
	targetFriends, err := collectFriendsByUser(ctx, repo, targetUserID)
	if err != nil {
		return nil, err
	}
	targetSet := make(map[string]struct{}, len(targetFriends))
	for _, item := range targetFriends {
		targetSet[item.User.UserID] = struct{}{}
	}
	mutual := make([]FriendItem, 0)
	for _, item := range viewerFriends {
		if _, ok := targetSet[item.User.UserID]; ok {
			mutual = append(mutual, item)
		}
	}
	return mutual, nil
}

func collectFriendsByUser(ctx context.Context, repo Repository, userID string) ([]FriendItem, error) {
	items := make([]FriendItem, 0)
	cursor := ""
	for {
		page, err := repo.ListFriends(ctx, userID, 100, cursor)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return items, nil
}

func intersectTags(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	seen := make(map[string]string, len(left))
	for _, item := range left {
		key := strings.ToLower(item)
		if _, ok := seen[key]; !ok {
			seen[key] = item
		}
	}
	common := make([]string, 0)
	added := make(map[string]struct{})
	for _, item := range right {
		key := strings.ToLower(item)
		original, ok := seen[key]
		if !ok {
			continue
		}
		if _, dup := added[key]; dup {
			continue
		}
		common = append(common, original)
		added[key] = struct{}{}
	}
	return common
}

func minInt(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
