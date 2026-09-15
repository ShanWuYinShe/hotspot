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
	record.Status = ApplicationStatusAccepted
	now := time.Now().UTC().Format(time.RFC3339)
	record.HandledAt = &now
	record.HandledBy = &UserInfo{UserID: approverID, Nickname: approverID}
	r.applications[applicationID] = record
	if group, ok := r.groups[record.GroupID]; ok && group.MaxMembers > 0 && len(r.members[record.GroupID]) >= group.MaxMembers {
		return ErrGroupFull
	}
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
	record.Status = ApplicationStatusRejected
	now := time.Now().UTC().Format(time.RFC3339)
	record.HandledAt = &now
	if reason != nil {
		record.Message = reason
	}
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
