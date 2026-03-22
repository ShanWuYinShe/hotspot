package privatechats

import (
	"context"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu      sync.RWMutex
	created map[string]time.Time
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{created: make(map[string]time.Time)}
}
func key(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}
func (r *InMemoryRepository) GetOrCreate(_ context.Context, userID, targetUserID string) (time.Time, error) {
	k := key(userID, targetUserID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if createdAt, ok := r.created[k]; ok {
		return createdAt, nil
	}
	now := time.Now().UTC()
	r.created[k] = now
	return now, nil
}
func (r *InMemoryRepository) List(_ context.Context, userID string) (map[string]time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]time.Time{}
	for k, createdAt := range r.created {
		out[k] = createdAt
	}
	_ = userID
	return out, nil
}
