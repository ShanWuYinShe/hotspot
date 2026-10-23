package groups

import (
	"context"
	"hotspot/backend-go/internal/platform/types"
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
	// 消费游标：跳过 (JoinedAt, UserID) <= cursor 的项。游标与 PG 共用
	// groupPageCursorCodec（审查 P2-8-5：此前明文 joinedAt:userID 与 PG 的
	// AES-GCM 游标互不兼容，且 RFC3339 时间串自带冒号无法可靠拆分）。
	if cursor != "" {
		cursorAt, cursorUID, err := decodeGroupPageCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		start := 0
		for i, item := range items {
			itemAt, err := time.Parse(time.RFC3339Nano, item.JoinedAt)
			if err != nil {
				return nil, "", err
			}
			if itemAt.After(cursorAt) || (itemAt.Equal(cursorAt) && item.User.UserID > cursorUID) {
				break
			}
			start = i + 1
		}
		items = items[start:]
	}
	if len(items) > limit {
		lastAt, err := time.Parse(time.RFC3339Nano, items[limit-1].JoinedAt)
		if err != nil {
			return nil, "", err
		}
		return items[:limit], encodeGroupPageCursor(lastAt, items[limit-1].User.UserID), nil
	}
	return items, "", nil
}

// ListAdminMembers returns the admin-perspective member list (includes the
// truthy muted_until; null = not muted). groups/admin/members/list only.
func (r *InMemoryRepository) ListAdminMembers(_ context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]AdminGroupMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group, ok := r.groups[groupID]
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	items := make([]AdminGroupMemberItem, 0)
	for userID, membership := range r.members[groupID] {
		if role != nil && membership.Role != *role {
			continue
		}
		items = append(items, AdminGroupMemberItem{
			User:       UserInfo{UserID: userID, Nickname: userID},
			Role:       membership.Role,
			JoinedAt:   membership.JoinedAt,
			MutedUntil: groupMutedUntil(membership.MutedUntil),
		})
	}
	_ = group
	sort.Slice(items, func(i, j int) bool {
		if items[i].JoinedAt == items[j].JoinedAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].JoinedAt < items[j].JoinedAt
	})
	limit = types.NormalizeLimit(limit)
	// 管理端成员列表此前完全忽略游标（审查 P2-8-5），补 keyset 消费与
	// 共享 codec（与 ListMembers 同序同口径）。
	if cursor != "" {
		cursorAt, cursorUID, err := decodeGroupPageCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		start := 0
		for i, item := range items {
			itemAt, err := time.Parse(time.RFC3339Nano, item.JoinedAt)
			if err != nil {
				return nil, "", err
			}
			if itemAt.After(cursorAt) || (itemAt.Equal(cursorAt) && item.User.UserID > cursorUID) {
				break
			}
			start = i + 1
		}
		items = items[start:]
	}
	if len(items) > limit {
		lastAt, err := time.Parse(time.RFC3339Nano, items[limit-1].JoinedAt)
		if err != nil {
			return nil, "", err
		}
		return items[:limit], encodeGroupPageCursor(lastAt, items[limit-1].User.UserID), nil
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

// AreMembers 批量判定成员资格（内存实现为一次锁下逐 key 查 map，无 IO）。
func (r *InMemoryRepository) AreMembers(_ context.Context, groupIDs []string, userID string) (map[string]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]bool, len(groupIDs))
	for _, groupID := range groupIDs {
		if _, ok := result[groupID]; ok {
			continue
		}
		if members, ok := r.members[groupID]; ok {
			_, result[groupID] = members[userID]
		}
	}
	return result, nil
}

