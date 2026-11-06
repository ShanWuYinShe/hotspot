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
	// 分页（审查 P2-8-6）与游标编码（审查 P2-8-5）：与 PG 共用
	// groupPageCursorCodec——此前内存版是明文 createdAt:id，与 PG 的 AES-GCM
	// 游标互不兼容，且 RFC3339 时间串自带冒号无法可靠拆分。
	if cursor != "" {
		cursorAt, cursorID, err := decodeGroupPageCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		start := 0
		for i, item := range items {
			itemAt, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
			if err != nil {
				return nil, "", err
			}
			// 列表按 created_at DESC, id DESC：跳过游标及之前的项，遇到严格
			// 更小的 (created_at, id) 即下一页起点。
			if itemAt.Before(cursorAt) || (itemAt.Equal(cursorAt) && item.InviteID < cursorID) {
				break
			}
			start = i + 1
		}
		items = items[start:]
	}
	if len(items) > limit {
		lastAt, err := time.Parse(time.RFC3339Nano, items[limit-1].CreatedAt)
		if err != nil {
			return nil, "", err
		}
		return items[:limit], encodeGroupPageCursor(lastAt, items[limit-1].InviteID), nil
	}
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
	// 容量检查必须在状态变更之前（审查 P2-8-1）：此前先置 accepted 再查容量，
	// 群满返回 ErrGroupFull 时邀请已被污染成「已接受但人没入群」，与 PG 事务
	// 内的顺序（容量→状态→插成员）不一致。
	if group, ok := r.groups[record.GroupID]; ok && group.MaxMembers > 0 && len(r.members[record.GroupID]) >= group.MaxMembers {
		return ErrGroupFull
	}
	now := time.Now().UTC().Format(time.RFC3339)
	record.Status = ApplicationStatusAccepted
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
	// 分页（审查 P2-8-6）与游标编码（审查 P2-8-5）：与 PG 共用
	// groupPageCursorCodec——此前内存版是明文 createdAt:id，与 PG 的 AES-GCM
	// 游标互不兼容，且 RFC3339 时间串自带冒号无法可靠拆分。
	if cursor != "" {
		cursorAt, cursorID, err := decodeGroupPageCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		start := 0
		for i, item := range items {
			itemAt, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
			if err != nil {
				return nil, "", err
			}
			// 列表按 created_at DESC, id DESC：跳过游标及之前的项，遇到严格
			// 更小的 (created_at, id) 即下一页起点。
			if itemAt.Before(cursorAt) || (itemAt.Equal(cursorAt) && item.InviteID < cursorID) {
				break
			}
			start = i + 1
		}
		items = items[start:]
	}
	if len(items) > limit {
		lastAt, err := time.Parse(time.RFC3339Nano, items[limit-1].CreatedAt)
		if err != nil {
			return nil, "", err
		}
		return items[:limit], encodeGroupPageCursor(lastAt, items[limit-1].InviteID), nil
	}
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
