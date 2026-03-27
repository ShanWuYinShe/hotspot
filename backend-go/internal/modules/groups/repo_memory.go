package groups

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu     sync.RWMutex
	groups map[string]GroupItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{groups: make(map[string]GroupItem)}
}
func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateGroupRequest) (GroupItem, error) {
	_ = userID
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	groupID := fmt.Sprintf("group_%d", now.UnixNano())
	isSearchable := true
	if req.IsSearchable != nil {
		isSearchable = *req.IsSearchable
	}
	maxMembers := req.MaxMembers
	if maxMembers == 0 {
		maxMembers = 200
	}
	joinMethod := req.JoinMethod
	if joinMethod == "" {
		joinMethod = "open"
	}
	item := GroupItem{Group: GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: req.AvatarFileID, MembersCount: 1}, Description: req.Description, Tags: req.Tags, HasPassword: req.Password != "", Location: req.Location, MyState: &MyGroupState{Role: "owner", IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: now.Format(time.RFC3339)}, JoinMethod: joinMethod, IsSearchable: isSearchable, MaxMembers: maxMembers}
	r.groups[groupID] = item
	return item, nil
}
func (r *InMemoryRepository) Get(_ context.Context, groupID string) (GroupItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.groups[groupID]
	return item, ok, nil
}
func (r *InMemoryRepository) Search(_ context.Context, keyword string) (PaginatedGroups, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []GroupItem{}
	for _, item := range r.groups {
		if keyword == "" || item.Group.Name == keyword {
			items = append(items, item)
		}
	}
	return PaginatedGroups{Items: items}, nil
}

func (r *InMemoryRepository) Update(_ context.Context, userID string, req UpdateGroupRequest) error {
	_ = userID
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.groups[req.GroupID]
	if !ok {
		return ErrGroupNotFound
	}
	if req.Name != "" {
		item.Group.Name = req.Name
	}
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.AvatarFileID != "" {
		item.Group.AvatarURL = req.AvatarFileID
	}
	if req.Tags != nil {
		item.Tags = req.Tags
	}
	if req.ClearPassword != nil && *req.ClearPassword {
		item.HasPassword = false
	} else if req.Password != nil {
		item.HasPassword = *req.Password != ""
	}
	if req.JoinMethod != nil {
		item.JoinMethod = *req.JoinMethod
	}
	if req.IsSearchable != nil {
		item.IsSearchable = *req.IsSearchable
	}
	if req.MaxMembers != nil {
		item.MaxMembers = *req.MaxMembers
	}
	if req.Location != nil {
		item.Location = *req.Location
	}
	r.groups[req.GroupID] = item
	return nil
}

func (r *InMemoryRepository) Dismiss(_ context.Context, groupID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	delete(r.groups, groupID)
	return nil
}

func (r *InMemoryRepository) ListMembers(_ context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group, ok := r.groups[groupID]
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	now := "2024-01-15T08:30:00.000Z"
	items := []GroupMemberItem{
		{
			User: UserInfo{
				UserID:    group.Group.GroupID + "_owner",
				Nickname:  group.Group.Name + " Owner",
				AvatarURL: "",
			},
			Role:       GroupRoleOwner,
			JoinedAt:   now,
		},
	}
	if role == nil || *role == GroupRoleMember {
		items = append(items, GroupMemberItem{
			User: UserInfo{
				UserID:    group.Group.GroupID + "_member",
				Nickname:  group.Group.Name + " Member",
				AvatarURL: "",
			},
			Role:     GroupRoleMember,
			JoinedAt: now,
		})
	}
	if len(items) > limit {
		return items[:limit], items[limit-1].JoinedAt + ":" + items[limit-1].User.UserID, nil
	}
	return items, "", nil
}

func (r *InMemoryRepository) SearchMembers(_ context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group, ok := r.groups[groupID]
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	now := "2024-01-15T08:30:00.000Z"
	items := []GroupMemberItem{}
	if role == nil || *role == GroupRoleOwner {
		items = append(items, GroupMemberItem{
			User: UserInfo{
				UserID:    group.Group.GroupID + "_owner",
				Nickname:  group.Group.Name + " Owner",
				AvatarURL: "",
			},
			Role:     GroupRoleOwner,
			JoinedAt: now,
		})
	}
	filtered := []GroupMemberItem{}
	for _, item := range items {
		if keyword == "" || item.User.Nickname == keyword {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) > limit {
		return filtered[:limit], filtered[limit-1].JoinedAt + ":" + filtered[limit-1].User.UserID, nil
	}
	return filtered, "", nil
}

func (r *InMemoryRepository) ListMyGroups(_ context.Context, userID string, limit int, cursor string) ([]GroupItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []GroupItem{}
	for _, group := range r.groups {
		if group.MyState != nil && group.MyState.IsJoined {
			items = append(items, group)
		}
	}
	if len(items) > limit {
		return items[:limit], items[limit-1].Group.GroupID, nil
	}
	return items, "", nil
}

func (r *InMemoryRepository) JoinGroup(_ context.Context, userID, groupID string, password *string, message *string) (JoinGroupResult, error) {
	_ = userID
	_ = password
	_ = message
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[groupID]
	if !ok {
		return JoinGroupResult{}, ErrGroupNotFound
	}
	if group.MyState != nil && group.MyState.IsJoined {
		return JoinGroupResult{Status: ApplicationStatusAccepted}, nil
	}
	if group.JoinMethod == GroupJoinMethodInviteOnly {
		return JoinGroupResult{}, ErrPermissionDenied
	}
	if group.JoinMethod == GroupJoinMethodApproval {
		appWrapper := &struct {
			ApplicationID string `json:"application_id"`
		}{
			ApplicationID: "app_" + groupID,
		}
		result := JoinGroupResult{Status: ApplicationStatusPending}
		result.Application = appWrapper
		return result, nil
	}
	group.MyState = &MyGroupState{Role: GroupRoleMember, IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false}
	group.Group.MembersCount++
	r.groups[groupID] = group
	return JoinGroupResult{Status: ApplicationStatusAccepted}, nil
}

func (r *InMemoryRepository) CreateAnnouncement(_ context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	announcementID := "announce_" + now.Format("20060102150405.000000000")
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}
	item := GroupAnnouncementItem{
		AnnouncementID: announcementID,
		GroupID:        req.GroupID,
		Title:          req.Title,
		Content:        req.Content,
		IsPinned:       isPinned,
		CreatedAt:      now.Format(time.RFC3339),
	}
	return item, nil
}

