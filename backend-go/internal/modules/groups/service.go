package groups

import (
	"context"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
)

const (
	// MaxMuteDurationMinutes is the maximum duration for muting a group member (30 days)
	MaxMuteDurationMinutes = 43200
)

type Service struct {
	repo Repository
}

func NewService() *Service {
	return NewServiceWithRepo(NewInMemoryRepository())
}

func NewServiceWithRepo(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateGroupRequest) (GroupItem, error) {
	startedAt := time.Now()
	observability.L().Info("groups.create started")
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("groups.create failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupItem{}, err
	}
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
	observability.L().Info("groups.open_the_door completed", "group_id", groupID, "found", ok, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return item, ok, nil
}

func (s *Service) Search(ctx context.Context, keyword string) (PaginatedGroups, error) {
	startedAt := time.Now()
	observability.L().Info("groups.search started", "keyword", keyword)
	result, err := s.repo.Search(ctx, keyword)
	if err != nil {
		observability.L().Error("groups.search failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return PaginatedGroups{}, err
	}
	observability.L().Info("groups.search completed", "elapsed_ms", time.Since(startedAt).Milliseconds())
	return result, nil
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
		return ErrPermissionDenied
	}
	if err := s.repo.Update(ctx, principal.UserID, req); err != nil {
		observability.L().Error("groups.update failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
	observability.L().Info("groups.update completed", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return nil
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
		return ErrPermissionDenied
	}
	if err := s.repo.Dismiss(ctx, principal.UserID); err != nil {
		observability.L().Error("groups.dismiss failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return err
	}
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
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	items, cursor, err := s.repo.ListMembers(ctx, req.GroupID, req.Role, limit, req.Cursor)
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
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	items, cursor, err := s.repo.SearchMembers(ctx, req.GroupID, req.Keyword, req.Role, limit, req.Cursor)
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
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	items, cursor, err := s.repo.ListMyGroups(ctx, principal.UserID, limit, req.Cursor)
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
		return JoinGroupResult{Status: ApplicationStatusAccepted}, nil
	}
	result, err := s.repo.JoinGroup(ctx, principal.UserID, req.GroupID, req.Password, req.Message)
	if err != nil {
		observability.L().Error("groups.join_group failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return JoinGroupResult{}, err
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
		return GroupAnnouncementItem{}, ErrPermissionDenied
	}
	item, err := s.repo.CreateAnnouncement(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("groups.create_announcement failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, err
	}
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
		return ErrPermissionDenied
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
		return ErrPermissionDenied
	}
	return s.repo.DeleteAnnouncement(ctx, req.GroupID, req.AnnouncementID)
}

func (s *Service) ListApplications(ctx context.Context, req GroupApplicationsRequest) ([]GroupApplicationItem, string, error) {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		if principal.UserID == "" {
			return nil, "", ErrPermissionDenied
		}
	}
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListApplications(ctx, req.GroupID, req.Status, limit, req.Cursor)
}

func (s *Service) ApproveApplication(ctx context.Context, req ApproveGroupApplicationRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.ApproveApplication(ctx, principal.UserID, req.ApplicationID)
}

func (s *Service) RejectApplication(ctx context.Context, req RejectGroupApplicationRequest) error {
	return s.repo.RejectApplication(ctx, req.ApplicationID, req.Reason)
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
		return ErrPermissionDenied
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
		return ErrPermissionDenied
	}
	if group.MyState.Role == GroupRoleOwner {
		return ErrPermissionDenied
	}
	return s.repo.LeaveGroup(ctx, principal.UserID, req.GroupID)
}

func (s *Service) ListMyApplications(ctx context.Context, req MyApplicationsRequest) ([]GroupApplicationItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListMyApplications(ctx, principal.UserID, req.Status, limit, req.Cursor)
}

func (s *Service) CancelApplication(ctx context.Context, req CancelApplicationRequest) error {
	return s.repo.CancelApplication(ctx, req.ApplicationID)
}

func (s *Service) ListMyInvites(ctx context.Context, req MyInvitesRequest) ([]GroupInviteItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListMyInvites(ctx, principal.UserID, req.Status, limit, req.Cursor)
}

func (s *Service) AcceptInvite(ctx context.Context, req AcceptInviteRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.AcceptInvite(ctx, principal.UserID, req.InviteID)
}

func (s *Service) RejectInvite(ctx context.Context, req RejectInviteRequest) error {
	return s.repo.RejectInvite(ctx, req.InviteID)
}

func (s *Service) InviteMember(ctx context.Context, req InviteMemberRequest) (Invitation, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return Invitation{}, err
		}
		return Invitation{}, ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return Invitation{}, ErrPermissionDenied
	}
	return s.repo.InviteMember(ctx, principal.UserID, req)
}

