package groups

import (
	"context"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

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
			return nil, "", ErrAccessRestricted
		}
		return nil, "", ErrAccessRestricted
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
	application, group, err := s.findApplicationForApprover(ctx, principal.UserID, req.ApplicationID)
	if err != nil {
		return err
	}
	if err := s.repo.ApproveApplication(ctx, principal.UserID, req.ApplicationID); err != nil {
		return err
	}
	s.notifyGroupApplicationAccepted(ctx, group, application)
	realtime.DefaultHub().BroadcastToGroup(group.Group.GroupID, realtime.Event{Type: "GroupMemberJoined", Payload: map[string]any{"group_id": group.Group.GroupID, "user": groupUserInfo(application.Applicant), "role": string(GroupRoleMember), "member_count": group.Group.MembersCount + 1, "joined_at": time.Now().UTC().Format(time.RFC3339)}})
	return nil
}

func (s *Service) RejectApplication(ctx context.Context, req RejectGroupApplicationRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	_, _, err := s.findApplicationForApprover(ctx, principal.UserID, req.ApplicationID)
	if err != nil {
		return err
	}
	if err := s.repo.RejectApplication(ctx, req.ApplicationID, req.Reason); err != nil {
		return err
	}
	return nil
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
		return ErrAccessRestricted
	}
	if req.Role != GroupRoleAdmin && req.Role != GroupRoleMember {
		return ErrAccessRestricted
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
		return ErrAccessRestricted
	}
	if req.DurationMinutes < 1 || req.DurationMinutes > MaxMuteDurationMinutes {
		return ErrAccessRestricted
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
		return ErrAccessRestricted
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
		return ErrAccessRestricted
	}
	if req.UserID == principal.UserID {
		return ErrAccessRestricted
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
		return ErrAccessRestricted
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
		return nil, "", ErrAccessRestricted
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
		return ErrAccessRestricted
	}
	if req.UserID == principal.UserID {
		return ErrAccessRestricted
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
		return ErrAccessRestricted
	}
	if req.NewOwnerID == principal.UserID {
		return ErrAccessRestricted
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

func (s *Service) findApplicationForApprover(ctx context.Context, approverUserID string, applicationID string) (GroupApplicationItem, GroupItem, error) {
	groups, _, err := s.repo.ListMyGroups(ctx, approverUserID, 1000, "")
	if err != nil {
		return GroupApplicationItem{}, GroupItem{}, err
	}
	for _, group := range groups {
		items, _, err := s.repo.ListApplications(ctx, group.Group.GroupID, nil, 1000, "")
		if err != nil {
			return GroupApplicationItem{}, GroupItem{}, err
		}
		for _, item := range items {
			if item.ApplicationID == applicationID {
				if group.MyState == nil || (group.MyState.Role != GroupRoleOwner && group.MyState.Role != GroupRoleAdmin) {
					return GroupApplicationItem{}, GroupItem{}, ErrAccessRestricted
				}
				if item.Status != ApplicationStatusPending {
					return GroupApplicationItem{}, GroupItem{}, ErrAlreadyProcessed
				}
				return item, group, nil
			}
		}
	}
	return GroupApplicationItem{}, GroupItem{}, ErrGroupNotFound
}
