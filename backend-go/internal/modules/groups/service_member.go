package groups

import (
	"context"
	"hotspot/backend-go/internal/platform/types"
	"strings"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

// MaxMuteDurationMinutes is the maximum duration for muting a group member (30 days)
const MaxMuteDurationMinutes = 43200

func (s *Service) ListApplications(ctx context.Context, req GroupApplicationsRequest) ([]GroupApplicationItem, string, error) {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return nil, "", ErrAccessRestricted
	}
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListApplications(ctx, req.GroupID, req.Status, limit, types.PaginationCursor(req.Pagination))
}

func (s *Service) ApproveApplication(ctx context.Context, req ApproveGroupApplicationRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	application, group, err := s.findApplicationForApprover(ctx, principal.UserID, req.ApplicationID)
	if err != nil {
		return err
	}
	if req.GroupID != application.GroupID {
		// Group-level auth anchor mismatch: report not-found so callers cannot
		// probe applications across groups (spec api_spec_group.md).
		return ErrApplicationNotFound
	}
	if err := s.repo.ApproveApplication(ctx, principal.UserID, req.ApplicationID); err != nil {
		return err
	}
	s.notifyGroupApplicationAccepted(ctx, group, application)
	realtime.DefaultHub().BroadcastToGroup(group.Group.GroupID, realtime.Event{Type: realtime.EventTypeGroupMemberJoined, Payload: map[string]any{"group_id": group.Group.GroupID, "user": s.resolveUserInfo(ctx, application.Applicant.UserID), "role": string(GroupRoleMember), "member_count": group.Group.MembersCount + 1, "joined_at": time.Now().UTC().Format(time.RFC3339)}})
	return nil
}

