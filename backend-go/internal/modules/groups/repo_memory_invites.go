package groups

import (
	"context"

	"sort"

	"time"
)

// GetInvite 按 invite_id 直查邀请（repo.go 接口注释）。
func (r *InMemoryRepository) GetInvite(_ context.Context, inviteID string) (GroupInviteItem, string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.invites[inviteID]
	if !ok {
		return GroupInviteItem{}, "", false, nil
	}
	group := r.groups[record.GroupID]
	item := GroupInviteItem{InviteID: record.InviteID, Group: group.Group, Inviter: UserInfo{UserID: record.InviterID, Nickname: record.InviterID}, Message: record.Message, Status: record.Status, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt, RespondedAt: record.RespondedAt}
	return item, record.InviteeID, true, nil
}

func (r *InMemoryRepository) ListMyInvites(_ context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]GroupInviteItem, 0)
	for _, record := range r.invites {
		if record.InviteeID != userID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		group := r.groups[record.GroupID]
		items = append(items, GroupInviteItem{InviteID: record.InviteID, Group: group.Group, Inviter: UserInfo{UserID: record.InviterID, Nickname: record.InviterID}, Message: record.Message, Status: record.Status, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].InviteID > items[j].InviteID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, "", nil
}

func (r *InMemoryRepository) AcceptInvite(_ context.Context, userID string, inviteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.invites[inviteID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrInviteExpired
	}
	record.Status = ApplicationStatusAccepted
	now := time.Now().UTC().Format(time.RFC3339)
	record.RespondedAt = &now
	r.invites[inviteID] = record
	if group, ok := r.groups[record.GroupID]; ok && group.MaxMembers > 0 && len(r.members[record.GroupID]) >= group.MaxMembers {
		return ErrGroupFull
	}
	if r.members[record.GroupID] == nil {
		r.members[record.GroupID] = make(map[string]inMemoryMembership)
	}
	r.members[record.GroupID][userID] = inMemoryMembership{Role: GroupRoleMember, JoinedAt: now}
	if group, ok := r.groups[record.GroupID]; ok {
		group.Group.MembersCount = len(r.members[record.GroupID])
		r.groups[record.GroupID] = group
	}
	return nil
}

func (r *InMemoryRepository) RejectInvite(_ context.Context, inviteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.invites[inviteID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrInviteExpired
	}
	record.Status = ApplicationStatusRejected
	now := time.Now().UTC().Format(time.RFC3339)
	record.RespondedAt = &now
	r.invites[inviteID] = record
	return nil
}

func (r *InMemoryRepository) InviteMember(_ context.Context, userID string, req InviteMemberRequest, expiresAt *time.Time) (Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[req.GroupID]
	if !ok {
		return Invitation{}, ErrGroupNotFound
	}
	if _, member := r.members[req.GroupID][req.UserID]; member {
		return Invitation{}, ErrAlreadyMember
	}
	for _, record := range r.invites {
		if record.GroupID == req.GroupID && record.InviteeID == req.UserID && record.Status == ApplicationStatusPending {
			return Invitation{InviteID: &record.InviteID, ExpiresAt: record.ExpiresAt}, nil
		}
	}
	now := time.Now().UTC()
	var expiresAtStr *string
	if expiresAt != nil {
		v := expiresAt.UTC().Format(time.RFC3339)
		expiresAtStr = &v
	}
	inviteID := "invite_" + now.Format("20060102150405.000000000")
	record := inMemoryInviteRecord{InviteID: inviteID, GroupID: req.GroupID, InviterID: userID, InviteeID: req.UserID, Status: ApplicationStatusPending, Message: req.Message, CreatedAt: now.Format(time.RFC3339), ExpiresAt: expiresAtStr}
	r.invites[inviteID] = record
	group.Group.MembersCount = len(r.members[req.GroupID])
	r.groups[req.GroupID] = group
	return Invitation{InviteID: &inviteID, ExpiresAt: expiresAtStr}, nil
}

func (r *InMemoryRepository) ListSentInvites(_ context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	items := make([]GroupSentInviteItem, 0)
	for _, record := range r.invites {
		if record.GroupID != groupID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		items = append(items, GroupSentInviteItem{InviteID: record.InviteID, Inviter: UserInfo{UserID: record.InviterID, Nickname: record.InviterID}, Invitee: UserInfo{UserID: record.InviteeID, Nickname: record.InviteeID}, Status: record.Status, Message: record.Message, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt, RespondedAt: record.RespondedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].InviteID > items[j].InviteID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, "", nil
}

func (r *InMemoryRepository) RevokeInvite(_ context.Context, groupID string, inviteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	record, ok := r.invites[inviteID]
	if !ok || record.GroupID != groupID {
		// 与 PG 版（queryRevokeGroupInvite 归属双键 + pending 守卫）对齐：
		// 非本群邀请或已流转的邀请不可撤销。
		return ErrInviteExpired
	}
	if record.Status != ApplicationStatusPending {
		return ErrInviteExpired
	}
	record.Status = ApplicationStatusRevoked
	now := time.Now().UTC().Format(time.RFC3339)
	record.RespondedAt = &now
	r.invites[inviteID] = record
	return nil
}
