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

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	momentID := fmt.Sprintf("moment_%d", now.UnixNano())
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	item := MomentItem{MomentID: momentID, Description: req.Description, Tags: normalizeMomentTags(req.Tags), Mentions: req.Mentions, CreatedAt: now.Format(time.RFC3339), LikesCount: 0, CommentsCount: 0, SharesCount: 0, User: UserInfo{UserID: userID, Nickname: userID}, Location: req.Location, PrivacyLevel: privacy, MyState: &MyMomentState{IsLiked: false, IsFavorited: false}}
	r.moments[momentID] = item
	r.byUser[userID] = append(r.byUser[userID], momentID)
	return item, nil
}

func (r *InMemoryRepository) Get(_ context.Context, momentID string) (MomentItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.moments[momentID]
	return item, ok, nil
}

func (r *InMemoryRepository) ListMine(_ context.Context, userID string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := append([]string(nil), r.byUser[userID]...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	items := make([]MomentItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, r.moments[id])
	}
	return PaginatedMoments{Items: items}, nil
}

func (r *InMemoryRepository) Exists(_ context.Context, momentID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.moments[momentID]
	return ok, nil
}

func (r *InMemoryRepository) IncrementCommentCount(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return nil
	}
	item.CommentsCount++
	r.moments[momentID] = item
	return nil
}

func (r *InMemoryRepository) Search(_ context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []MomentItem
	for _, m := range r.moments {
		if m.PrivacyLevel == "private" {
			continue
		}
		if req.Keyword != nil && *req.Keyword != "" {
			if !containsString(m.Description, *req.Keyword) {
				continue
			}
		}
		if req.Tag != nil && *req.Tag != "" {
			if !containsTag(m.Tags, *req.Tag) {
				continue
			}
		}
		items = append(items, m)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	if len(items) == 0 {
		items = []MomentItem{}
	}
	return PaginatedMoments{Items: items}, nil
}

func (r *InMemoryRepository) Update(_ context.Context, userID string, req EditMomentRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[req.MomentID]
	if !ok {
		return ErrMomentNotFound
	}
	if item.User.UserID != userID {
		return ErrPermissionDenied
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Location != nil {
		item.Location = *req.Location
	}
	if req.PrivacyLevel != nil {
		item.PrivacyLevel = *req.PrivacyLevel
	}
	if req.Tags != nil {
		item.Tags = normalizeMomentTags(req.Tags)
	}
	r.moments[req.MomentID] = item
	return nil
}

func (r *InMemoryRepository) Delete(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return ErrMomentNotFound
	}
	delete(r.moments, momentID)
	userIDs := r.byUser[item.User.UserID]
	for i, id := range userIDs {
		if id == momentID {
			r.byUser[item.User.UserID] = append(userIDs[:i], userIDs[i+1:]...)
			break
		}
	}
	return nil
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
