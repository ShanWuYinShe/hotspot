package groups

import (
	"context"
	"fmt"
	"slices"
	"time"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

const (
	// MaxMuteDurationMinutes is the maximum duration for muting a group member (30 days)
	MaxMuteDurationMinutes = 43200
)

type Service struct {
	repo          Repository
	notifications groupNotificationWriter
	users         actor.Provider
	locations     LocationProvider // optional; nil => skip user-location fallback
	appConfig     appconfig.Source // optional; nil => default TTLs for invite/application expiry
}

// LocationProvider 提供已登录用户最近一次 update_location 的位置(回退链第 2 级,
// api_spec_types.md「reference_point 解析规则」)。由 users 模块实现。
type LocationProvider interface {
	RecentLocation(ctx context.Context, userID string) (geo.Point, bool)
}

type groupNotificationWriter interface {
	AppendGroupInviteReceived(ctx context.Context, targetUserID string, actor map[string]any, groupID string, groupName string, inviteID string, message *string) (string, error)
	AppendGroupInviteAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, inviteID string, user map[string]any) (string, error)
	AppendGroupApplicationReceived(ctx context.Context, targetUserID string, applicant map[string]any, groupID string, groupName string, applicationID string) (string, error)
	AppendGroupApplicationAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string) (string, error)
	AppendGroupInviteRevoked(ctx context.Context, targetUserID string, groupID string, groupName string, inviteID string, revokedBy map[string]any, reason *string) (string, error)
	AppendGroupInviteRejected(ctx context.Context, targetUserID string, groupID string, groupName string, inviteID string, rejectedBy map[string]any, reason *string) (string, error)
	AppendGroupApplicationCancelled(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string, applicant map[string]any, reason *string) (string, error)
	AppendGroupApplicationRejected(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string, rejectedBy map[string]any, reason *string) (string, error)
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
// GroupItem) based on the viewer's membership/role carried in MyState:
//   - has_password is @member-only — hidden from non-members (don't tip off
//     crawlers/brute-forcers about which groups have passwords)
//   - join_method / requires_approval are @all — visible to everyone (users must
//     know how to join before attempting to)
//   - is_searchable / max_members are @group-admin-only
//   - owner_id is detail-page-only (kept here; search/list redact it separately)
//
// Call this at every serialization boundary.
func redactGroupFields(item *GroupItem) {
	if item == nil {
		return
	}
	isMember := item.MyState != nil && item.MyState.IsJoined
	if !isMember {
		item.HasPassword = false
	}
	isAdmin := item.MyState != nil && (item.MyState.Role == GroupRoleOwner || item.MyState.Role == GroupRoleAdmin)
	if !isAdmin {
		item.IsSearchable = false
		item.MaxMembers = 0
	}
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
		candidate := groupLocationPoint(result.Items[i].Location)
		distance := float64(geo.DistanceMeters(reference, candidate))
		bearing := float64(geo.BearingDegrees(reference, candidate))
		result.Items[i].Location.Distance = &distance
		result.Items[i].Location.BearingDegrees = &bearing
		// Search/list context: owner_id is detail-page-only per spec, and
		// member/admin fields are redacted per viewer membership/role.
		result.Items[i].OwnerID = ""
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

func (s *Service) Update(ctx context.Context, req UpdateGroupRequest) error {
	startedAt := time.Now()
	observability.L().Info("groups.update started", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.update failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	if !ok {
		observability.L().Warn("groups.update not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return ErrGroupNotFound
	}
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		observability.L().Warn("groups.update permission denied", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return ErrAccessRestricted
	}
	if err := s.repo.Update(ctx, principal.UserID, req); err != nil {
		observability.L().Error("groups.update failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return fmt.Errorf("groups.Service.Update: repo: %w", err)
	}
	// 重新读取以构造 updated_fields（对比变更前后）；受控字段见 buildGroupUpdatedFields。
	updated, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.update refresh failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return fmt.Errorf("groups.Service.Update: refresh: %w", err)
	}
	if !ok {
		return ErrGroupNotFound
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
	return nil
}

// buildGroupUpdatedFields 计算更新前后 GroupItem 的可见变更字段（spec websocket.md GroupUpdatedPayload）。
// @group-admin-only 字段（is_searchable/max_members）有意不推送：群广播通道无法按订阅者角色过滤，
// 少推送是安全的（管理员经 HTTP get_detail 获取）。has_password 为 @member-only，而群订阅者恒为成员，可推。
func buildGroupUpdatedFields(before, after GroupItem, req UpdateGroupRequest) map[string]any {
	fields := make(map[string]any)
	if after.Group.Name != before.Group.Name {
		fields["name"] = after.Group.Name
	}
	if after.Group.AvatarURL != before.Group.AvatarURL {
		fields["avatar_url"] = after.Group.AvatarURL
	}
	if after.Description != before.Description {
		fields["description"] = after.Description
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
	if !equalStringPtr(after.NextActivity, before.NextActivity) {
		fields["next_activity"] = derefStringPtr(after.NextActivity)
	}
	if after.HasPassword != before.HasPassword {
		fields["has_password"] = after.HasPassword
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
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		observability.L().Warn("groups.dismiss permission denied", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return ErrAccessRestricted
	}
	if err := s.repo.Dismiss(ctx, req.GroupID); err != nil {
		observability.L().Error("groups.dismiss failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return fmt.Errorf("groups.Service.Dismiss: repo: %w", err)
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
	_, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.list_members failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	if !ok {
		observability.L().Warn("groups.list_members not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", ErrGroupNotFound
	}
	limit := paginationLimit(req.Pagination, 20, 100)
	items, cursor, err := s.repo.ListMembers(ctx, req.GroupID, req.Role, limit, paginationCursor(req.Pagination))
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
	_, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.search_members failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	if !ok {
		observability.L().Warn("groups.search_members not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", ErrGroupNotFound
	}
	limit := paginationLimit(req.Pagination, 20, 100)
	items, cursor, err := s.repo.SearchMembers(ctx, req.GroupID, req.Keyword, req.Role, limit, paginationCursor(req.Pagination))
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
	limit := paginationLimit(req.Pagination, 20, 100)
	items, cursor, err := s.repo.ListMyGroups(ctx, principal.UserID, limit, paginationCursor(req.Pagination))
	if err != nil {
		observability.L().Error("groups.list_my_groups failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, "", err
	}
	observability.L().Info("groups.list_my_groups completed", "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return items, cursor, nil
}

func (s *Service) JoinGroup(ctx context.Context, req JoinGroupRequest) (JoinGroupResult, error) {
	startedAt := time.Now()
	observability.L().Info("groups.join_group started", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.join_group failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return JoinGroupResult{}, err
	}
	if !ok {
		observability.L().Warn("groups.join_group not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return JoinGroupResult{}, ErrGroupNotFound
	}
	if group.MyState != nil && group.MyState.IsJoined {
		observability.L().Info("groups.join_group already joined", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return JoinGroupResult{Status: JoinGroupStatusJoined}, nil
	}
	result, err := s.repo.JoinGroup(ctx, principal.UserID, req.GroupID, req.Password, req.Message)
	if err != nil {
		observability.L().Error("groups.join_group failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return JoinGroupResult{}, fmt.Errorf("groups.Service.JoinGroup: repo: %w", err)
	}
	if result.Status == JoinGroupStatusPendingApproval {
		s.notifyGroupApplicationReceived(ctx, group, principal.UserID, result.Application, req.Message)
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
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
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

func (s *Service) UpdateAnnouncement(ctx context.Context, req UpdateGroupAnnouncementRequest) error {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	return s.repo.UpdateAnnouncement(ctx, req.GroupID, req)
}

func (s *Service) DeleteAnnouncement(ctx context.Context, req DeleteGroupAnnouncementRequest) error {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	return s.repo.DeleteAnnouncement(ctx, req.GroupID, req.AnnouncementID)
}

func (s *Service) UpdateMySettings(ctx context.Context, req UpdateGroupMySettingsRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || !group.MyState.IsJoined {
		return ErrNotMember
	}
	return s.repo.UpdateMySettings(ctx, principal.UserID, req)
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
	if group.MyState == nil || !group.MyState.IsJoined {
		return ErrNotMember
	}
	if group.MyState.Role == GroupRoleOwner {
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
	limit := paginationLimit(req.Pagination, 20, 100)
	return s.repo.ListMyApplications(ctx, principal.UserID, req.Status, limit, paginationCursor(req.Pagination))
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
	if err := s.repo.CancelApplication(ctx, req.ApplicationID); err != nil {
		return err
	}
	s.notifyGroupApplicationCancelled(ctx, application)
	return nil
}

// notifyGroupApplicationCancelled persists group_application_cancelled
// notifications and pushes GroupApplicationCancelled events to the group admins.
func (s *Service) notifyGroupApplicationCancelled(ctx context.Context, application GroupApplicationItem) {
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
		notificationID := ""
		if s.notifications != nil {
			id, err := s.notifications.AppendGroupApplicationCancelled(ctx, member.User.UserID, group.Group.GroupID, group.Group.Name, application.ApplicationID, applicant, nil)
			if err != nil {
				observability.L().Error("groups.notify_group_application_cancelled append failed", "error", err, "application_id", application.ApplicationID)
			} else {
				notificationID = id
			}
		}
		realtime.DefaultHub().BroadcastToUser(member.User.UserID, realtime.Event{
			Type: realtime.EventTypeGroupApplicationCancelled,
			Payload: map[string]any{
				"notification_id": notificationID,
				"application_id":  application.ApplicationID,
				"group_id":        group.Group.GroupID,
				"applicant":       applicant,
				"cancelled_at":    now,
			},
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
		notificationID := ""
		if s.notifications != nil {
			id, err := s.notifications.AppendGroupApplicationCancelled(ctx, member.User.UserID, group.Group.GroupID, group.Group.Name, application.ApplicationID, applicant, nil)
			if err != nil {
				observability.L().Error("groups.notify_group_application_expired append failed", "error", err, "application_id", application.ApplicationID)
			} else {
				notificationID = id
			}
		}
		realtime.DefaultHub().BroadcastToUser(member.User.UserID, realtime.Event{
			Type: realtime.EventTypeGroupApplicationExpired,
			Payload: map[string]any{
				"notification_id": notificationID,
				"application_id":  application.ApplicationID,
				"group_id":        group.Group.GroupID,
				"applicant":       applicant,
				"expired_at":      now,
			},
		})
	}
}

// notifyGroupApplicationRejected persists a group_application_rejected
// notification and pushes GroupApplicationRejected to the applicant.
func (s *Service) notifyGroupApplicationRejected(ctx context.Context, group GroupItem, application GroupApplicationItem, rejectedByUserID string, reason *string) {
	rejectedBy := s.resolveUserInfo(ctx, rejectedByUserID)
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendGroupApplicationRejected(ctx, application.Applicant.UserID, group.Group.GroupID, group.Group.Name, application.ApplicationID, rejectedBy, reason)
		if err != nil {
			observability.L().Error("groups.notify_group_application_rejected append failed", "error", err, "application_id", application.ApplicationID)
		} else {
			notificationID = id
		}
	}
	payload := map[string]any{
		"notification_id": notificationID,
		"application_id":  application.ApplicationID,
		"group": map[string]any{
			"group_id":      group.Group.GroupID,
			"name":          group.Group.Name,
			"avatar_url":    group.Group.AvatarURL,
			"members_count": group.Group.MembersCount,
		},
		"rejected_by": rejectedBy,
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
	if user.AvatarURL != "" {
		payload["avatar_url"] = user.AvatarURL
	}
	return payload
}

func (s *Service) resolveUserInfo(ctx context.Context, userID string) map[string]any {
	if s.users != nil {
		info, err := s.users.GetActorInfo(ctx, userID)
		if err == nil {
			actor := map[string]any{"user_id": info.UserID, "nickname": info.Nickname}
			if info.AvatarURL != "" {
				actor["avatar_url"] = info.AvatarURL
			}
			return actor
		}
	}
	return groupUserInfo(UserInfo{UserID: userID, Nickname: userID})
}

func (s *Service) notifyGroupInviteReceived(ctx context.Context, group GroupItem, inviterUserID string, req InviteMemberRequest, invitation Invitation) {
	actor := s.resolveUserInfo(ctx, inviterUserID)
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendGroupInviteReceived(ctx, req.UserID, actor, group.Group.GroupID, group.Group.Name, invitation.InviteID, req.Message)
		if err != nil {
			observability.L().Error("groups.notify AppendGroupInviteReceived failed", "error", err, "invite_id", invitation.InviteID)
		} else {
			notificationID = id
		}
	}
	payload := map[string]any{
		"notification_id": notificationID,
		"invite_id":       invitation.InviteID,
		"group": map[string]any{
			"group_id":      group.Group.GroupID,
			"name":          group.Group.Name,
			"avatar_url":    group.Group.AvatarURL,
			"members_count": group.Group.MembersCount,
		},
		"inviter":    actor,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	if req.Message != nil && *req.Message != "" {
		payload["message"] = *req.Message
	}
	realtime.DefaultHub().BroadcastToUser(req.UserID, realtime.Event{Type: realtime.EventTypeGroupInviteReceived, Payload: payload})
}

func (s *Service) notifyGroupInviteAccepted(ctx context.Context, invite GroupInviteItem, userID string) {
	user := s.resolveUserInfo(ctx, userID)
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendGroupInviteAccepted(ctx, invite.Inviter.UserID, invite.Group.GroupID, invite.Group.Name, invite.InviteID, user)
		if err != nil {
			observability.L().Error("groups.notify_group_invite_accepted append failed", "error", err, "invite_id", invite.InviteID)
		} else {
			notificationID = id
		}
	}
	realtime.DefaultHub().BroadcastToUser(invite.Inviter.UserID, realtime.Event{Type: realtime.EventTypeGroupInviteResponse, Payload: map[string]any{"notification_id": notificationID, "invite_id": invite.InviteID, "group_id": invite.Group.GroupID, "user": user, "responded_at": time.Now().UTC().Format(time.RFC3339)}})
}

func (s *Service) notifyGroupApplicationReceived(ctx context.Context, group GroupItem, applicantUserID string, application *struct {
	ApplicationID string `json:"application_id"`
}, message *string) {
	if application == nil {
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
			id, err := s.notifications.AppendGroupApplicationReceived(ctx, member.User.UserID, applicant, group.Group.GroupID, group.Group.Name, application.ApplicationID)
			if err != nil {
				observability.L().Error("groups.notify AppendGroupApplicationReceived failed", "error", err, "application_id", application.ApplicationID)
			} else {
				notificationID = id
			}
		}
		payload := map[string]any{"notification_id": notificationID, "application_id": application.ApplicationID, "group_id": group.Group.GroupID, "applicant": applicant, "created_at": time.Now().UTC().Format(time.RFC3339)}
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
	realtime.DefaultHub().BroadcastToUser(application.Applicant.UserID, realtime.Event{Type: realtime.EventTypeGroupApplicationResponse, Payload: map[string]any{"notification_id": notificationID, "application_id": application.ApplicationID, "group": map[string]any{"group_id": group.Group.GroupID, "name": group.Group.Name, "avatar_url": group.Group.AvatarURL, "members_count": group.Group.MembersCount}, "responded_at": time.Now().UTC().Format(time.RFC3339)}})
}
