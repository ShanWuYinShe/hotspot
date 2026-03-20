package private_messages

import (
	"context"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu       sync.RWMutex
	messages map[string][]PrivateMessageItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{messages: make(map[string][]PrivateMessageItem)}
}
func conversationKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}
func (r *InMemoryRepository) Send(_ context.Context, senderID, targetUserID string, req SendPrivateMessageRequest) MessageSent {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	msg := PrivateMessageItem{MessageID: "pm_" + now.Format("20060102150405.000000000"), Type: req.Type, Content: req.Content, ShareCode: req.ShareCode, Sender: UserInfo{UserID: senderID, Nickname: senderID}, SentAt: now.Format(time.RFC3339), IsRead: false}
	key := conversationKey(senderID, targetUserID)
	r.messages[key] = append(r.messages[key], msg)
	return MessageSent{MessageID: msg.MessageID, SentAt: msg.SentAt}
}
func (r *InMemoryRepository) History(_ context.Context, userID, targetUserID string) PaginatedMessages {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := conversationKey(userID, targetUserID)
	items := append([]PrivateMessageItem(nil), r.messages[key]...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].SentAt == items[j].SentAt {
			return items[i].MessageID > items[j].MessageID
		}
		return items[i].SentAt > items[j].SentAt
	})
	return PaginatedMessages{Items: items}
}
func (r *InMemoryRepository) Recent(ctx context.Context, userID, targetUserID string, limit int) PaginatedMessages {
	history := r.History(ctx, userID, targetUserID)
	if len(history.Items) > limit {
		history.Items = history.Items[:limit]
	}
	return history
}
func (r *InMemoryRepository) Last(ctx context.Context, userID, targetUserID string) *PrivateMessageItem {
	history := r.History(ctx, userID, targetUserID)
	if len(history.Items) == 0 {
		return nil
	}
	return &history.Items[0]
}
