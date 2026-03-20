package moments

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu      sync.RWMutex
	moments map[string]MomentItem
	byUser  map[string][]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{moments: make(map[string]MomentItem), byUser: make(map[string][]string)}
}

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateMomentRequest) MomentItem {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	momentID := fmt.Sprintf("moment_%d", now.UnixNano())
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	item := MomentItem{MomentID: momentID, Description: req.Description, Tags: req.Tags, Mentions: req.Mentions, CreatedAt: now.Format(time.RFC3339), LikesCount: 0, CommentsCount: 0, SharesCount: 0, User: UserInfo{UserID: userID, Nickname: userID}, Location: req.Location, PrivacyLevel: privacy, MyState: &MyMomentState{IsLiked: false, IsFavorited: false}}
	r.moments[momentID] = item
	r.byUser[userID] = append(r.byUser[userID], momentID)
	return item
}
func (r *InMemoryRepository) Get(_ context.Context, momentID string) (MomentItem, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.moments[momentID]
	return item, ok
}
func (r *InMemoryRepository) ListMine(_ context.Context, userID string) PaginatedMoments {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := append([]string(nil), r.byUser[userID]...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	items := make([]MomentItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, r.moments[id])
	}
	return PaginatedMoments{Items: items}
}
func (r *InMemoryRepository) IncrementCommentCount(_ context.Context, momentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return
	}
	item.CommentsCount++
	r.moments[momentID] = item
}
