package groups

import (
	"context"
	"sort"
	"strings"
	"time"
)

func (r *InMemoryRepository) ListMembers(_ context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group, ok := r.groups[groupID]
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	items := make([]GroupMemberItem, 0)
	for userID, membership := range r.members[groupID] {
		if role != nil && membership.Role != *role {
			continue
		}
		items = append(items, GroupMemberItem{User: UserInfo{UserID: userID, Nickname: userID}, Role: membership.Role, JoinedAt: membership.JoinedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].JoinedAt == items[j].JoinedAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].JoinedAt < items[j].JoinedAt
	})
	for i := range items {
		if items[i].User.UserID == group.Group.GroupID+"_owner" {
			items[i].User.Nickname = group.Group.Name + " Owner"
		}
	}
	// 消费游标：跳过 (JoinedAt, UserID) <= cursor 的项
	if cursor != "" {
		if parts := strings.SplitN(cursor, ":", 2); len(parts) == 2 {
			cursorAt, cursorUID := parts[0], parts[1]
			start := 0
			for i, item := range items {
				if item.JoinedAt > cursorAt || (item.JoinedAt == cursorAt && item.User.UserID > cursorUID) {
					break
				}
				start = i + 1
			}
			items = items[start:]
		}
	}
	if len(items) > limit {
		return items[:limit], items[limit-1].JoinedAt + ":" + items[limit-1].User.UserID, nil
	}
	return items, "", nil
}

// IsMember is a direct membership probe used on hot paths (realtime authz),
// replacing the prior pattern of paging the whole member list.
func (r *InMemoryRepository) IsMember(_ context.Context, groupID, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if members, ok := r.members[groupID]; ok {
		_, isMember := members[userID]
		return isMember, nil
	}
	return false, nil
}

// GetApplication loads a single application by ID (O(1) map lookup) for paths
// that previously scanned all of an approver's groups and each group's
// applications to find one record.
func (r *InMemoryRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	items, _, err := r.ListMembers(ctx, groupID, role, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	filtered := make([]GroupMemberItem, 0, len(items))
	for _, item := range items {
		if keyword == "" || item.User.Nickname == keyword || item.User.UserID == keyword {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) > limit {
		return filtered[:limit], filtered[limit-1].JoinedAt + ":" + filtered[limit-1].User.UserID, nil
	}
	return filtered, "", nil
}

func (r *InMemoryRepository) SetMemberRole(_ context.Context, userID string, req SetMemberRoleRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.UserID]
	if !ok {
		return ErrNotMember
	}
	membership.Role = req.Role
	members[req.UserID] = membership
	_ = userID
	return nil
}

func (r *InMemoryRepository) MuteMember(_ context.Context, userID string, req MuteMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.UserID]
	if !ok {
		return ErrNotMember
	}
	membership.IsMuted = true
	mutedUntil := time.Now().UTC().Add(time.Duration(req.DurationMinutes) * time.Minute)
	membership.MutedUntil = mutedUntil.Format(time.RFC3339)
	members[req.UserID] = membership
	_ = userID
	return nil
}

func (r *InMemoryRepository) UnmuteMember(_ context.Context, userID string, req UnmuteMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.UserID]
	if !ok {
		return ErrNotMember
	}
	membership.IsMuted = false
	membership.MutedUntil = ""
	members[req.UserID] = membership
	_ = userID
	return nil
}

func (r *InMemoryRepository) BanMember(_ context.Context, userID string, req BanMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	if r.banned[req.GroupID] == nil {
		r.banned[req.GroupID] = make(map[string]inMemoryBanRecord)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.banned[req.GroupID][req.UserID] = inMemoryBanRecord{BannedAt: now, Reason: req.Reason, BannedBy: userID}
	if members, ok := r.members[req.GroupID]; ok {
		delete(members, req.UserID)
		group := r.groups[req.GroupID]
		group.Group.MembersCount = len(members)
		r.groups[req.GroupID] = group
	}
	return nil
}

func (r *InMemoryRepository) UnbanMember(_ context.Context, userID string, req UnbanMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	if banned, ok := r.banned[req.GroupID]; ok {
		delete(banned, req.UserID)
	}
	_ = userID
	return nil
}

func (r *InMemoryRepository) ListBannedMembers(_ context.Context, groupID string, limit int, cursor string) ([]BannedMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	_ = cursor
	items := make([]BannedMemberItem, 0, len(r.banned[groupID]))
	for userID, record := range r.banned[groupID] {
		item := BannedMemberItem{
			User:     UserInfo{UserID: userID, Nickname: userID},
			BannedAt: record.BannedAt,
			Reason:   record.Reason,
		}
		if record.BannedBy != "" {
			item.BannedBy = &UserInfo{UserID: record.BannedBy, Nickname: record.BannedBy}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].BannedAt == items[j].BannedAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].BannedAt > items[j].BannedAt
	})
	if limit <= 0 {
		limit = 20
	}
	// 消费游标：跳过 (BannedAt, UserID) >= cursor 的项（DESC 序）
	if cursor != "" {
		if parts := strings.SplitN(cursor, ":", 2); len(parts) == 2 {
			cursorAt, cursorUID := parts[0], parts[1]
			start := 0
			for i, item := range items {
				if item.BannedAt < cursorAt || (item.BannedAt == cursorAt && item.User.UserID > cursorUID) {
					break
				}
				start = i + 1
			}
			items = items[start:]
		}
	}
	if len(items) > limit {
		last := items[limit-1]
		return items[:limit], last.BannedAt + ":" + last.User.UserID, nil
	}
	return items, "", nil
}

func (r *InMemoryRepository) RemoveMember(_ context.Context, userID string, req RemoveMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return nil
	}
	if _, ok := members[req.UserID]; !ok {
		return nil
	}
	delete(members, req.UserID)
	group := r.groups[req.GroupID]
	group.Group.MembersCount = len(members)
	r.groups[req.GroupID] = group
	_ = userID
	return nil
}

func (r *InMemoryRepository) TransferOwnership(_ context.Context, userID string, req TransferOwnershipRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[req.GroupID]
	if !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.NewOwnerID]
	if !ok {
		return ErrNotMember
	}
	if oldMembership, ok := members[userID]; ok {
		oldMembership.Role = GroupRoleAdmin
		members[userID] = oldMembership
	}
	membership.Role = GroupRoleOwner
	members[req.NewOwnerID] = membership
	group.OwnerID = req.NewOwnerID
	r.groups[req.GroupID] = group
	return nil
}

func (r *InMemoryRepository) cloneGroupForUser(group GroupItem, userID string) GroupItem {
	cloned := group
	if memberships, ok := r.members[group.Group.GroupID]; ok {
		cloned.Group.MembersCount = len(memberships)
		if membership, joined := memberships[userID]; joined {
			state := &MyGroupState{Role: membership.Role, IsJoined: true, UnreadCount: 0, IsMuted: membership.IsMuted, MutedUntil: membership.MutedUntil, IsPinned: membership.IsPinned, JoinedAt: membership.JoinedAt}
			cloned.MyState = state
		} else {
			cloned.MyState = nil
		}
	}
	return cloned
}