func (r *InMemoryRepository) ListAnnouncements(_ context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, ErrGroupNotFound
	}
	return []GroupAnnouncementItem{}, nil
}

func (r *InMemoryRepository) UpdateAnnouncement(_ context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	return nil
}

func (r *InMemoryRepository) DeleteAnnouncement(_ context.Context, groupID string, announcementID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	_ = announcementID
	return nil
}

func (r *InMemoryRepository) ListApplications(_ context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	return []GroupApplicationItem{}, "", nil
}

func (r *InMemoryRepository) ApproveApplication(_ context.Context, approverID, applicationID string) error {
	_ = approverID
	_ = applicationID
	return nil
}

func (r *InMemoryRepository) RejectApplication(_ context.Context, applicationID string, reason *string) error {
	_ = applicationID
	_ = reason
	return nil
}

func (r *InMemoryRepository) UpdateMySettings(_ context.Context, userID string, req UpdateGroupMySettingsRequest) error {
	_ = userID
	_ = req
	return nil
}

func (r *InMemoryRepository) LeaveGroup(_ context.Context, userID string, groupID string) error {
	_ = userID
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[groupID]
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || !group.MyState.IsJoined {
		return ErrPermissionDenied
	}
	if group.MyState.Role == GroupRoleOwner {
		return ErrPermissionDenied
	}
	group.MyState = nil
	r.groups[groupID] = group
	return nil
}

func (r *InMemoryRepository) ListMyApplications(_ context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	_ = userID
	_ = status
	_ = cursor
	return []GroupApplicationItem{}, "", nil
}

func (r *InMemoryRepository) CancelApplication(_ context.Context, applicationID string) error {
	_ = applicationID
	return nil
}

func (r *InMemoryRepository) ListMyInvites(_ context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	_ = userID
	_ = status
	_ = cursor
	return []GroupInviteItem{}, "", nil
}

func (r *InMemoryRepository) AcceptInvite(_ context.Context, userID string, inviteID string) error {
	_ = userID
	_ = inviteID
	return nil
}

func (r *InMemoryRepository) RejectInvite(_ context.Context, inviteID string) error {
	_ = inviteID
	return nil
}

func (r *InMemoryRepository) InviteMember(_ context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return Invitation{}, ErrGroupNotFound
	}
	_ = userID
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	return Invitation{
		InviteID:  "invite_" + now.Format("20060102150405.000000000"),
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (r *InMemoryRepository) ListSentInvites(_ context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	_ = status
	_ = cursor
	if limit == 0 {
		limit = 20
	}
	_ = limit
	return []GroupSentInviteItem{}, "", nil
}

func (r *InMemoryRepository) RevokeInvite(_ context.Context, groupID string, inviteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	_ = inviteID
	return nil
}

func (r *InMemoryRepository) SetMemberRole(_ context.Context, userID string, req SetMemberRoleRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	_ = userID
	return nil
}

func (r *InMemoryRepository) MuteMember(_ context.Context, userID string, req MuteMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	_ = userID
	return nil
}

func (r *InMemoryRepository) UnmuteMember(_ context.Context, userID string, req UnmuteMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	_ = userID
	return nil
}

func (r *InMemoryRepository) BanMember(_ context.Context, userID string, req BanMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	_ = userID
	return nil
}

func (r *InMemoryRepository) UnbanMember(_ context.Context, userID string, req UnbanMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
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
	if limit == 0 {
		limit = 20
	}
	_ = limit
	return []BannedMemberItem{}, "", nil
}

func (r *InMemoryRepository) RemoveMember(_ context.Context, userID string, req RemoveMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	_ = userID
	return nil
}

func (r *InMemoryRepository) TransferOwnership(_ context.Context, userID string, req TransferOwnershipRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	_ = userID
	return nil
}
