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
