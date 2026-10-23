package groups

import (
	"context"

	"sort"

	"time"
)

func (r *InMemoryRepository) GetApplication(_ context.Context, applicationID string) (GroupApplicationItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return GroupApplicationItem{}, false, nil
	}
	group := r.groups[record.GroupID]
	return GroupApplicationItem{
		ApplicationID: record.ApplicationID,
		GroupID:       record.GroupID,
		Group:         group.Group,
		Applicant:     UserInfo{UserID: record.ApplicantID, Nickname: record.ApplicantID},
		Message:       record.Message,
		Status:        record.Status,
		CreatedAt:     record.CreatedAt,
		HandledAt:     record.HandledAt,
	}, true, nil
}

func (r *InMemoryRepository) ListApplications(_ context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	items := make([]GroupApplicationItem, 0)
	for _, record := range r.applications {
		if record.GroupID != groupID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		group := r.groups[record.GroupID]
		items = append(items, GroupApplicationItem{
			ApplicationID: record.ApplicationID,
			Group:         group.Group,
			Applicant:     UserInfo{UserID: record.ApplicantID, Nickname: record.ApplicantID},
			Message:       record.Message,
			Status:        record.Status,
			CreatedAt:     record.CreatedAt,
			HandledAt:     record.HandledAt,
		})
	}
	return items, "", nil
}

func (r *InMemoryRepository) ApproveApplication(_ context.Context, approverID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrAlreadyProcessed
	}
	// 与 Postgres 实现的审批事务内封禁复检同语义（审计 F-04）。
	if banned, ok := r.banned[record.GroupID]; ok {
		if _, isBanned := banned[record.ApplicantID]; isBanned {
			return ErrBannedInGroup
		}
	}
	// 容量检查前置（审查 P2-8-1）：避免「申请已通过但人没入群」的半提交态，
	// 与 PG 事务顺序（容量→状态→插成员）一致。
	if group, ok := r.groups[record.GroupID]; ok && group.MaxMembers > 0 && len(r.members[record.GroupID]) >= group.MaxMembers {
		return ErrGroupFull
	}
	now := time.Now().UTC().Format(time.RFC3339)
	record.Status = ApplicationStatusAccepted
	record.HandledAt = &now
	record.HandledBy = &UserInfo{UserID: approverID, Nickname: approverID}
	r.applications[applicationID] = record
	if r.members[record.GroupID] == nil {
		r.members[record.GroupID] = make(map[string]inMemoryMembership)
	}
	r.members[record.GroupID][record.ApplicantID] = inMemoryMembership{Role: GroupRoleMember, JoinedAt: now}
	if group, ok := r.groups[record.GroupID]; ok {
		group.Group.MembersCount = len(r.members[record.GroupID])
		r.groups[record.GroupID] = group
	}
	return nil
}

func (r *InMemoryRepository) RejectApplication(_ context.Context, applicationID string, reason *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrAlreadyProcessed
	}
	// 拒绝原因不入 message：PG 的 queryUpdateGroupApplication 只更新状态/处理人，
	// 从不覆盖申请人的原始留言（审查 P2-8-2，此前内存实现把留言替换成拒绝理由，
	// 列表回读时原始留言丢失）。
	record.Status = ApplicationStatusRejected
	now := time.Now().UTC().Format(time.RFC3339)
	record.HandledAt = &now
	_ = reason
	r.applications[applicationID] = record
	return nil
}

func (r *InMemoryRepository) ListMyApplications(_ context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]GroupApplicationItem, 0)
	for _, record := range r.applications {
		if record.ApplicantID != userID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		group := r.groups[record.GroupID]
		items = append(items, GroupApplicationItem{ApplicationID: record.ApplicationID, Group: group.Group, Applicant: UserInfo{UserID: record.ApplicantID, Nickname: record.ApplicantID}, Message: record.Message, Status: record.Status, CreatedAt: record.CreatedAt, HandledAt: record.HandledAt})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ApplicationID > items[j].ApplicationID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	// 分页参数此前被忽略（审查 P2-8-6），对齐 PG 的 keyset 语义（明文
	// createdAt:id 游标，与同模块 ListMembers 约定一致）。
	// 分页（审查 P2-8-6）与游标编码（审查 P2-8-5）：与 PG 共用
	// groupPageCursorCodec（此前明文 createdAt:id 与 PG 的 AES-GCM 游标互不兼容）。
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
			if itemAt.Before(cursorAt) || (itemAt.Equal(cursorAt) && item.ApplicationID < cursorID) {
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
		return items[:limit], encodeGroupPageCursor(lastAt, items[limit-1].ApplicationID), nil
	}
	return items, "", nil
}

func (r *InMemoryRepository) CancelApplication(_ context.Context, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return ErrGroupNotFound
	}
	// 与 PG 版（pending 守卫）对齐：仅 pending 申请可撤销，防止已 accepted
	// 的申请被申请人改回 revoked 产生「已是成员但申请被撤销」的矛盾数据。
	if record.Status != ApplicationStatusPending {
		return ErrAlreadyProcessed
	}
	record.Status = ApplicationStatusRevoked
	r.applications[applicationID] = record
	return nil
}
