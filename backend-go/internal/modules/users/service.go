package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// AccountCredentialWriter 由 auth 模块实现（bootstrap 注入，对齐
// SetContentProviders 等跨模块注入模式）：为既有用户写入登录凭证。
type AccountCredentialWriter interface {
	// AttachCredentials 将 login_id+password 写入登录凭证存储（auth_accounts）。
	// login_id 已被占用时返回包装了 ErrLoginIDTaken 的错误（映射 10400）；
	// ctx 携带事务时实现须加入该事务，与 user_type 同事务落库。
	AttachCredentials(ctx context.Context, userID, loginID, password string) error
}

// AccountAvatarMediaResolver 校验 avatar_file_id 归属（media 模块实现，
// bootstrap 注入，仿 moments 的 ImageResolver 模式）。
type AccountAvatarMediaResolver interface {
	// ResolveAvatarImage 校验 fileID 存在、归属 userID、category=avatar 且
	// 上传已完成；不满足时返回错误（update_account 映射 10300，
	// spec api_spec_user.md update_account 错误码）。
	ResolveAvatarImage(ctx context.Context, fileID, userID string) error
}

type Service struct {
	repo          Repository
	notifications userNotificationWriter
	appConfig     appconfig.Source
	// 跨模块读聚合（个人主页聚合），bootstrap 注入 groups/moments Service；
	// nil 时对应端点按 ErrInvalidRequest 之外的服务错误拒绝（不可静默空结果）。
	groupsByUser  GroupsByUserProvider
	momentsByUser MomentsByUserProvider
	// 凭证写入（upgrade），bootstrap 注入 auth 模块实现 + 事务执行器；
	// Upgrade 依赖二者，缺一按服务错误拒绝（见 ErrUpgradeUnavailable）。
	credentials AccountCredentialWriter
	runInTx     persistence.TxRunner
	// avatar_file_id 归属校验（update_account），bootstrap 注入 media 模块实现；
	// nil 时跳过校验（与 moments.validateImageOwnership 的 nil 口径一致）。
	avatarMedia AccountAvatarMediaResolver
}

// SetAvatarMediaResolver 注入 avatar_file_id 归属校验实现（bootstrap 装配）。
func (s *Service) SetAvatarMediaResolver(r AccountAvatarMediaResolver) { s.avatarMedia = r }

// SetCredentialWriter 注入凭证写入实现与事务执行器（bootstrap 装配）。
// Upgrade 在同一事务内写凭证与 user_type，二者须同时提供。
func (s *Service) SetCredentialWriter(w AccountCredentialWriter, runInTx persistence.TxRunner) {
	s.credentials = w
	s.runInTx = runInTx
}