func (s *Service) ListSentInvites(ctx context.Context, req GroupSentInvitesRequest) ([]GroupSentInviteItem, string, error) {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return nil, "", err
		}
		return nil, "", ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return nil, "", ErrPermissionDenied
	}
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListSentInvites(ctx, req.GroupID, req.Status, limit, req.Cursor)
}

func (s *Service) RevokeInvite(ctx context.Context, req RevokeSentInviteRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrPermissionDenied
	}
	_ = principal
	return s.repo.RevokeInvite(ctx, req.GroupID, req.InviteID)
}

func (s *Service) SetMemberRole(ctx context.Context, req SetMemberRoleRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		return ErrPermissionDenied
	}
	if req.Role != GroupRoleAdmin && req.Role != GroupRoleMember {
		return ErrPermissionDenied
	}
	_ = principal
	return s.repo.SetMemberRole(ctx, principal.UserID, req)
}

func (s *Service) MuteMember(ctx context.Context, req MuteMemberRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrPermissionDenied
	}
	if req.DurationMinutes < 1 || req.DurationMinutes > MaxMuteDurationMinutes {
		return ErrPermissionDenied
	}
	_ = principal
	return s.repo.MuteMember(ctx, principal.UserID, req)
}

func (s *Service) UnmuteMember(ctx context.Context, req UnmuteMemberRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrPermissionDenied
	}
	_ = principal
	return s.repo.UnmuteMember(ctx, principal.UserID, req)
}

func (s *Service) BanMember(ctx context.Context, req BanMemberRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrPermissionDenied
	}
	if req.UserID == principal.UserID {
		return ErrPermissionDenied
	}
	_ = principal
	return s.repo.BanMember(ctx, principal.UserID, req)
}

func (s *Service) UnbanMember(ctx context.Context, req UnbanMemberRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrPermissionDenied
	}
	_ = principal
	return s.repo.UnbanMember(ctx, principal.UserID, req)
}

func (s *Service) ListBannedMembers(ctx context.Context, req BannedMembersRequest) ([]BannedMemberItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return nil, "", err
		}
		return nil, "", ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return nil, "", ErrPermissionDenied
	}
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	_ = principal
	return s.repo.ListBannedMembers(ctx, req.GroupID, limit, req.Cursor)
}

func (s *Service) RemoveMember(ctx context.Context, req RemoveMemberRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return ErrPermissionDenied
	}
	if req.UserID == principal.UserID {
		return ErrPermissionDenied
	}
	_ = principal
	return s.repo.RemoveMember(ctx, principal.UserID, req)
}

func (s *Service) TransferOwnership(ctx context.Context, req TransferOwnershipRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		return ErrPermissionDenied
	}
	if req.NewOwnerID == principal.UserID {
		return ErrPermissionDenied
	}

	members, _, err := s.repo.ListMembers(ctx, req.GroupID, nil, 1000, "")
	if err != nil {
		return err
	}

	isMember := false
	for _, m := range members {
		if m.User.UserID == req.NewOwnerID {
			isMember = true
			break
		}
	}
	if !isMember {
		return ErrTargetNotMember
	}

	_ = principal
	return s.repo.TransferOwnership(ctx, principal.UserID, req)
}