func (s *Service) RejectApplication(ctx context.Context, req RejectGroupApplicationRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	application, group, err := s.findApplicationForApprover(ctx, principal.UserID, req.ApplicationID)
	if err != nil {
		return err
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
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	if err := s.repo.RejectApplication(ctx, req.ApplicationID, reasonPtr); err != nil {
		return err
	}
	s.notifyGroupApplicationRejected(ctx, group, application, reasonPtr)
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
	if group.ViewerState == nil || group.ViewerState.Role != GroupRoleOwner {
		return ErrAccessRestricted
	}
	if req.Role != GroupRoleAdmin && req.Role != GroupRoleMember {
		return ErrAccessRestricted
	}
	if err := s.repo.SetMemberRole(ctx, principal.UserID, req); err != nil {
		return err
	}
	// 推送（spec group.md → set_role）：向群内广播 GroupMemberRoleChanged
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
		Type: realtime.EventTypeGroupMemberRoleChanged,
		Payload: map[string]any{
			"group_id":   req.GroupID,
			"user":       s.resolveUserInfo(ctx, req.UserID),
			"role":       string(req.Role),
			"changed_by": s.resolveUserInfo(ctx, principal.UserID),
			"changed_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	realtime.DefaultHub().BroadcastToUser(req.UserID, realtime.Event{
		Type: realtime.EventTypeGroupMemberRoleChanged,
		Payload: map[string]any{
			"group_id":   req.GroupID,
			"user":       s.resolveUserInfo(ctx, req.UserID),
			"role":       string(req.Role),
			"changed_by": s.resolveUserInfo(ctx, principal.UserID),
			"changed_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	return nil
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
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	// 目标保护（spec group.md → set_mute）：群主不受禁言
	if isGroupOwner(group.OwnerID, req.UserID) {
		return ErrBusinessConflict
	}
	if req.DurationMinutes < 1 || req.DurationMinutes > MaxMuteDurationMinutes {
		return ErrAccessRestricted
	}
	if err := s.repo.MuteMember(ctx, principal.UserID, req); err != nil {
		return err
	}
	event := realtime.Event{
		Type: realtime.EventTypeGroupMemberMuted,
		Payload: map[string]any{
			"group_id":         req.GroupID,
			"user":             s.resolveUserInfo(ctx, req.UserID),
			"muted_by":         s.resolveUserInfo(ctx, principal.UserID),
			"duration_minutes": req.DurationMinutes,
			"muted_at":         time.Now().UTC().Format(time.RFC3339),
		},
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, event)
	realtime.DefaultHub().BroadcastToUser(req.UserID, event)
	return nil
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
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	if err := s.repo.UnmuteMember(ctx, principal.UserID, req); err != nil {
		return err
	}
	event := realtime.Event{
		Type: realtime.EventTypeGroupMemberUnmuted,
		Payload: map[string]any{
			"group_id":   req.GroupID,
			"user":       s.resolveUserInfo(ctx, req.UserID),
			"unmuted_by": s.resolveUserInfo(ctx, principal.UserID),
			"unmuted_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, event)
	realtime.DefaultHub().BroadcastToUser(req.UserID, event)
	return nil
}

func (s *Service) SetMemberBan(ctx context.Context, req SetMemberBanRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	if req.Active && req.UserID == principal.UserID {
		return ErrAccessRestricted
	}
	// 目标保护（spec group.md → set_ban）：群主不可被封禁
	if req.Active && isGroupOwner(group.OwnerID, req.UserID) {
		return ErrBusinessConflict
	}
	if req.Active {
		if err := s.repo.BanMember(ctx, principal.UserID, BanMemberRequest{
			GroupID: req.GroupID,
			UserID:  req.UserID,
			Reason:  req.Reason,
		}); err != nil {
			return err
		}
		event := realtime.Event{
			Type: realtime.EventTypeGroupMemberRemoved,
			Payload: map[string]any{
				"group_id":     req.GroupID,
				"user":         s.resolveUserInfo(ctx, req.UserID),
				"removed_by":   s.resolveUserInfo(ctx, principal.UserID),
				"reason":       "banned",
				"removed_at":   time.Now().UTC().Format(time.RFC3339),
				"member_count": group.Group.MembersCount,
			},
		}
		realtime.DefaultHub().BroadcastToGroup(req.GroupID, event)
		realtime.DefaultHub().BroadcastToUser(req.UserID, event)
		return nil
	}
	return s.repo.UnbanMember(ctx, principal.UserID, UnbanMemberRequest{
		GroupID: req.GroupID,
		UserID:  req.UserID,
	})
}

func (s *Service) ListBannedMembers(ctx context.Context, req BannedMembersRequest) ([]AdminBannedMemberItem, string, error) {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return nil, "", err
		}
		return nil, "", ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return nil, "", ErrAccessRestricted
	}
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListBannedMembers(ctx, req.GroupID, limit, types.PaginationCursor(req.Pagination))
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
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	if req.UserID == principal.UserID {
		return ErrAccessRestricted
	}
	// 目标保护（spec group.md → members/remove）：群主不可被移除
	if isGroupOwner(group.OwnerID, req.UserID) {
		return ErrBusinessConflict
	}
	if err := s.repo.RemoveMember(ctx, principal.UserID, req); err != nil {
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
		Type: realtime.EventTypeGroupMemberRemoved,
		Payload: map[string]any{
			"group_id":     req.GroupID,
			"user":         s.resolveUserInfo(ctx, req.UserID),
			"removed_by":   s.resolveUserInfo(ctx, principal.UserID),
			"reason":       "kicked",
			"removed_at":   time.Now().UTC().Format(time.RFC3339),
			"member_count": group.Group.MembersCount,
		},
	})
	realtime.DefaultHub().BroadcastToUser(req.UserID, realtime.Event{
		Type: realtime.EventTypeGroupMemberRemoved,
		Payload: map[string]any{
			"group_id":     req.GroupID,
			"reason":       "kicked",
			"removed_at":   time.Now().UTC().Format(time.RFC3339),
			"member_count": group.Group.MembersCount,
		},
	})
	return nil
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
	if group.ViewerState == nil || group.ViewerState.Role != GroupRoleOwner {
		return ErrAccessRestricted
	}
	if req.NewOwnerID == principal.UserID {
		return ErrAccessRestricted
	}

	// Direct membership probe instead of paging up to 1000 members and linear
	// scanning for the target user.
	isMember, err := s.repo.IsMember(ctx, req.GroupID, req.NewOwnerID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrTargetNotMember
	}

	if err := s.repo.TransferOwnership(ctx, principal.UserID, req); err != nil {
		return err
	}
	// 推送（spec group.md → transfer_ownership）：owner 变更即角色变更，向群内广播
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
		Type: realtime.EventTypeGroupMemberRoleChanged,
		Payload: map[string]any{
			"group_id":   req.GroupID,
			"user":       s.resolveUserInfo(ctx, req.NewOwnerID),
			"role":       string(GroupRoleOwner),
			"changed_by": s.resolveUserInfo(ctx, principal.UserID),
			"changed_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
	return nil
}

func (s *Service) findApplicationForApprover(ctx context.Context, approverUserID string, applicationID string) (GroupApplicationItem, GroupItem, error) {
	// Direct application lookup (single indexed query) instead of the previous
	// O(approver's groups × each group's applications) nested scan that could
	// touch up to 10^6 rows to locate one record.
	item, found, err := s.repo.GetApplication(ctx, applicationID)
	if err != nil {
		return GroupApplicationItem{}, GroupItem{}, err
	}
	if !found {
		return GroupApplicationItem{}, GroupItem{}, ErrGroupNotFound
	}
	// Get loads the approver's ViewerState (role) from the auth context, so we can
	// authorize without paging member lists.
	group, groupFound, err := s.repo.Get(ctx, item.GroupID)
	if err != nil {
		return GroupApplicationItem{}, GroupItem{}, err
	}
	if !groupFound {
		return GroupApplicationItem{}, GroupItem{}, ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return GroupApplicationItem{}, GroupItem{}, ErrAccessRestricted
	}
	if item.Status != ApplicationStatusPending {
		return GroupApplicationItem{}, GroupItem{}, ErrAlreadyProcessed
	}
	if s.applicationExpired(ctx, item) {
		// 惰性过期检测:申请已超过 TTL,将 pending 置 revoked,推送
		// GroupApplicationExpired 事件并落库 group_application_cancelled 通知。
		if err := s.repo.CancelApplication(ctx, applicationID); err != nil {
			return GroupApplicationItem{}, GroupItem{}, err
		}
		s.notifyGroupApplicationExpired(ctx, group, item)
		return GroupApplicationItem{}, GroupItem{}, ErrAlreadyProcessed
	}
	return item, group, nil
}
