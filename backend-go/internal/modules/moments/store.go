package moments

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu      sync.RWMutex
	moments map[string]MomentItem
	byUser  map[string][]string
}

func NewStore() *Store {
	return &Store{moments: make(map[string]MomentItem), byUser: make(map[string][]string)}
}

func (s *Store) Create(userID string, req CreateMomentRequest) MomentItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	momentID := fmt.Sprintf("moment_%d", now.UnixNano())
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	item := MomentItem{
		MomentID:      momentID,
		Description:   req.Description,
		Tags:          normalizeMomentTags(req.Tags),
		Mentions:      req.Mentions,
		CreatedAt:     now.Format(time.RFC3339),
		LikesCount:    0,
		CommentsCount: 0,
		SharesCount:   0,
		User:          UserInfo{UserID: userID, Nickname: userID},
		Location:      &req.Location,
		PrivacyLevel:  privacy,
		IsFavorited:   false,
	}
	s.moments[momentID] = item
	s.byUser[userID] = append(s.byUser[userID], momentID)
	return item
}

func (s *Store) Get(momentID string) (MomentItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.moments[momentID]
	return item, ok
}

func (s *Store) ListMine(userID string) PaginatedMoments {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := append([]string(nil), s.byUser[userID]...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	items := make([]MomentItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, s.moments[id])
	}
	return PaginatedMoments{Items: items}
}

func (s *Store) IncrementCommentCount(momentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.moments[momentID]
	if !ok {
		return
	}
	item.CommentsCount++
	s.moments[momentID] = item
}
