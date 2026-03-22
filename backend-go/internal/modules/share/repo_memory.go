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

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateShareRequest) (ShareCreated, error) {
	now := time.Now().UTC()
	shareID := fmt.Sprintf("share_%d", now.UnixNano())
	shareCode := strings.ToUpper(fmt.Sprintf("S%011d", now.UnixNano()%100000000000))
	record := shareRecord{ShareListItem: ShareListItem{ShareID: shareID, ShareCode: shareCode, ShareType: req.ShareType, CreatedAt: now.Format(time.RFC3339), IsRevoked: false}, Payload: req.Payload, Creator: map[string]any{"user_id": userID, "nickname": userID}}
	if req.ExpiresInSeconds > 0 {
		record.ExpiresAt = now.Add(time.Duration(req.ExpiresInSeconds) * time.Second).Format(time.RFC3339)
	}
	r.mu.Lock()
	r.shares[shareID] = record
	r.byUser[userID] = append(r.byUser[userID], shareID)
	r.mu.Unlock()
	return ShareCreated{ShareID: shareID, ShareCode: shareCode, ExpiresAt: record.ExpiresAt}, nil
}

func (r *InMemoryRepository) Get(_ context.Context, shareCode string) (ShareDetail, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now().UTC()
	for _, record := range r.shares {
		if record.ShareCode != strings.ToUpper(shareCode) || record.IsRevoked {
			continue
		}
		if record.ExpiresAt != "" {
			expiresAt, err := time.Parse(time.RFC3339, record.ExpiresAt)
			if err == nil && !expiresAt.After(now) {
				return ShareDetail{}, false, nil
			}
		}
		share := map[string]any{"share_id": record.ShareID, "share_code": record.ShareCode, "share_type": record.ShareType, "creator": record.Creator, "created_at": record.CreatedAt}
		if record.ExpiresAt != "" {
			share["expires_at"] = record.ExpiresAt
		}
		return ShareDetail{Share: share, Payload: record.Payload, Preview: record.Preview}, true, nil
	}
	return ShareDetail{}, false, nil
}

func (r *InMemoryRepository) List(_ context.Context, userID string) (PaginatedShares, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ShareListItem, 0)
	for _, id := range r.byUser[userID] {
		items = append(items, r.shares[id].ShareListItem)
	}
	return PaginatedShares{Items: items}, nil
}

func (r *InMemoryRepository) Revoke(_ context.Context, userID, shareID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.shares[shareID]
	if !ok || record.IsRevoked {
		return false, nil
	}
	found := false
	for _, id := range r.byUser[userID] {
		if id == shareID {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}
	record.IsRevoked = true
	r.shares[shareID] = record
	return true, nil
}
