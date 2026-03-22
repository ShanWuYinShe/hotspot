package groupmessages

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[string][]GroupMessageItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{items: make(map[string][]GroupMessageItem)}
}
func (r *InMemoryRepository) Send(_ context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	now := time.Now().UTC()
	msg := GroupMessageItem{MessageID: fmt.Sprintf("gm_%d", now.UnixNano()), Type: req.Type, Content: req.Content, SentAt: now.Format(time.RFC3339)}
	r.mu.Lock()
	r.items[req.GroupID] = append(r.items[req.GroupID], msg)
	r.mu.Unlock()
	return MessageSent{MessageID: msg.MessageID, SentAt: msg.SentAt}, nil
}
func (r *InMemoryRepository) History(_ context.Context, groupID string) (PaginatedGroupMessages, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]GroupMessageItem(nil), r.items[groupID]...)
	sort.Slice(items, func(i, j int) bool { return items[i].MessageID > items[j].MessageID })
	return PaginatedGroupMessages{Items: items}, nil
}
