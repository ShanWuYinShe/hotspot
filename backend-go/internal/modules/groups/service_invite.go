package groups

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/types"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

// defaultPendingStatus 落实 spec「status 缺省 pending；查历史显式传 all」
// （fetch_received / fetch_submitted / fetch_sent 共用；此前直通导致缺省返回全状态，
// 已决邀请重回待办区块或已决邀请混入发件待办并带可点按钮）。
func defaultPendingStatus(status *ApplicationStatus) *ApplicationStatus {
	if status == nil {
		pending := ApplicationStatusPending
		return &pending
	}
	if *status == "all" {
		return nil
	}
	return status
}

func (s *Service) ListMyInvites(ctx context.Context, req MyInvitesRequest) ([]GroupInviteItem, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListMyInvites(ctx, principal.UserID, defaultPendingStatus(req.Status), limit, types.PaginationCursor(req.Pagination))
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
	// 与 RejectInvite 对称的两道检查：仅 pending 邀请可接受；已过期则惰性
	// 置 revoked 并拒绝（此前缺失——被撤销/已拒绝/已过期的邀请仍能入群）。
	if invite.Status != ApplicationStatusPending {
		return ErrInviteExpired
	}
	if inviteExpired(invite) {
		if err := s.repo.RevokeInvite(ctx, invite.Group.GroupID, req.InviteID); err != nil {
			return err
		}
		s.notifyGroupInviteExpired(ctx, principal.UserID, invite)
		return ErrInviteExpired
	}
	// C.3 同款封禁拦截（对齐 JoinGroup service.go:464）：JoinGroup 入口有该
	// 检查而 AcceptInvite 此前没有，被群封禁用户换个入口即可回归。数据层
	// BanMember 已级联撤销 pending 邀请，此处再做一次服务级防御（覆盖封禁
	// 与接受之间的并发窗口）。
	if banned, err := s.repo.IsBanned(ctx, invite.Group.GroupID, principal.UserID); err != nil {
		return err
	} else if banned {
		return ErrBannedInGroup
	}
	if err := s.repo.AcceptInvite(ctx, principal.UserID, req.InviteID); err != nil {
		return err
	}
	// 接受邀请即入群：失效可能存在的「非成员」防穿透缓存值（对齐 JoinGroup/
	// ApproveApplication 两条入群路径——groupmessages/realtime 的发送与订阅
	// 校验走同一键族读穿缓存，漏删会让受邀者在 TTL 内仍被判非成员）。
	s.invalidateMemberCache(ctx, invite.Group.GroupID, principal.UserID)
	s.notifyGroupInviteAccepted(ctx, invite, principal.UserID)
	// member_count 用受邀入群后的权威计数（审查 P2-5，快照+1 在并发下失真）。
	realtime.DefaultHub().BroadcastToGroup(invite.Group.GroupID, realtime.Event{Type: realtime.EventTypeGroupMemberJoined, Payload: GroupMemberJoinedPayload{GroupID: invite.Group.GroupID, User: s.resolveUserInfo(ctx, principal.UserID), Role: string(GroupRoleMember), MemberCount: s.memberCountAfterMutation(ctx, invite.Group.GroupID, invite.Group.MembersCount+1), JoinedAt: time.Now().UTC().Format(time.RFC3339)}})
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
		// 惰性过期检测:邀请已超 expires_at,将 pending 置 revoked,并推送
		// GroupInviteExpired WS 事件（瞬态通知，不落库——契约 NotifyType
		// union 无 group_invite_revoked）。
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
	expiresAt, err := time.Parse(time.RFC3339Nano, *invite.ExpiresAt)
	if err != nil {
		return false
	}
	return time.Now().UTC().After(expiresAt)
}

func (s *Service) InviteMember(ctx context.Context, req InviteMemberRequest) (Invitation, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if apiErr := validateGroupMessage(req.Message); apiErr != nil {
		return Invitation{}, apiErr
	}
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if !ok {
		if err != nil {
			return Invitation{}, err
		}
		return Invitation{}, ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return Invitation{}, ErrAccessRestricted
	}
	// 契约（spec api_spec_group.md → groups/admin/invites/create 错误码）：
	// 邀请已被封禁的用户返回 10400。此前不校验，被封禁用户可经邀请直通回归。
	if banned, err := s.repo.IsBanned(ctx, req.GroupID, req.UserID); err != nil {
		return Invitation{}, err
	} else if banned {
		return Invitation{}, ErrBannedInGroup
	}
	// 被邀请者已是成员则 no-op 成功（spec group.md → 邀请响应）：不建邀请、
	// 不推送通知。内存实现另有同款防御，此处统一服务层口径覆盖双实现。
	if member, err := s.repo.IsMember(ctx, req.GroupID, req.UserID); err != nil {
		return Invitation{}, err
	} else if member {
		return Invitation{}, nil
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
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return nil, "", ErrAccessRestricted
	}
	limit := types.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListSentInvites(ctx, req.GroupID, defaultPendingStatus(req.Status), limit, types.PaginationCursor(req.Pagination))
}

func (s *Service) RevokeInvite(ctx context.Context, req RevokeSentInviteRequest) error {
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
	// Locate the invite to address the notification/event to the invitee.
	// 直查替代 limit=1000 的列表扫描（超限截断会漏检）；invitee 仅在「邀请
	// 属于本群且仍 pending」时用于通知寻址（与旧口径一致）。
	invite, inviteeID, found, err := s.repo.GetInvite(ctx, req.InviteID)
	if err != nil {
		return err
	}
	if !found || invite.Group.GroupID != req.GroupID || invite.Status != ApplicationStatusPending {
		inviteeID = ""
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
	payload := GroupInviteRevokedPayload{
		InviteID:  inviteID,
		GroupID:   group.Group.GroupID,
		RevokedBy: revokedBy,
		RevokedAt: now,
	}
	if reason != nil && *reason != "" {
		payload.Reason = reason
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
		Payload: GroupInviteExpiredPayload{
			InviteID:  invite.InviteID,
			GroupID:   invite.Group.GroupID,
			ExpiredAt: now,
		},
	})
}

func (s *Service) findInviteForUser(ctx context.Context, userID string, inviteID string) (GroupInviteItem, bool, error) {
	// 直查替代 limit=1000 的列表扫描；邀请不属于该用户（invitee 不符）按
	// 不存在处理，不泄漏他人邀请的存在性。invitee 为空仅出现在测试桩回退
	// （生产 PG/内存实现恒返回非空 invitee），此时跳过归属校验。
	item, inviteeUserID, found, err := s.repo.GetInvite(ctx, inviteID)
	if err != nil || !found {
		return GroupInviteItem{}, false, err
	}
	if inviteeUserID != "" && inviteeUserID != userID {
		return GroupInviteItem{}, false, nil
	}
	return item, true, nil
}
