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
	"hotspot/backend-go/internal/platform/types"
)

type Service struct {
	repo          Repository
	notifications userNotificationWriter
	appConfig     appconfig.Source
}

type userNotificationWriter interface {
	AppendNewFollower(ctx context.Context, targetUserID string, actor types.UserInfo, source string, sourceMomentID string) (string, error)
	AppendMutualFollow(ctx context.Context, targetUserID string, actor types.UserInfo) (string, error)
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

func (s *Service) UpdateMyProfile(ctx context.Context, req UpdateAccountRequest) error {
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
		// 契约（spec api_spec_user.md update_account）：tags 含被屏蔽标签 → 10400。
		if blocked, err := s.repo.CheckBlockedTags(ctx, *req.Tags); err != nil {
			observability.L().Error("users.update_my_profile blocked-tags check failed", "error", err)
			return err
		} else if len(blocked) > 0 {
			return ErrBlockedTag
		}
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

func (s *Service) Search(ctx context.Context, req SearchUsersRequest) (PaginatedUsers, error) {
	startedAt := time.Now()
	observability.L().Info("users.search started")
	// reference_point 缺失时按回退链解析:用户存储位置 → default_location
	req.ReferencePoint = s.resolveReferencePoint(ctx, req.ReferencePoint)
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
	// Batch-load settings once for the whole page to avoid an N+1 of GetSettings calls.
	searchUserIDs := make([]string, 0, len(profiles.Items))
	for _, profile := range profiles.Items {
		searchUserIDs = append(searchUserIDs, profile.UserID)
	}
	settingsByUserID, err := s.repo.GetSettingsBatch(ctx, searchUserIDs)
	if err != nil {
		return PaginatedUsers{}, err
	}
	// Batch-load relationships once for the whole page to avoid an N+1 of
	// Relationship calls. Only meaningful when the caller is authenticated.
	var relationsByUserID map[string]Relationship
	if hasPrincipal && principal.UserID != "" {
		targets := make([]string, 0, len(profiles.Items))
		for _, profile := range profiles.Items {
			if profile.UserID == principal.UserID {
				continue
			}
			targets = append(targets, profile.UserID)
		}
		relationsByUserID, err = s.repo.RelationshipBatch(ctx, principal.UserID, targets)
		if err != nil {
			return PaginatedUsers{}, err
		}
	}
	for _, profile := range profiles.Items {
		settings, ok := settingsByUserID[profile.UserID]
		if !ok {
			settings = defaultSettings()
		}
		isSelf := hasPrincipal && principal.UserID == profile.UserID
		item := userItemFromProfile(profile, nil, isSelf || settings.PresenceVisible, isSelf || settings.LocationVisible, false)
		if item.Location != nil {
			d, b := computeRelativeDistance(req.ReferencePoint, profile.Location.Point)
			d64 := float64(d)
			b64 := float64(b)
			item.Location.Distance = &d64
			item.Location.BearingDegrees = &b64
		}
		if relationsByUserID != nil && !isSelf {
			if relation, ok := relationsByUserID[profile.UserID]; ok && (relation.IsFollowing || relation.IsFollowedBy || relation.IsBlocked) {
				item.Relation = &relation
			}
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
	if !ok || profile.IsDeleted {
		// 已注销用户按不存在处理（spec api_spec_user.md：不再可被搜索/访问）
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
		// 无关系（三态皆假/匿名/本人）→ null（spec api_spec_base.md 恒定 nullable）
		if rel.IsFollowing || rel.IsFollowedBy || rel.IsBlocked {
			relation = &rel
		}
	}
	item := userItemFromProfile(profile, relation, isSelf || settings.PresenceVisible, isSelf || settings.LocationVisible, isSelf || settings.StatsVisible)
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

// BatchGet resolves multiple users at once (spec api_spec_user.md users/batch_get).
// Returns a user_id -> UserItem map; ids that do not exist (or fail privacy
// checks) are simply absent from the map. Each user is resolved through the
// same OpenTheDoor privacy path so visibility rules stay consistent with the
// single-user endpoint.
func (s *Service) BatchGet(ctx context.Context, userIDs []string) (map[string]UserItem, error) {
	result := make(map[string]UserItem, len(userIDs))
	for _, userID := range userIDs {
		item, ok, err := s.OpenTheDoor(ctx, userID)
		if err != nil {
			observability.L().Error("users.batch_get single lookup failed", "error", err, "user_id", userID)
			continue
		}
		if !ok {
			continue
		}
		result[userID] = item
	}
	return result, nil
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

func (s *Service) Upgrade(ctx context.Context, req UpgradeAccountRequest) (MyProfile, error) {
	startedAt := time.Now()
	observability.L().Info("users.upgrade started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return MyProfile{}, err
	}
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return MyProfile{}, err
	}
	if !ok {
		profile = s.defaultProfile(ctx, principal.UserID)
	}
	if profile.UserType == UserTypeNormal {
		return MyProfile{}, ErrConflict
	}
	profile.UserType = UserTypeNormal
	// 推断规则（spec api_spec_user.md AccountData.email/phone）：
	// 含 '@' → email；纯数字 → phone；用户名 → 两字段均省略。
	switch {
	case strings.Contains(req.LoginID, "@"):
		profile.Email = req.LoginID
	case isNumericLoginID(req.LoginID):
		profile.Phone = req.LoginID
	}
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return MyProfile{}, err
	}
	// Token/Session 语义（spec api_spec_user.md）：升级不吊销当前 session、不重发
	// token——user_type 更新后现有 token 继续有效，客户端无需重新登录。
	observability.L().Info("users.upgrade completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return profile, nil
}

// isNumericLoginID 判断 login_id 是否为纯数字（手机号）。
func isNumericLoginID(loginID string) bool {
	if loginID == "" {
		return false
	}
	for _, r := range loginID {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *Service) DeleteAccount(ctx context.Context, req DeleteAccountRequest) error {
	startedAt := time.Now()
	observability.L().Info("users.delete_account started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return err
	}
	// 契约（spec api_spec_user.md delete_account）：confirmation 必须精确匹配，否则 10000。
	if req.Confirmation != "DELETE_MY_ACCOUNT" {
		return ErrInvalidRequest
	}
	if err := s.repo.DeleteProfile(ctx, principal.UserID); err != nil {
		observability.L().Error("users.delete_account failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("users.delete_account completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

// RecentLocation 返回用户最近一次 update_location 存储的位置(回退链第 2 级,
// 见 api_spec_types.md「reference_point 解析规则」)。用户不存在或从未更新过
// 位置时返回 found=false,由调用方继续回退到 ClientConfig.defaults.default_location。
func (s *Service) RecentLocation(ctx context.Context, userID string) (geo.Point, bool) {
	if userID == "" {
		return geo.Point{}, false
	}
	profile, ok, err := s.repo.GetProfile(ctx, userID)
	if err != nil || !ok {
		return geo.Point{}, false
	}
	pt := profile.Location.Point
	if pt.Latitude == 0 && pt.Longitude == 0 {
		return geo.Point{}, false
	}
	return geo.Point{Latitude: pt.Latitude, Longitude: pt.Longitude}, true
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
	lat := geo.DefaultFallbackLatitude
	lng := geo.DefaultFallbackLongitude
	locName := "默认位置"
	if s.appConfig != nil {
		defaults := s.appConfig.Get(ctx).Defaults
		lat = defaults.DefaultLocation.Point.Latitude
		lng = defaults.DefaultLocation.Point.Longitude
		locName = defaults.DefaultLocation.LocationName
	}
	return MyProfile{UserID: userID, Nickname: userID, UserType: UserTypeTemp, Location: Location{Point: GeoPoint{Latitude: lat, Longitude: lng}, LocationName: locName}, PresenceStatus: UserPresenceOffline, LastOnline: now, CreatedAt: now, Stats: UserStats{}}
}

// resolveReferencePoint 实现 reference_point 回退链(api_spec_types.md):
// 1. 显式传入(非零值) → 直接使用
// 2. 已登录用户最近一次 users/me/location/update 的位置
// 3. ClientConfig.defaults.default_location.point
func (s *Service) resolveReferencePoint(ctx context.Context, ref GeoPoint) GeoPoint {
	if ref.Latitude != 0 || ref.Longitude != 0 {
		return ref
	}
	// 回退 2:已登录用户的存储位置
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok && principal.UserID != "" {
		if profile, ok, err := s.repo.GetProfile(ctx, principal.UserID); err == nil && ok {
			if profile.Location.Point.Latitude != 0 || profile.Location.Point.Longitude != 0 {
				return profile.Location.Point
			}
		}
	}
	// 回退 3:default_location
	lat := geo.DefaultFallbackLatitude
	lng := geo.DefaultFallbackLongitude
	if s.appConfig != nil {
		defaults := s.appConfig.Get(ctx).Defaults
		lat = defaults.DefaultLocation.Point.Latitude
		lng = defaults.DefaultLocation.Point.Longitude
	}
	return GeoPoint{Latitude: lat, Longitude: lng}
}

func defaultSettings() UserSettings {
	return UserSettings{PrivacyLevel: "public", LocationVisible: true, PresenceVisible: true, SocialListsVisible: true, GroupsVisible: true, ReactedMomentsVisible: true, StatsVisible: true, AllowFollow: true, AllowPrivateChat: ChatPermEveryone, Searchable: true, PhoneSearchable: false, NicknameSearchable: true, LocationSearchable: true, NearbyMomentNotify: false}
}

func userItemFromProfile(profile MyProfile, relation *Relationship, includePresence bool, includeLocation bool, includeStats bool) UserItem {
	item := UserItem{User: UserInfo{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: profile.AvatarURL}, Tags: profile.Tags, Relation: relation}
	if profile.Bio != "" {
		bio := profile.Bio
		item.Bio = &bio
	}
	if includePresence {
		ps := profile.PresenceStatus
		item.PresenceStatus = &ps
		lastOnline := profile.LastOnline
		item.LastOnline = &lastOnline
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
