package groups

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
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
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Create(ctx, principal.UserID, req)
}

func (s *Service) OpenTheDoor(ctx context.Context, groupID string) (GroupItem, bool, error) {
	return s.repo.Get(ctx, groupID)
}

func (s *Service) Search(ctx context.Context, keyword string) (PaginatedGroups, error) {
	return s.repo.Search(ctx, keyword)
}

func (s *Service) Update(ctx context.Context, req UpdateGroupRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		return ErrPermissionDenied
	}
	return s.repo.Update(ctx, principal.UserID, req)
}

func (s *Service) Dismiss(ctx context.Context, req DismissGroupRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		return ErrPermissionDenied
	}
	return s.repo.Dismiss(ctx, principal.UserID)
}

func (s *Service) ListMembers(ctx context.Context, req GroupMembersRequest) ([]GroupMemberItem, string, error) {
	_, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListMembers(ctx, req.GroupID, req.Role, limit, req.Cursor)
}

func (s *Service) SearchMembers(ctx context.Context, req SearchGroupMembersRequest) ([]GroupMemberItem, string, error) {
	_, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.SearchMembers(ctx, req.GroupID, req.Keyword, req.Role, limit, req.Cursor)
}

func (s *Service) ListMyGroups(ctx context.Context, req MyGroupsRequest) ([]GroupItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListMyGroups(ctx, principal.UserID, limit, req.Cursor)
}

func (s *Service) JoinGroup(ctx context.Context, req JoinGroupRequest) (JoinGroupResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return JoinGroupResult{}, err
	}
	if !ok {
		return JoinGroupResult{}, ErrGroupNotFound
	}
	if group.MyState != nil && group.MyState.IsJoined {
		return JoinGroupResult{Status: ApplicationStatusAccepted}, nil
	}
	return s.repo.JoinGroup(ctx, principal.UserID, req.GroupID, req.Password, req.Message)
}

func (s *Service) CreateAnnouncement(ctx context.Context, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	if !ok {
		return GroupAnnouncementItem{}, ErrGroupNotFound
	}
	if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
		return GroupAnnouncementItem{}, ErrPermissionDenied
	}
	return s.repo.CreateAnnouncement(ctx, principal.UserID, req)
}

func (s *Service) ListAnnouncements(ctx context.Context, req GroupAnnouncementsRequest) ([]GroupAnnouncementItem, error) {
	return s.repo.ListAnnouncements(ctx, req.GroupID)
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
	principal, _ := platformauth.PrincipalFromContext(ctx)
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