// SetContentProviders 注入跨模块读聚合实现（groups/moments Service）。
func (s *Service) SetContentProviders(groups GroupsByUserProvider, moments MomentsByUserProvider) {
	s.groupsByUser = groups
	s.momentsByUser = moments
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

func (s *Service) GetMyProfile(ctx context.Context) (AccountData, error) {
	startedAt := time.Now()
	observability.L().Info("users.get_my_profile started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return AccountData{}, err
	}
	if profile, ok, err := s.repo.GetProfile(ctx, principal.UserID); err != nil {
		observability.L().Error("users.get_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AccountData{}, err
	} else if ok {
		observability.L().Info("users.get_my_profile completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return toAccountData(profile), nil
	}
	profile := s.defaultProfile(ctx, principal.UserID)
	if err := s.repo.PutProfile(ctx, profile); err != nil {
		observability.L().Error("users.get_my_profile failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AccountData{}, err
	}
	observability.L().Info("users.get_my_profile completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return toAccountData(profile), nil
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
		// 归属校验（spec api_spec_user.md update_account 错误码 10300）：
		// avatar_file_id 须来自当前用户已完成上传的 avatar 类媒体；空字符串
		// 表示清除头像，不校验。校验实现未接线时跳过（对齐 moments
		// validateImageOwnership 的 nil 口径，bootstrap 恒注入）。
		if *req.AvatarFileID != "" && s.avatarMedia != nil {
			if err := s.avatarMedia.ResolveAvatarImage(ctx, *req.AvatarFileID, principal.UserID); err != nil {
				observability.L().Warn("users.update_account avatar denied", "error", err, "user_id", principal.UserID)
				return ErrAvatarNotOwned
			}
		}
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
	var next *string
	if profiles.NextCursor != "" {
		next = &profiles.NextCursor
	}
	return PaginatedUsers{Items: items, NextCursor: next}, nil
}

func (s *Service) OpenTheDoor(ctx context.Context, userID string) (UserItem, bool, error) {
	startedAt := time.Now()
	observability.L().Debug("users.open_the_door started", "user_id", userID)
	profile, ok, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		observability.L().Error("users.open_the_door failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return UserItem{}, false, err
	}
	if !ok || profile.IsDeleted {
		// 已注销用户按不存在处理（spec api_spec_user.md：不再可被搜索/访问）
		observability.L().Debug("users.open_the_door not found", "user_id", userID, "elapsed_ms", time.Since(startedAt).Milliseconds())
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
	observability.L().Debug("users.open_the_door completed", "user_id", userID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return item, true, nil
}

// Relationship 直查双向关注关系（配额判定等只需要关系、不需要完整上下文的
// 热路径用；不走 OpenTheDoor 全量社交上下文）。
func (s *Service) Relationship(ctx context.Context, userID, targetUserID string) (Relationship, error) {
	return s.repo.Relationship(ctx, userID, targetUserID)
}

// BatchGet resolves multiple users at once (spec api_spec_user.md
// users/batch_fetch). Returns a user_id -> UserInfo map（共享类型，仅为补全
// 昵称/头像等基础信息，不携带 relation/stats 等查看者字段）；ids that do
// not exist (or are deleted) are simply absent from the map.
//
// User 本体不含任何 viewer 门控字段（门控只影响 presence/location/stats），
// 故一次 GetProfiles 批量取画像即与 OpenTheDoor 逐个解析等价，且天然跳过
// social_context 拼装——此前循环调 OpenTheDoor 把 15+ 查询/人的全量上下文
// （含好友列表分页求交）全部算完再只取 User，是 BatchGet 放大的主因。
func (s *Service) BatchGet(ctx context.Context, userIDs []string) (map[string]UserInfo, error) {
	result := make(map[string]UserInfo, len(userIDs))
	profiles, err := s.repo.GetProfiles(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, userID := range userIDs {
		profile, ok := profiles[userID]
		if !ok || profile.IsDeleted {
			continue
		}
		result[userID] = userItemFromProfile(profile, nil, false, false, false).User
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

func (s *Service) Upgrade(ctx context.Context, req UpgradeAccountRequest) (AccountData, error) {
	startedAt := time.Now()
	observability.L().Info("users.upgrade started")
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return AccountData{}, err
	}
	profile, ok, err := s.repo.GetProfile(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AccountData{}, err
	}
	if !ok {
		profile = s.defaultProfile(ctx, principal.UserID)
	}
	if profile.UserType == UserTypeNormal {
		return AccountData{}, ErrConflict
	}
	// 凭证写入（spec api_spec_user.md → upgrade 契约级语义：升级成功后
	// login_id+password 落登录凭证，否则升级用户无法重新登录）依赖 bootstrap
	// 注入的 auth 模块实现；未接线时拒绝升级，不可静默只改 user_type。
	if s.credentials == nil || s.runInTx == nil {
		observability.L().Error("users.upgrade failed: credential writer not wired", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AccountData{}, ErrUpgradeUnavailable
	}
	profile.UserType = UserTypeNormal
	// login_id 为用户名型（手机号/邮箱已被 handler 依 spec 拦截），故不再
	// 从 login_id 推断 profile.email/phone。凭证与 user_type 在同一事务落库：
	// 先写凭证，login_id 占用（唯一约束 → 10400）时整个事务回滚。
	writeAll := func(txCtx context.Context) error {
		if err := s.credentials.AttachCredentials(txCtx, principal.UserID, req.LoginID, req.Password); err != nil {
			if errors.Is(err, ErrLoginIDTaken) {
				return ErrLoginIDTaken
			}
			return fmt.Errorf("users.Upgrade: attach credentials: %w", err)
		}
		return s.repo.PutProfile(txCtx, profile)
	}
	if err := s.runInTx(ctx, writeAll); err != nil {
		observability.L().Error("users.upgrade failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AccountData{}, err
	}
	// Token/Session 语义（spec api_spec_user.md）：升级不吊销当前 session、不重发
	// token——user_type 更新后现有 token 继续有效，客户端无需重新登录。
	observability.L().Info("users.upgrade completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return toAccountData(profile), nil
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
	item := UserItem{User: UserInfo{UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: types.StrPtr(profile.AvatarURL)}, Tags: profile.Tags, Relation: relation}
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

// ListGroupsByUser / ListMomentsByAuthor / ListReactedByUser / CountCommonGroups
// 透传注入的跨模块聚合 provider（供测试把 Service 自身同时用作 provider 桩）。
func (s *Service) ListGroupsByUser(ctx context.Context, userID string, limit int, cursor string) (grouptypes.PaginatedGroups, error) {
	return s.groupsByUser.ListGroupsByUser(ctx, userID, limit, cursor)
}
func (s *Service) CountCommonGroups(ctx context.Context, viewerUserID, targetUserID string) (int, error) {
	return s.groupsByUser.CountCommonGroups(ctx, viewerUserID, targetUserID)
}
func (s *Service) ListMomentsByAuthor(ctx context.Context, userID string, limit int, cursor string, includePrivate bool) (momenttypes.PaginatedMoments, error) {
	return s.momentsByUser.ListMomentsByAuthor(ctx, userID, limit, cursor, includePrivate)
}
func (s *Service) ListReactedByUser(ctx context.Context, userID, viewerUserID string, limit int, cursor string) (momenttypes.PaginatedMoments, error) {
	return s.momentsByUser.ListReactedByUser(ctx, userID, viewerUserID, limit, cursor)
}
