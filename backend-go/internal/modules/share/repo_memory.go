package share

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type shareRecord struct {
	ShareListItem
	Payload map[string]any
	Creator map[string]any
}

type InMemoryRepository struct {
	mu     sync.RWMutex
	shares map[string]shareRecord
	byUser map[string][]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{shares: make(map[string]shareRecord), byUser: make(map[string][]string)}
}
func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateShareRequest) ShareCreated {
	now := time.Now().UTC()
	shareID := fmt.Sprintf("share_%d", now.UnixNano())
	shareCode := strings.ToUpper(fmt.Sprintf("S%011d", now.UnixNano()%100000000000))
	record := shareRecord{ShareListItem: ShareListItem{ShareID: shareID, ShareCode: shareCode, ShareType: req.ShareType, CreatedAt: now.Format(time.RFC3339), IsRevoked: false}, Payload: req.Payload, Creator: map[string]any{"user_id": userID, "nickname": userID}}
	r.mu.Lock()
	r.shares[shareID] = record
	r.byUser[userID] = append(r.byUser[userID], shareID)
	r.mu.Unlock()
	return ShareCreated{ShareID: shareID, ShareCode: shareCode}
}
func (r *InMemoryRepository) Get(_ context.Context, shareCode string) (ShareDetail, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, record := range r.shares {
		if record.ShareCode == strings.ToUpper(shareCode) && !record.IsRevoked {
			return ShareDetail{Share: map[string]any{"share_id": record.ShareID, "share_code": record.ShareCode, "share_type": record.ShareType, "creator": record.Creator, "created_at": record.CreatedAt}, Payload: record.Payload, Preview: map[string]any{"title": record.ShareType}}, true
		}
	}
	return ShareDetail{}, false
}
func (r *InMemoryRepository) List(_ context.Context, userID string) PaginatedShares {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ShareListItem, 0)
	for _, id := range r.byUser[userID] {
		items = append(items, r.shares[id].ShareListItem)
	}
	return PaginatedShares{Items: items}
}
func (r *InMemoryRepository) Revoke(_ context.Context, userID, shareID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.shares[shareID]
	if !ok || record.IsRevoked {
		return false
	}
	found := false
	for _, id := range r.byUser[userID] {
		if id == shareID {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	record.IsRevoked = true
	r.shares[shareID] = record
	return true
}
