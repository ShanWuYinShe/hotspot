package users

import (
	"context"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

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
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeSelfSettingsUpdated, Payload: map[string]any{
		"settings": settings,
	}})
	observability.L().Info("users.update_my_settings completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
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
