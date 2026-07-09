package groups

import (
	"context"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

func (s *Service) ListMyInvites(ctx context.Context, req MyInvitesRequest) ([]GroupInviteItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := req.Pagination.limit(20, 100)
	return s.repo.ListMyInvites(ctx, principal.UserID, req.Status, limit, req.Pagination.cursor())
}

func (s *Service) AcceptInvite(ctx context.Context, req AcceptInviteRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	invite, ok, err := s.findInviteForUser(ctx, principal.UserID, req.InviteID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if err := s.repo.AcceptInvite(ctx, principal.UserID, req.InviteID); err != nil {
		return err
	}
	s.notifyGroupInviteAccepted(ctx, invite, principal.UserID)
	realtime.DefaultHub().BroadcastToGroup(invite.Group.GroupID, realtime.Event{Type: realtime.EventTypeGroupMemberJoined, Payload: map[string]any{"group_id": invite.Group.GroupID, "user": s.resolveUserInfo(ctx, principal.UserID), "role": string(GroupRoleMember), "member_count": invite.Group.MembersCount + 1, "joined_at": time.Now().UTC().Format(time.RFC3339)}})
	return nil
}

func (s *Service) RejectInvite(ctx context.Context, req RejectInviteRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	invite, ok, err := s.findInviteForUser(ctx, principal.UserID, req.InviteID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if invite.Status != ApplicationStatusPending {
		return ErrInviteExpired
	}
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
		return Invitation{}, ErrAccessRestricted
	}
	invitation, err := s.repo.InviteMember(ctx, principal.UserID, req)
	if err != nil {
		return Invitation{}, err
	}
	s.notifyGroupInviteReceived(ctx, group, principal.UserID, req, invitation)
	return invitation, nil
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
		return nil, "", ErrAccessRestricted
	}
	limit := req.Pagination.limit(20, 100)
	return s.repo.ListSentInvites(ctx, req.GroupID, req.Status, limit, req.Pagination.cursor())
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
		return ErrAccessRestricted
	}
	_ = principal
	return s.repo.RevokeInvite(ctx, req.GroupID, req.InviteID)
}

func (s *Service) findInviteForUser(ctx context.Context, userID string, inviteID string) (GroupInviteItem, bool, error) {
	items, _, err := s.repo.ListMyInvites(ctx, userID, nil, 1000, "")
	if err != nil {
		return GroupInviteItem{}, false, err
	}
	for _, item := range items {
		if item.InviteID == inviteID {
			return item, true, nil
		}
	}
	return GroupInviteItem{}, false, nil
}
