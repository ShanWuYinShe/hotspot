package groups

import (
	"context"

	"sort"

	"time"
)

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

func (r *InMemoryRepository) InviteMember(_ context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
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
			return Invitation{InviteID: record.InviteID, ExpiresAt: derefString(record.ExpiresAt)}, nil
		}
	}
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour).Format(time.RFC3339)
	inviteID := "invite_" + now.Format("20060102150405.000000000")
	record := inMemoryInviteRecord{InviteID: inviteID, GroupID: req.GroupID, InviterID: userID, InviteeID: req.UserID, Status: ApplicationStatusPending, Message: req.Message, CreatedAt: now.Format(time.RFC3339), ExpiresAt: &expiresAt}
	r.invites[inviteID] = record
	group.Group.MembersCount = len(r.members[req.GroupID])
	r.groups[req.GroupID] = group
	return Invitation{InviteID: inviteID, ExpiresAt: expiresAt}, nil
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
		items = append(items, GroupSentInviteItem{InviteID: record.InviteID, Invitee: UserInfo{UserID: record.InviteeID, Nickname: record.InviteeID}, Status: record.Status, Message: record.Message, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt, RespondedAt: record.RespondedAt})
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
	if !ok {
		return nil
	}
	record.Status = ApplicationStatusRevoked
	now := time.Now().UTC().Format(time.RFC3339)
	record.RespondedAt = &now
	r.invites[inviteID] = record
	return nil
}
