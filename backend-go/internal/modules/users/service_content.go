package users

import (
	"context"
	"fmt"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/types"
)

func (s *Service) GetMySettings(ctx context.Context) (UserSettings, error) {
	op := observability.Begin("users.get_my_settings")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return UserSettings{}, err
	}
	if settings, ok, err := s.repo.GetSettings(ctx, principal.UserID); err != nil {
		op.Fail(err)
		return UserSettings{}, err
	} else if ok {
		op.Complete()
		return settings, nil
	}
	settings := defaultSettings()
	if err := s.repo.PutSettings(ctx, principal.UserID, settings); err != nil {
		op.Fail(err)
		return UserSettings{}, err
	}
	op.Complete()
	return settings, nil
}

// ChatSettingsFor returns the target user's private-chat permission setting.
// Explicit "by user id" read for cross-module callers (privatechats) — replaces
// the former hack of forging a Principal for the target user to reuse
// GetMySettings. found=false means the user never saved settings; callers
// should treat that as the default (everyone).
func (s *Service) ChatSettingsFor(ctx context.Context, userID string) (allowPrivateChat string, found bool, err error) {
	settings, ok, err := s.repo.GetSettings(ctx, userID)
	if err != nil || !ok {
		return "", ok, err
	}
	return string(settings.AllowPrivateChat), true, nil
}

// SettingsFor 按 user_id 读取目标用户的完整设置，纯读、无写副作用。
// 与 GetMySettings 的区别：后者在设置行缺失时会写入默认行（读路径带写副作用），
// 因此不能用于探测其他用户（会凭空为对方插入 settings 行，审计 F-10）。
// found=false 表示该用户从未保存过设置，调用方应按默认值处理。
func (s *Service) SettingsFor(ctx context.Context, userID string) (UserSettings, bool, error) {
	return s.repo.GetSettings(ctx, userID)
}

func (s *Service) UpdateMySettings(ctx context.Context, req UpdateSettingsRequest) error {
	op := observability.Begin("users.update_my_settings")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	// 枚举校验：非法 allow_private_chat 一旦落库，发送侧 switch 对未知值
	// fall-through 放行，隐私设置会静默失效（fail-open）。
	if req.Settings.AllowPrivateChat != nil && !req.Settings.AllowPrivateChat.Valid() {
		return fmt.Errorf("%w: allow_private_chat", ErrInvalidRequest)
	}
	settings, ok, err := s.repo.GetSettings(ctx, principal.UserID)
	if err != nil {
		op.Fail(err)
		return err
	}
	if !ok {
		settings = defaultSettings()
	}
	applySettingsPatch(&settings, req.Settings)
	if err := s.repo.PutSettings(ctx, principal.UserID, settings); err != nil {
		op.Fail(err)
		return err
	}
	// 不推送 self-sync 事件（设置状态不做跨设备实时同步，见 api_spec_index.md 预留）
	op.Complete()
	return nil
}

func (s *Service) ListReactedMoments(ctx context.Context, req UserReactedMomentsRequest) (momenttypes.PaginatedMoments, error) {
	op := observability.Begin("users.list_reacted_moments", "user_id", req.UserID)
	if req.UserID == "" {
		return momenttypes.PaginatedMoments{}, ErrInvalidRequest
	}
	if err := ensureUserExists(ctx, s.repo, req.UserID); err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	settings, ok, err := s.repo.GetSettings(ctx, req.UserID)
	if err != nil {
		op.Fail(err)
		return momenttypes.PaginatedMoments{}, err
	}
	if !ok {
		settings = defaultSettings()
	}
	viewerUserID := ""
	if hasPrincipal {
		viewerUserID = principal.UserID
	}
	if !settings.ReactedMomentsVisible && viewerUserID != req.UserID {
		return momenttypes.PaginatedMoments{}, ErrPermissionDenied
	}
	limit, cursor := normalizeMomentPagination(req.Pagination)
	if s.momentsByUser == nil {
		// provider 未装配（不可能发生于生产装配）：返回空分页而非报错，
		// 聚合数据缺失不阻断响应。
		observability.L().Warn("users.list_reacted_moments provider not wired")
		return momenttypes.PaginatedMoments{}, nil
	}
	result, err := s.momentsByUser.ListReactedByUser(ctx, req.UserID, viewerUserID, limit, cursor)
	if err != nil {
		op.Fail(err)
		return momenttypes.PaginatedMoments{}, err
	}
	for i := range result.Items {
		if result.Items[i].User.UserID != viewerUserID {
			result.Items[i].PrivacyLevel = ""
		}
	}
	op.Complete()
	return result, nil
}

