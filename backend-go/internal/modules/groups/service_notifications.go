package groups

import (
	"context"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
	"time"
)

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
