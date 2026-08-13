package groups

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/types"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

func (s *Service) ListMyInvites(ctx context.Context, req MyInvitesRequest) ([]GroupInviteItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListMyInvites(ctx, principal.UserID, req.Status, limit, types.PaginationCursor(req.Pagination))
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
	if inviteExpired(invite) {
		// 惰性过期检测:邀请已超 expires_at,将 pending 置 revoked,推送
		// GroupInviteExpired 事件并落库 group_invite_revoked 通知(revoked_by 省略)。
		if err := s.repo.RevokeInvite(ctx, invite.Group.GroupID, req.InviteID); err != nil {
			return err
		}
		s.notifyGroupInviteExpired(ctx, principal.UserID, invite)
		return ErrInviteExpired
	}
	if err := s.repo.RejectInvite(ctx, req.InviteID); err != nil {
		return err
	}
	s.notifyGroupInviteRejected(ctx, invite, principal.UserID, req.Reason)
	return nil
}

// inviteExpired reports whether the invite's expires_at has passed.
func inviteExpired(invite GroupInviteItem) bool {
	if invite.ExpiresAt == nil || *invite.ExpiresAt == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, *invite.ExpiresAt)
	if err != nil {
		return false
	}
	return time.Now().UTC().After(expiresAt)
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
	// TTL（spec types.md group_invite_ttl_hours）：0 = 永不过期（expires_at 省略）
	var expiresAt *time.Time
	if s.appConfig != nil {
		if ttl := s.appConfig.Get(ctx).Limits.GroupInviteTTLHours; ttl > 0 {
			exp := time.Now().UTC().Add(time.Duration(ttl) * time.Hour)
			expiresAt = &exp
		}
	}
	invitation, err := s.repo.InviteMember(ctx, principal.UserID, req, expiresAt)
	if err != nil {
		if errors.Is(err, ErrAlreadyMember) {
			// 被邀请者已是成员（spec group.md → members/invite no-op 成功，invite_id/expires_at 省略）
			return Invitation{}, nil
		}
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
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListSentInvites(ctx, req.GroupID, req.Status, limit, types.PaginationCursor(req.Pagination))
}

func (s *Service) RevokeInvite(ctx context.Context, req RevokeSentInviteRequest) error {
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
	// Locate the invite to address the notification/event to the invitee.
	var inviteeID string
	sent, _, err := s.repo.ListSentInvites(ctx, req.GroupID, nil, 1000, "")
	if err == nil {
		for _, item := range sent {
			if item.InviteID == req.InviteID && item.Status == ApplicationStatusPending {
				inviteeID = item.Invitee.UserID
				break
			}
		}
	}
	if err := s.repo.RevokeInvite(ctx, req.GroupID, req.InviteID); err != nil {
		return err
	}
	if inviteeID != "" {
		s.notifyGroupInviteRevoked(ctx, group, req.InviteID, inviteeID, req.Reason)
	}
	return nil
}

// notifyGroupInviteRevoked pushes the GroupInviteRevoked event to the
// invitee. 纯实时事件（spec api_spec_websocket.md：不落库通知）。
func (s *Service) notifyGroupInviteRevoked(ctx context.Context, group GroupItem, inviteID string, inviteeUserID string, reason *string) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	revokedBy := s.resolveUserInfo(ctx, principal.UserID)
	now := time.Now().UTC().Format(time.RFC3339)
	payload := map[string]any{
		"invite_id":  inviteID,
		"group_id":   group.Group.GroupID,
		"revoked_by": revokedBy,
		"revoked_at": now,
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	realtime.DefaultHub().BroadcastToUser(inviteeUserID, realtime.Event{Type: realtime.EventTypeGroupInviteRevoked, Payload: payload})
}

// notifyGroupInviteRejected pushes the GroupInviteRejected event to the
// inviter (invite.Inviter). 纯实时事件（spec：不落库通知）。
func (s *Service) notifyGroupInviteRejected(ctx context.Context, invite GroupInviteItem, rejectedByUserID string, reason *string) {
	rejectedBy := s.resolveUserInfo(ctx, rejectedByUserID)
	now := time.Now().UTC().Format(time.RFC3339)
	payload := map[string]any{
		"invite_id":   invite.InviteID,
		"group_id":    invite.Group.GroupID,
		"rejected_by": rejectedBy,
		"rejected_at": now,
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	realtime.DefaultHub().BroadcastToUser(invite.Inviter.UserID, realtime.Event{Type: realtime.EventTypeGroupInviteRejected, Payload: payload})
}

// notifyGroupInviteExpired pushes the GroupInviteExpired event to the
// invitee. 纯实时事件（spec：不落库通知）。
func (s *Service) notifyGroupInviteExpired(ctx context.Context, inviteeUserID string, invite GroupInviteItem) {
	now := time.Now().UTC().Format(time.RFC3339)
	realtime.DefaultHub().BroadcastToUser(inviteeUserID, realtime.Event{
		Type: realtime.EventTypeGroupInviteExpired,
		Payload: map[string]any{
			"invite_id":  invite.InviteID,
			"group_id":   invite.Group.GroupID,
			"expired_at": now,
		},
	})
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
