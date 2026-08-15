package groups

import (
	"context"
	"fmt"
	"hotspot/backend-go/internal/platform/types"
	"slices"
	"strings"
	"time"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/rbac"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

const (
	// MaxMuteDurationMinutes is the maximum duration for muting a group member (30 days)
	MaxMuteDurationMinutes = 43200
)

// ownerIDValue dereferences a group's owner_id pointer (nil → "").
func ownerIDValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// isGroupOwner reports whether userID is the group owner (nil owner → false).
func isGroupOwner(owner *string, userID string) bool {
	return owner != nil && *owner == userID
}

// groupBlockedTagChecker 校验标签是否被平台屏蔽。由 bootstrap 注入 admin 实现；
// nil（未注入）时跳过校验（fail-open），避免创建群组被误拦。
type groupBlockedTagChecker interface {
	IsBlocked(tag string) bool
}

// blockedTagChecker 是包级注入点（bootstrap 通过 SetBlockedTagChecker 接入 admin 实现）。
// C.4：create 的 tags 含被屏蔽标签 → 10400。nil 时跳过校验。
var blockedTagChecker groupBlockedTagChecker

// SetBlockedTagChecker 注入被屏蔽标签检查器（bootstrap 接线）。
func SetBlockedTagChecker(checker groupBlockedTagChecker) {
	blockedTagChecker = checker
}

// serviceGroupRoleResolver 解析调用者在目标群的角色（守卫子树用）。由 bootstrap 注入
// groups 的 Service.RoleOf 实现。
var serviceGroupRoleResolver rbac.GroupRoleResolver

// SetGroupRoleResolver 注入群角色解析器（bootstrap 接线，守卫 fail-closed）。
func SetGroupRoleResolver(resolver rbac.GroupRoleResolver) {
	serviceGroupRoleResolver = resolver
}

// RecallMessageAsAdmin 由 bootstrap 注入 groupmessages 实现。未注入时 groups 返回 10600
// （管理员撤回任意成员消息仅经群守卫子树暴露，见 C.8）。
var RecallMessageAsAdmin func(ctx context.Context, groupID, messageID, operatorID string) error

type Service struct {
	repo          Repository
	notifications groupNotificationWriter
	users         actor.Provider
	locations     LocationProvider        // optional; nil => skip user-location fallback
	appConfig     appconfig.Source        // optional; nil => default TTLs for invite/application expiry
	audit         GroupDismissAuditWriter // optional; nil => dismiss audit is skipped
}

// GroupDismissAuditWriter records a group-dismiss audit entry (spec
// api_spec_group.md: the dismiss reason is audit-only and must never be
// surfaced to members). Implemented by the admin module and wired in
// bootstrap to avoid a groups → admin import cycle.
type GroupDismissAuditWriter interface {
	AppendGroupDismissAudit(ctx context.Context, operatorID string, groupID string, groupName string, reason string) error
}

// SetGroupDismissAuditWriter wires the optional audit writer (admin module).
func (s *Service) SetGroupDismissAuditWriter(w GroupDismissAuditWriter) {
	if s != nil {
		s.audit = w
	}
}

// LocationProvider 提供已登录用户最近一次 update_location 的位置(回退链第 2 级,
// api_spec_types.md「reference_point 解析规则」)。由 users 模块实现。
type LocationProvider interface {
	RecentLocation(ctx context.Context, userID string) (geo.Point, bool)
}

type groupNotificationWriter interface {
	AppendGroupInviteReceived(ctx context.Context, targetUserID string, actor types.UserInfo, groupID string, groupName string, inviteID string, message *string) (string, error)
	AppendGroupApplicationReceived(ctx context.Context, targetUserID string, applicant types.UserInfo, groupID string, groupName string, applicationID string, message *string) (string, error)
	AppendGroupApplicationAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string) (string, error)
	AppendGroupApplicationRejected(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string, reason *string) (string, error)
}

func NewService() *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), notify.NewService(), nil)
}

func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDeps(repo, notify.NewService(), nil)
}

func NewServiceWithDeps(repo Repository, notificationWriter groupNotificationWriter, actorProvider actor.Provider) *Service {
	return &Service{repo: repo, notifications: notificationWriter, users: actorProvider}
}