// IsBanned reports whether the user is banned from the group (join-time check, C.3).
func (r *InMemoryRepository) IsBanned(_ context.Context, groupID, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if banned, ok := r.banned[groupID]; ok {
		_, isBanned := banned[userID]
		return isBanned, nil
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
	// 与 PG 版 ILIKE 子串匹配同口径（审查 P2-8-7）：此前内存版仅精确匹配
	// （nickname==keyword），单测在内存实现下断言「按关键词搜索」会掩盖生产
	// 的子串匹配行为。PG 侧 ILIKE 默认大小写不敏感，这里同步 lower 比较。
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	filtered := make([]GroupMemberItem, 0, len(items))
	for _, item := range items {
		if keyword == "" ||
			strings.Contains(strings.ToLower(item.User.Nickname), keyword) ||
			strings.Contains(strings.ToLower(item.User.UserID), keyword) {
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
	// 与 Postgres 实现的 role <> 'owner' 守卫同语义（审计 F-05 防线二）：
	// owner 行不允许经 SetMemberRole 变更，转让群主另有专用路径。
	if membership.Role == GroupRoleOwner {
		return ErrBusinessConflict
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
	// 封禁级联（契约级，spec api_spec_group.md → set_ban）：被 ban 用户发出的
	// 该群 pending 申请、发给被 ban 用户的该群 pending 邀请，均置 revoked。
	// 与 PG 实现同口径，防止封禁被「邀请直通 / 审批通过」两条旁路绕过。
	revoked := ApplicationStatusRevoked
	for id, application := range r.applications {
		if application.GroupID == req.GroupID && application.ApplicantID == req.UserID && application.Status == ApplicationStatusPending {
			application.Status = revoked
			application.HandledAt = &now
			r.applications[id] = application
		}
	}
	for id, invite := range r.invites {
		if invite.GroupID == req.GroupID && invite.InviteeID == req.UserID && invite.Status == ApplicationStatusPending {
			invite.Status = revoked
			invite.RespondedAt = &now
			r.invites[id] = invite
		}
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

func (r *InMemoryRepository) ListBannedMembers(_ context.Context, groupID string, limit int, cursor string) ([]AdminBannedMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	_ = cursor
	items := make([]AdminBannedMemberItem, 0, len(r.banned[groupID]))
	for userID, record := range r.banned[groupID] {
		item := AdminBannedMemberItem{
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
	limit = types.NormalizeLimit(limit)
	// 消费游标：跳过 (BannedAt, UserID) >= cursor 的项（DESC 序）。
	// 游标与 PG 共用 groupPageCursorCodec（审查 P2-8-5）。
	if cursor != "" {
		cursorAt, cursorUID, err := decodeGroupPageCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		start := 0
		for i, item := range items {
			itemAt, err := time.Parse(time.RFC3339Nano, item.BannedAt)
			if err != nil {
				return nil, "", err
			}
			if itemAt.Before(cursorAt) || (itemAt.Equal(cursorAt) && item.User.UserID > cursorUID) {
				break
			}
			start = i + 1
		}
		items = items[start:]
	}
	if len(items) > limit {
		last := items[limit-1]
		lastAt, err := time.Parse(time.RFC3339Nano, last.BannedAt)
		if err != nil {
			return nil, "", err
		}
		return items[:limit], encodeGroupPageCursor(lastAt, last.User.UserID), nil
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
	group.OwnerID = &req.NewOwnerID
	r.groups[req.GroupID] = group
	return nil
}

func (r *InMemoryRepository) cloneGroupForUser(group GroupItem, userID string) GroupItem {
	cloned := group
	if memberships, ok := r.members[group.Group.GroupID]; ok {
		cloned.Group.MembersCount = len(memberships)
		if membership, joined := memberships[userID]; joined {
			state := &GroupViewerState{Role: membership.Role, UnreadCount: 0, IsMuted: membership.IsMuted, MutedUntil: groupMutedUntil(membership.MutedUntil), IsPinned: membership.IsPinned, JoinedAt: membership.JoinedAt}
			cloned.ViewerState = state
		} else {
			cloned.ViewerState = nil
		}
	}
	return cloned
}
