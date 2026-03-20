package private_messages

import (
	"context"
	"sort"
	"strings"
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

func (r *InMemoryRepository) Update(_ context.Context, userID string, req UpdatePrivateMessageRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, msgs := range r.messages {
		for i := range msgs {
			if msgs[i].MessageID != req.MessageID {
				continue
			}
			if msgs[i].Sender.UserID != userID {
				return context.Canceled
			}
			if msgs[i].Type != "text" || msgs[i].RecalledAt != "" {
				return context.DeadlineExceeded
			}
			msgs[i].Content = req.Content
			msgs[i].EditedAt = time.Now().UTC().Format(time.RFC3339)
			r.messages[key] = msgs
			return nil
		}
	}
	return context.DeadlineExceeded
}

func (r *InMemoryRepository) Recall(_ context.Context, userID string, req RecallPrivateMessageRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, msgs := range r.messages {
		for i := range msgs {
			if msgs[i].MessageID != req.MessageID {
				continue
			}
			if msgs[i].Sender.UserID != userID {
				return context.Canceled
			}
			msgs[i].RecalledAt = time.Now().UTC().Format(time.RFC3339)
			r.messages[key] = msgs
			return nil
		}
	}
	return context.DeadlineExceeded
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, userID string, req MarkPrivateMessagesReadRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := conversationKey(userID, req.TargetUserID)
	msgs := r.messages[key]
	for i := range msgs {
		msgs[i].IsRead = true
		if req.UpToMessageID != "" && msgs[i].MessageID == req.UpToMessageID {
			break
		}
	}
	r.messages[key] = msgs
	return nil
}

func (r *InMemoryRepository) MarkAllRead(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, msgs := range r.messages {
		parts := strings.Split(key, ":")
		if len(parts) != 2 || (parts[0] != userID && parts[1] != userID) {
			continue
		}
		for i := range msgs {
			msgs[i].IsRead = true
		}
		r.messages[key] = msgs
	}
	return nil
}