// SetLocationProvider wires the users-module location resolver (fallback level 2
// of the reference_point chain). Optional: nil skips the user-location level.
func (s *Service) SetLocationProvider(lp LocationProvider) { s.locations = lp }

// SetAppConfig wires the app config source (used for invite/application expiry
// TTLs). Optional: nil falls back to default TTLs.
func (s *Service) SetAppConfig(source appconfig.Source) { s.appConfig = source }

func (s *Service) Create(ctx context.Context, req CreateGroupRequest) (GroupItem, error) {
	startedAt := time.Now()
	observability.L().Info("groups.create started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// C.4：tags 含被屏蔽标签 → 10400（经注入的 BlockedTagChecker；未注入时跳过）
	if blockedTagChecker != nil {
		for _, tag := range req.Tags {
			if blockedTagChecker.IsBlocked(tag) {
				observability.L().Warn("groups.create blocked tag", "tag", tag)
				return GroupItem{}, ErrBusinessConflict
			}
		}
	}
	item, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("groups.create failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, fmt.Errorf("groups.Service.Create: repo: %w", err)
	}
	resourceguard.RecordGroup(ctx, item.Group.GroupID)
	observability.L().Info("groups.create completed", "group_id", item.Group.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return item, nil
}

func (s *Service) OpenTheDoor(ctx context.Context, groupID string) (GroupItem, bool, error) {
	startedAt := time.Now()
	observability.L().Info("groups.open_the_door started", "group_id", groupID)
	item, ok, err := s.repo.Get(ctx, groupID)
	if err != nil {
		observability.L().Error("groups.open_the_door failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, false, err
	}
	redactGroupFields(&item)
	observability.L().Info("groups.open_the_door completed", "group_id", groupID, "found", ok, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return item, ok, nil
}

// redactGroupFields enforces the spec's field-visibility rules (api_spec_types.md
// GroupItem) based on the viewer's membership/role carried in ViewerState.
// The admin-management fields (has_password/is_searchable/members_searchable/
// max_members) are no longer serialized on GroupItem (they live in GroupSettings,
// groups/admin/settings), so no JSON-level redaction is required for them
// here; the viewer's member state is carried by a nil-vs-set ViewerState.
//
// Call this at every serialization boundary.
func redactGroupFields(item *GroupItem) {
	if item == nil {
		return
	}
	// OwnerID is detail-page-only: search/list clear it to null separately.
	// The admin-management fields need no redaction because they are not
	// serialized on GroupItem (json:"-").
}

func (s *Service) Search(ctx context.Context, req SearchGroupsRequest) (PaginatedGroups, error) {
	startedAt := time.Now()
	observability.L().Info("groups.search started", "keyword", req.Keyword, "tag", req.Tag, "sort_by", req.SortBy)
	if req.ReferencePoint.Latitude == 0 && req.ReferencePoint.Longitude == 0 {
		req.ReferencePoint = s.resolveReferencePoint(ctx)
	}
	result, err := s.repo.Search(ctx, req)
	if err != nil {
		observability.L().Error("groups.search failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return PaginatedGroups{}, err
	}
	reference := searchGeoPoint(req.ReferencePoint)
	for i := range result.Items {
		candidate := groupLocationPoint(*result.Items[i].Location)
		distance := float64(geo.DistanceMeters(reference, candidate))
		bearing := float64(geo.BearingDegrees(reference, candidate))
		result.Items[i].Location.Distance = &distance
		result.Items[i].Location.BearingDegrees = &bearing
		// Search/list context: owner_id is detail-page-only per spec, and
		// member/admin fields are redacted per viewer membership/role.
		result.Items[i].OwnerID = nil
		redactGroupFields(&result.Items[i])
	}
	observability.L().Info("groups.search completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return result, nil
}

// resolveReferencePoint 实现 reference_point 回退链(api_spec_types.md):
// 1. 请求显式传入(非零值) → 直接使用
// 2. 已登录用户最近一次 update_location 的位置(匿名或未更新时跳过)
// 3. ClientConfig.defaults.default_location.point(服务端 appConfig 提供)
func (s *Service) resolveReferencePoint(ctx context.Context) GeoPoint {
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok && principal.UserID != "" && s.locations != nil {
		if pt, found := s.locations.RecentLocation(ctx, principal.UserID); found {
			return GeoPoint{Latitude: pt.Latitude, Longitude: pt.Longitude}
		}
	}
	lat, lng := geo.DefaultFallbackLatitude, geo.DefaultFallbackLongitude
	if s.appConfig != nil {
		defaults := s.appConfig.Get(ctx).Defaults
		lat, lng = defaults.DefaultLocation.Point.Latitude, defaults.DefaultLocation.Point.Longitude
	}
	return GeoPoint{Latitude: lat, Longitude: lng}
}

func (s *Service) Update(ctx context.Context, req UpdateGroupRequest) (GroupItem, error) {
	startedAt := time.Now()
	observability.L().Info("groups.update started", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.update failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, err
	}
	if !ok {
		observability.L().Warn("groups.update not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, ErrGroupNotFound
	}
	// C.1：update 权限从 owner-only 改为 owner/admin（守卫中间件已拦非 admin；此处防御性再判）
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		observability.L().Warn("groups.update permission denied", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, httpx.Forbidden(httpx.ReasonNotGroupAdmin, "非群管理员")
	}
	if err := s.repo.Update(ctx, principal.UserID, req); err != nil {
		observability.L().Error("groups.update failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, fmt.Errorf("groups.Service.Update: repo: %w", err)
	}
	// 重新读取以构造 updated_fields（对比变更前后）并返回更新后完整对象
	// （内容类更新返回更新后对象，spec api_spec_base.md「设计规范」）。
	updated, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.update refresh failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, fmt.Errorf("groups.Service.Update: refresh: %w", err)
	}
	if !ok {
		return GroupItem{}, ErrGroupNotFound
	}
	payload := map[string]any{
		"group_id":   req.GroupID,
		"updated_by": s.resolveUserInfo(ctx, principal.UserID),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if updatedFields := buildGroupUpdatedFields(group, updated, req); len(updatedFields) > 0 {
		payload["updated_fields"] = updatedFields
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
		Type:    realtime.EventTypeGroupUpdated,
		Payload: payload,
	})
	observability.L().Info("groups.update completed", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return updated, nil
}

// buildGroupUpdatedFields 计算更新前后 GroupItem 的可见变更字段（spec websocket.md GroupUpdatedPayload）。
// @group-admin-only 字段（is_searchable/max_members）有意不推送：群广播通道无法按订阅者角色过滤，
// 少推送是安全的（管理员经 HTTP open_the_door 获取）。has_password 为 @member-only，而群订阅者恒为成员，可推。
func buildGroupUpdatedFields(before, after GroupItem, req UpdateGroupRequest) map[string]any {
	fields := make(map[string]any)
	if after.Group.Name != before.Group.Name {
		fields["name"] = after.Group.Name
	}
	if after.Group.AvatarURL != before.Group.AvatarURL {
		fields["avatar_url"] = after.Group.AvatarURL
	}
	if after.Description != before.Description {
		description := ""
		if after.Description != nil {
			description = *after.Description
		}
		fields["description"] = description
	}
	if !slices.Equal(after.Tags, before.Tags) {
		fields["tags"] = after.Tags
	}
	if after.JoinMethod != before.JoinMethod {
		fields["join_method"] = after.JoinMethod
	}
	if after.RequiresApproval != before.RequiresApproval {
		fields["requires_approval"] = after.RequiresApproval
	}
	if !equalStringPtr(after.NextActivityAt, before.NextActivityAt) {
		fields["next_activity_at"] = derefStringPtr(after.NextActivityAt)
	}
	if req.Location != nil {
		fields["location"] = after.Location
	}
	return fields
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (s *Service) Dismiss(ctx context.Context, req DismissGroupRequest) error {
	startedAt := time.Now()
	observability.L().Info("groups.dismiss started", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.dismiss failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		observability.L().Warn("groups.dismiss not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return ErrGroupNotFound
	}
	if group.ViewerState == nil || group.ViewerState.Role != GroupRoleOwner {
		observability.L().Warn("groups.dismiss permission denied", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return ErrAccessRestricted
	}
	reason := ""
	if req.Reason != nil {
		reason = strings.TrimSpace(*req.Reason)
		if len(reason) > 500 {
			return ErrInvalidReason
		}
	}
	if err := s.repo.Dismiss(ctx, req.GroupID); err != nil {
		observability.L().Error("groups.dismiss failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return fmt.Errorf("groups.Service.Dismiss: repo: %w", err)
	}
	if s.audit != nil {
		if err := s.audit.AppendGroupDismissAudit(ctx, principal.UserID, group.Group.GroupID, group.Group.Name, reason); err != nil {
			observability.L().Error("groups.dismiss audit failed", "error", err, "group_id", req.GroupID)
		}
	}
	event := realtime.Event{Type: realtime.EventTypeGroupDismissed, Payload: map[string]any{"group_id": req.GroupID, "dismissed_at": time.Now().UTC().Format(time.RFC3339), "reason": "owner_dismissed"}}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, event)
	observability.L().Info("groups.dismiss completed", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
}

func (s *Service) ListMembers(ctx context.Context, req GroupMembersRequest) ([]GroupMemberItem, string, error) {
	startedAt := time.Now()
	observability.L().Info("groups.list_members started", "group_id", req.GroupID)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.list_members failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	if !ok {
		observability.L().Warn("groups.list_members not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", ErrGroupNotFound
	}
	// C.2：调用者非成员且群 members_searchable=false → 10200+not_group_member（httpx.Forbidden）
	principal, _ := platformauth.PrincipalFromContext(ctx)
	isMember := principal.UserID != ""
	if isMember {
		isMember, err = s.repo.IsMember(ctx, req.GroupID, principal.UserID)
		if err != nil {
			return nil, "", err
		}
	}
	if !isMember && !group.MembersSearchable {
		observability.L().Warn("groups.list_members not eligible", "group_id", req.GroupID, "user_id", principal.UserID)
		return nil, "", httpx.Forbidden(httpx.ReasonNotGroupMember, "非群成员")
	}
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	items, cursor, err := s.repo.ListMembers(ctx, req.GroupID, req.Role, limit, types.PaginationCursor(req.Pagination))
	if err != nil {
		observability.L().Error("groups.list_members failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	observability.L().Info("groups.list_members completed", "group_id", req.GroupID, "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return items, cursor, nil
}

func (s *Service) SearchMembers(ctx context.Context, req SearchGroupMembersRequest) ([]GroupMemberItem, string, error) {
	startedAt := time.Now()
	observability.L().Info("groups.search_members started", "group_id", req.GroupID)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.search_members failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	if !ok {
		observability.L().Warn("groups.search_members not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", ErrGroupNotFound
	}
	// 搜索资格（spec api_spec_group.md members/search 10400）：非成员仅当群设置
	// members_searchable=true 时可见成员列表；成员不受限。
	principal, _ := platformauth.PrincipalFromContext(ctx)
	isMember := principal.UserID != ""
	if isMember {
		isMember, err = s.repo.IsMember(ctx, req.GroupID, principal.UserID)
		if err != nil {
			observability.L().Error("groups.search_members failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
			return nil, "", err
		}
	}
	if !isMember && !group.MembersSearchable {
		observability.L().Warn("groups.search_members not eligible", "group_id", req.GroupID, "user_id", principal.UserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", httpx.Forbidden(httpx.ReasonNotGroupMember, "非群成员")
	}
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	items, cursor, err := s.repo.SearchMembers(ctx, req.GroupID, req.Keyword, req.Role, limit, types.PaginationCursor(req.Pagination))
	if err != nil {
		observability.L().Error("groups.search_members failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	observability.L().Info("groups.search_members completed", "group_id", req.GroupID, "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return items, cursor, nil
}

func (s *Service) ListMyGroups(ctx context.Context, req MyGroupsRequest) ([]GroupItem, string, error) {
	startedAt := time.Now()
	observability.L().Info("groups.list_my_groups started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	items, cursor, err := s.repo.ListMyGroups(ctx, principal.UserID, limit, types.PaginationCursor(req.Pagination))
	if err != nil {
		observability.L().Error("groups.list_my_groups failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	observability.L().Info("groups.list_my_groups completed", "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return items, cursor, nil
}

func (s *Service) JoinGroup(ctx context.Context, req EnterGroupRequest) (EnterGroupResult, error) {
	startedAt := time.Now()
	observability.L().Info("groups.join_group started", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.join_group failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return EnterGroupResult{}, err
	}
	if !ok {
		observability.L().Warn("groups.join_group not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return EnterGroupResult{}, ErrGroupNotFound
	}
	if group.ViewerState != nil {
		observability.L().Info("groups.join_group already joined", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return EnterGroupResult{Status: EnterGroupStatusJoined}, nil
	}
	// C.3：被封禁用户无法加入群，返回 10200+banned_in_group（httpx.Forbidden）
	if banned, err := s.repo.IsBanned(ctx, req.GroupID, principal.UserID); err != nil {
		return EnterGroupResult{}, err
	} else if banned {
		observability.L().Warn("groups.join_group banned", "group_id", req.GroupID, "user_id", principal.UserID)
		return EnterGroupResult{}, httpx.Forbidden(httpx.ReasonBannedInGroup, "已被群封禁，无法加入")
	}
	// spec group.md → join 收口：invite_only 群不支持 join（加入唯一入口为
	// groups/invites/accept），一律返回 10400。
	if group.JoinMethod == GroupJoinMethodInviteOnly {
		observability.L().Warn("groups.join_group invite_only not supported", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return EnterGroupResult{}, ErrBusinessConflict
	}
	result, err := s.repo.JoinGroup(ctx, principal.UserID, req.GroupID, req.Password, req.Message)
	if err != nil {
		observability.L().Error("groups.join_group failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return EnterGroupResult{}, fmt.Errorf("groups.Service.JoinGroup: repo: %w", err)
	}
	if result.Status == EnterGroupStatusPendingApproval && result.IsNewApplication {
		// 仅新创建的申请才通知管理员；命中已有 pending 申请（自然去重）不重复通知
		s.notifyGroupApplicationReceived(ctx, group, principal.UserID, result.ApplicationID, req.Message)
	}
	if result.Status == EnterGroupStatusJoined {
		// 直接加入（spec group.md → join 推送）：向群内广播 GroupMemberJoined
		realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
			Type: realtime.EventTypeGroupMemberJoined,
			Payload: map[string]any{
				"group_id":     req.GroupID,
				"user":         s.resolveUserInfo(ctx, principal.UserID),
				"role":         string(GroupRoleMember),
				"member_count": group.Group.MembersCount,
				"joined_at":    time.Now().UTC().Format(time.RFC3339),
			},
		})
	}
	observability.L().Info("groups.join_group completed", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return result, nil
}

func (s *Service) CreateAnnouncement(ctx context.Context, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	startedAt := time.Now()
	observability.L().Info("groups.create_announcement started", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.create_announcement failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, err
	}
	if !ok {
		observability.L().Warn("groups.create_announcement not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		observability.L().Warn("groups.create_announcement permission denied", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, ErrAccessRestricted
	}
	item, err := s.repo.CreateAnnouncement(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("groups.create_announcement failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, err
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
		Type: realtime.EventTypeGroupAnnouncementCreated,
		Payload: map[string]any{
			"group_id":        req.GroupID,
			"announcement_id": item.AnnouncementID,
			"title":           item.Title,
			"created_by":      s.resolveUserInfo(ctx, principal.UserID),
			"created_at":      item.CreatedAt,
		},
	})
	observability.L().Info("groups.create_announcement completed", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return item, nil
}

func (s *Service) ListAnnouncements(ctx context.Context, req GroupAnnouncementsRequest) ([]GroupAnnouncementItem, error) {
	startedAt := time.Now()
	observability.L().Info("groups.list_announcements started", "group_id", req.GroupID)
	items, err := s.repo.ListAnnouncements(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.list_announcements failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, err
	}
	observability.L().Info("groups.list_announcements completed", "group_id", req.GroupID, "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return items, nil
}

func (s *Service) UpdateAnnouncement(ctx context.Context, req UpdateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	if !ok {
		return GroupAnnouncementItem{}, ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return GroupAnnouncementItem{}, ErrAccessRestricted
	}
	if err := s.repo.UpdateAnnouncement(ctx, req.GroupID, req); err != nil {
		return GroupAnnouncementItem{}, err
	}
	// 内容类更新返回更新后完整对象（spec api_spec_base.md「设计规范」）
	items, err := s.repo.ListAnnouncements(ctx, req.GroupID)
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	for _, item := range items {
		if item.AnnouncementID == req.AnnouncementID {
			return item, nil
		}
	}
	return GroupAnnouncementItem{}, ErrAnnouncementNotFound
}

func (s *Service) DeleteAnnouncement(ctx context.Context, req DeleteGroupAnnouncementRequest) error {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	return s.repo.DeleteAnnouncement(ctx, req.GroupID, req.AnnouncementID)
}

func (s *Service) UpdatePreferences(ctx context.Context, req UpdateGroupPreferencesRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.ViewerState == nil {
		return ErrNotMember
	}
	return s.repo.UpdatePreferences(ctx, principal.UserID, req)
}

func (s *Service) LeaveGroup(ctx context.Context, req LeaveGroupRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.ViewerState == nil {
		return ErrNotMember
	}
	if group.ViewerState.Role == GroupRoleOwner {
		return ErrAccessRestricted
	}
	if err := s.repo.LeaveGroup(ctx, principal.UserID, req.GroupID); err != nil {
		return fmt.Errorf("groups.Service.LeaveGroup: repo: %w", err)
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
		Type: realtime.EventTypeGroupMemberLeft,
		Payload: map[string]any{
			"group_id": req.GroupID,
			"user":     s.resolveUserInfo(ctx, principal.UserID),
			"left_at":  time.Now().UTC().Format(time.RFC3339),
		},
	})
	return nil
}

func (s *Service) ListMyApplications(ctx context.Context, req MyApplicationsRequest) ([]GroupApplicationItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListMyApplications(ctx, principal.UserID, req.Status, limit, types.PaginationCursor(req.Pagination))
}

func (s *Service) CancelApplication(ctx context.Context, req CancelApplicationRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	application, ok, err := s.repo.GetApplication(ctx, req.ApplicationID)
	if err != nil {
		return err
	}
	if !ok || application.Applicant.UserID != principal.UserID {
		return ErrApplicationNotFound
	}
	if req.GroupID != application.GroupID {
		// Group-level auth anchor mismatch: report not-found so callers cannot
		// probe applications across groups (spec api_spec_group.md).
		return ErrApplicationNotFound
	}
	reason := ""
	if req.Reason != nil {
		reason = strings.TrimSpace(*req.Reason)
		if len(reason) > 200 {
			return ErrInvalidReason
		}
	}
	if err := s.repo.CancelApplication(ctx, req.ApplicationID); err != nil {
		return err
	}
	s.notifyGroupApplicationCancelled(ctx, application, reason)
	return nil
}

// notifyGroupApplicationCancelled pushes GroupApplicationCancelled events to
// the group admins. 纯实时事件（spec api_spec_websocket.md：不落库通知）。
func (s *Service) notifyGroupApplicationCancelled(ctx context.Context, application GroupApplicationItem, reason string) {
	group, groupFound, err := s.repo.Get(ctx, application.GroupID)
	if err != nil || !groupFound {
		return
	}
	applicant := s.resolveUserInfo(ctx, application.Applicant.UserID)
	now := time.Now().UTC().Format(time.RFC3339)
	admins, _, err := s.repo.ListMembers(ctx, group.Group.GroupID, nil, 1000, "")
	if err != nil {
		return
	}
	for _, member := range admins {
		if member.Role != GroupRoleOwner && member.Role != GroupRoleAdmin {
			continue
		}
		payload := map[string]any{
			"application_id": application.ApplicationID,
			"group_id":       group.Group.GroupID,
			"applicant":      applicant,
			"cancelled_at":   now,
		}
		if reason != "" {
			payload["reason"] = reason
		}
		realtime.DefaultHub().BroadcastToUser(member.User.UserID, realtime.Event{
			Type:    realtime.EventTypeGroupApplicationCancelled,
			Payload: payload,
		})
	}
}

// notifyGroupApplicationExpired persists group_application_cancelled
// notifications (applicant present, no reason) and pushes
// GroupApplicationExpired events to the group admins.
func (s *Service) notifyGroupApplicationExpired(ctx context.Context, group GroupItem, application GroupApplicationItem) {
	applicant := s.resolveUserInfo(ctx, application.Applicant.UserID)
	now := time.Now().UTC().Format(time.RFC3339)
	admins, _, err := s.repo.ListMembers(ctx, group.Group.GroupID, nil, 1000, "")
	if err != nil {
		return
	}
	for _, member := range admins {
		if member.Role != GroupRoleOwner && member.Role != GroupRoleAdmin {
			continue
		}
		realtime.DefaultHub().BroadcastToUser(member.User.UserID, realtime.Event{
			Type: realtime.EventTypeGroupApplicationExpired,
			Payload: map[string]any{
				"application_id": application.ApplicationID,
				"group_id":       group.Group.GroupID,
				"applicant":      applicant,
				"expired_at":     now,
			},
		})
	}
}

// notifyGroupApplicationRejected persists a group_application_rejected
// notification and pushes GroupApplicationRejected to the applicant.
// F7: 不暴露审批者身份（与 accepted 对称），事件与通知不含 rejected_by。
func (s *Service) notifyGroupApplicationRejected(ctx context.Context, group GroupItem, application GroupApplicationItem, reason *string) {
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendGroupApplicationRejected(ctx, application.Applicant.UserID, group.Group.GroupID, group.Group.Name, application.ApplicationID, reason)
		if err != nil {
			observability.L().Error("groups.notify_group_application_rejected append failed", "error", err, "application_id", application.ApplicationID)
		} else {
			notificationID = id
		}
	}
	payload := map[string]any{
		"notify_id":      notificationID,
		"application_id": application.ApplicationID,
		"group": map[string]any{
			"group_id":      group.Group.GroupID,
			"name":          group.Group.Name,
			"avatar_url":    group.Group.AvatarURL,
			"members_count": group.Group.MembersCount,
		},
		"rejected_at": time.Now().UTC().Format(time.RFC3339),
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	realtime.DefaultHub().BroadcastToUser(application.Applicant.UserID, realtime.Event{Type: realtime.EventTypeGroupApplicationRejected, Payload: payload})
}

// applicationExpired reports whether a pending application is past its TTL
// (AppConfig.limits.group_application_ttl_hours, default 72h).
func (s *Service) applicationExpired(ctx context.Context, item GroupApplicationItem) bool {
	ttl := 72 * time.Hour
	if s.appConfig != nil {
		if ttlHours := s.appConfig.Get(ctx).Limits.GroupApplicationTTLHours; ttlHours > 0 {
			ttl = time.Duration(ttlHours) * time.Hour
		}
	}
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return false
	}
	return time.Now().UTC().After(createdAt.Add(ttl))
}

func groupUserInfo(user UserInfo) map[string]any {
	payload := map[string]any{"user_id": user.UserID, "nickname": user.Nickname}
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		payload["avatar_url"] = *user.AvatarURL
	}
	return payload
}

func (s *Service) resolveUserInfo(ctx context.Context, userID string) types.UserInfo {
	if s.users != nil {
		info, err := s.users.GetActorInfo(ctx, userID)
		if err == nil {
			return types.UserInfo{UserID: info.UserID, Nickname: info.Nickname, AvatarURL: types.StrPtr(info.AvatarURL)}
		}
	}
	return types.UserInfo{UserID: userID, Nickname: userID}
}

func (s *Service) notifyGroupInviteReceived(ctx context.Context, group GroupItem, inviterUserID string, req InviteMemberRequest, invitation Invitation) {
	actor := s.resolveUserInfo(ctx, inviterUserID)
	inviteID := ""
	if invitation.InviteID != nil {
		inviteID = *invitation.InviteID
	}
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendGroupInviteReceived(ctx, req.UserID, actor, group.Group.GroupID, group.Group.Name, inviteID, req.Message)
		if err != nil {
			observability.L().Error("groups.notify AppendGroupInviteReceived failed", "error", err, "invite_id", inviteID)
		} else {
			notificationID = id
		}
	}
	payload := map[string]any{
		"notify_id": notificationID,
		"invite_id": inviteID,
		"group": map[string]any{
			"group_id":      group.Group.GroupID,
			"name":          group.Group.Name,
			"avatar_url":    group.Group.AvatarURL,
			"members_count": group.Group.MembersCount,
		},
		"actor":      actor, // spec websocket.md：统一 actor 命名（原 inviter）
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	if req.Message != nil && *req.Message != "" {
		payload["message"] = *req.Message
	}
	realtime.DefaultHub().BroadcastToUser(req.UserID, realtime.Event{Type: realtime.EventTypeGroupInviteReceived, Payload: payload})
}

// notifyGroupInviteAccepted pushes the GroupInviteResponse event to the
// inviter. 纯实时事件（spec api_spec_websocket.md：不落库通知）。
func (s *Service) notifyGroupInviteAccepted(ctx context.Context, invite GroupInviteItem, userID string) {
	user := s.resolveUserInfo(ctx, userID)
	realtime.DefaultHub().BroadcastToUser(invite.Inviter.UserID, realtime.Event{Type: realtime.EventTypeGroupInviteResponse, Payload: map[string]any{"invite_id": invite.InviteID, "group_id": invite.Group.GroupID, "user": user, "responded_at": time.Now().UTC().Format(time.RFC3339)}})
}

func (s *Service) notifyGroupApplicationReceived(ctx context.Context, group GroupItem, applicantUserID string, applicationID *string, message *string) {
	if applicationID == nil {
		return
	}
	applicant := s.resolveUserInfo(ctx, applicantUserID)
	admins, _, err := s.repo.ListMembers(ctx, group.Group.GroupID, nil, 1000, "")
	if err != nil {
		return
	}
	for _, member := range admins {
		if member.Role != GroupRoleOwner && member.Role != GroupRoleAdmin {
			continue
		}
		notificationID := ""
		if s.notifications != nil {
			id, err := s.notifications.AppendGroupApplicationReceived(ctx, member.User.UserID, applicant, group.Group.GroupID, group.Group.Name, *applicationID, message)
			if err != nil {
				observability.L().Error("groups.notify AppendGroupApplicationReceived failed", "error", err, "application_id", applicationID)
			} else {
				notificationID = id
			}
		}
		payload := map[string]any{"notify_id": notificationID, "application_id": applicationID, "group_id": group.Group.GroupID, "applicant": applicant, "created_at": time.Now().UTC().Format(time.RFC3339)}
		if message != nil && *message != "" {
			payload["message"] = *message
		}
		realtime.DefaultHub().BroadcastToUser(member.User.UserID, realtime.Event{Type: realtime.EventTypeGroupApplicationReceived, Payload: payload})
	}
}

func (s *Service) notifyGroupApplicationAccepted(ctx context.Context, group GroupItem, application GroupApplicationItem) {
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendGroupApplicationAccepted(ctx, application.Applicant.UserID, group.Group.GroupID, group.Group.Name, application.ApplicationID)
		if err != nil {
			observability.L().Error("groups.notify_group_application_accepted append failed", "error", err, "application_id", application.ApplicationID)
		} else {
			notificationID = id
		}
	}
	realtime.DefaultHub().BroadcastToUser(application.Applicant.UserID, realtime.Event{Type: realtime.EventTypeGroupApplicationResponse, Payload: map[string]any{"notify_id": notificationID, "application_id": application.ApplicationID, "group": map[string]any{"group_id": group.Group.GroupID, "name": group.Group.Name, "avatar_url": group.Group.AvatarURL, "members_count": group.Group.MembersCount}, "responded_at": time.Now().UTC().Format(time.RFC3339)}})
}
