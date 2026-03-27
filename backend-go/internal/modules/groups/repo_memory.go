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
