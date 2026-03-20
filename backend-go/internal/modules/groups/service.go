package groups

import (
	"context"
	"fmt"
	"sync"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Service struct {
	mu     sync.RWMutex
	groups map[string]GroupItem
}

func NewService() *Service {
	return &Service{groups: make(map[string]GroupItem)}
}

func (s *Service) Create(ctx context.Context, req CreateGroupRequest) GroupItem {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	groupID := fmt.Sprintf("group_%d", time.Now().UTC().UnixNano())
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
	item := GroupItem{
		Group:        GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: req.AvatarFileID, MembersCount: 1},
		Description:  req.Description,
		Tags:         req.Tags,
		HasPassword:  req.Password != "",
		Location:     req.Location,
		MyState:      &MyGroupState{Role: "owner", IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: time.Now().UTC().Format(time.RFC3339)},
		JoinMethod:   joinMethod,
		IsSearchable: isSearchable,
		MaxMembers:   maxMembers,
	}
	_ = principal
	s.groups[groupID] = item
	return item
}

func (s *Service) OpenTheDoor(ctx context.Context, groupID string) (GroupItem, bool) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	_ = principal
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.groups[groupID]
	return item, ok
}

func (s *Service) Search(ctx context.Context, keyword string) PaginatedGroups {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]GroupItem, 0, len(s.groups))
	for _, item := range s.groups {
		if keyword == "" || item.Group.Name == keyword {
			items = append(items, item)
		}
	}
	return PaginatedGroups{Items: items}
}