func (s *Service) ListFriends(ctx context.Context, req ListFriendsRequest) (PaginatedFriends, error) {
	// 契约（spec api_spec_user.md）：user_id 必传，认证 optional，缺省不回退当前用户。
	targetUserID := req.UserID
	if targetUserID == "" {
		return PaginatedFriends{}, ErrInvalidRequest
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return PaginatedFriends{}, err
	}
	isSelf := hasPrincipal && principal.UserID == targetUserID
	if err := ensureSocialListsVisible(ctx, s.repo, targetUserID, isSelf); err != nil {
		return PaginatedFriends{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListFriends(ctx, targetUserID, limit, cursor)
	if err != nil {
		return PaginatedFriends{}, err
	}
	applyPresenceVisibilityToFriends(ctx, s.repo, result.Items)
	if !isSelf {
		for i := range result.Items {
			// 查他人列表 remark 恒定 null（spec api_spec_user.md RelatedUserItem.remark）。
			result.Items[i].Remark = nil
		}
	}
	return result, nil
}

func (s *Service) ListFollowing(ctx context.Context, req ListFollowingRequest) (PaginatedFollowing, error) {
	// 契约（spec api_spec_user.md）：user_id 必传，认证 optional，缺省不回退当前用户。
	targetUserID := req.UserID
	if targetUserID == "" {
		return PaginatedFollowing{}, ErrInvalidRequest
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return PaginatedFollowing{}, err
	}
	isSelf := hasPrincipal && principal.UserID == targetUserID
	if err := ensureSocialListsVisible(ctx, s.repo, targetUserID, isSelf); err != nil {
		return PaginatedFollowing{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListFollowing(ctx, targetUserID, limit, cursor)
	if err != nil {
		return PaginatedFollowing{}, err
	}
	applyPresenceVisibilityToFollowing(ctx, s.repo, result.Items)
	for i := range result.Items {
		// list_following 返回 RelatedUserItem。该端点不返回备注，恒为 null。
		result.Items[i].Remark = nil
	}
	return result, nil
}

func (s *Service) ListFollowers(ctx context.Context, req ListFollowersRequest) (PaginatedFollowers, error) {
	// 契约（spec api_spec_user.md）：user_id 必传，认证 optional，缺省不回退当前用户。
	targetUserID := req.UserID
	if targetUserID == "" {
		return PaginatedFollowers{}, ErrInvalidRequest
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return PaginatedFollowers{}, err
	}
	isSelf := hasPrincipal && principal.UserID == targetUserID
	if err := ensureSocialListsVisible(ctx, s.repo, targetUserID, isSelf); err != nil {
		return PaginatedFollowers{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	result, err := s.repo.ListFollowers(ctx, targetUserID, limit, cursor)
	if err != nil {
		return PaginatedFollowers{}, err
	}
	applyPresenceVisibilityToFollowers(ctx, s.repo, result.Items)
	for i := range result.Items {
		// list_followers 返回 RelatedUserItem。该端点不返回备注，恒为 null。
		result.Items[i].Remark = nil
	}
	return result, nil
}

func (s *Service) ListGroups(ctx context.Context, req UserGroupsRequest) (grouptypes.PaginatedGroups, error) {
	// 契约（spec api_spec_user.md）：user_id 必传，认证 optional，缺省不回退当前用户。
	targetUserID := req.UserID
	if targetUserID == "" {
		return grouptypes.PaginatedGroups{}, ErrInvalidRequest
	}
	principal, hasPrincipal := platformauth.PrincipalFromContext(ctx)
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
	if s.groupsByUser == nil {
		observability.L().Warn("users.list_groups provider not wired")
		return grouptypes.PaginatedGroups{}, nil
	}
	result, err := s.groupsByUser.ListGroupsByUser(ctx, targetUserID, limit, cursor)
	if err != nil {
		return grouptypes.PaginatedGroups{}, err
	}
	// 公开主页视角（spec api_spec_user.md）：只剥离查看者状态；
	// join_method 等加入门槛字段本就是公开字段；群管理字段已移入 GroupSettings。
	for i := range result.Items {
		result.Items[i].ViewerState = nil
	}
	return result, nil
}

func (s *Service) ListMoments(ctx context.Context, req UserMomentsRequest) (momenttypes.PaginatedMoments, error) {
	// 契约（spec api_spec_user.md）：user_id 必传，认证 optional，缺省不回退当前用户。
	// 纯公开视角：本端点只返回 privacy_level='public' 的瞬间；拥有者视角（含 private 与软删占位）
	// 见 moments/fetch_created。
	targetUserID := req.UserID
	if targetUserID == "" {
		return momenttypes.PaginatedMoments{}, ErrInvalidRequest
	}
	if err := ensureUserExists(ctx, s.repo, targetUserID); err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	limit, cursor := normalizePagination(req.Pagination)
	if s.momentsByUser == nil {
		observability.L().Warn("users.list_moments provider not wired")
		return momenttypes.PaginatedMoments{}, nil
	}
	result, err := s.momentsByUser.ListMomentsByAuthor(ctx, targetUserID, limit, cursor, false)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	for i := range result.Items {
		// privacy_level 属拥有者数据（OwnerMomentItem），公开视角不返回。
		result.Items[i].PrivacyLevel = ""
	}
	return result, nil
}

// GetProfileBundle 支撑 users/open_the_door 的 include_recent_moments 可选段
// （spec user.md → open_the_door：false/缺省仅返回 user，true 时附加主页首屏 recent_moments）。
func (s *Service) GetProfileBundle(ctx context.Context, userID string, includeRecentMoments bool) (UserProfileBundle, bool, error) {
	user, ok, err := s.OpenTheDoor(ctx, userID)
	if err != nil {
		return UserProfileBundle{}, false, err
	}
	if !ok {
		return UserProfileBundle{}, false, nil
	}
	bundle := UserProfileBundle{User: user}
	if !includeRecentMoments {
		return bundle, true, nil
	}
	momentLimit := 6
	if s.appConfig != nil {
		if n := s.appConfig.Get(ctx).ResponseDefaults.UserProfileMomentsCount; n > 0 {
			momentLimit = n
		}
	}
	moments, err := s.ListMoments(ctx, UserMomentsRequest{UserID: userID, Pagination: &PaginationParams{Limit: momentLimit}})
	if err != nil {
		return UserProfileBundle{}, false, err
	}
	bundle.RecentMoments = &moments
	return bundle, true, nil
}

func normalizePagination(p *PaginationParams) (int, string) {
	limit := 20
	cursor := ""
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if limit > types.MaxPageSize {
			limit = types.MaxPageSize
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
		if limit > types.MaxPageSize {
			limit = types.MaxPageSize
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
	if patch.ReactedMomentsVisible != nil {
		settings.ReactedMomentsVisible = *patch.ReactedMomentsVisible
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
	if patch.NearbyMomentNotify != nil {
		settings.NearbyMomentNotify = *patch.NearbyMomentNotify
	}
}

// NearbyMomentNotifyTargets 查询附近开启了「附近新瞬间通知」的用户
// （服务 moments 的附近新瞬间通知链）。委托 repo（postgres 包围盒 /
// memory 遍历），精确距离由调用方计算。
func (s *Service) NearbyMomentNotifyTargets(ctx context.Context, centerLat, centerLng float64, radiusMeters int, excludeUserID string) ([]NearbyNotifyTarget, error) {
	return s.repo.NearbyMomentNotifyTargets(ctx, centerLat, centerLng, radiusMeters, excludeUserID)
}
