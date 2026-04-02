package private_messages

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu         sync.RWMutex
	messages   map[string][]PrivateMessageItem
	readCursor map[string]map[string]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{messages: make(map[string][]PrivateMessageItem), readCursor: make(map[string]map[string]string)}
}
func conversationKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}
func (r *InMemoryRepository) Send(_ context.Context, senderID, targetUserID string, req SendPrivateMessageRequest) (MessageSent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	msg := PrivateMessageItem{MessageID: "pm_" + now.Format("20060102150405.000000000"), Type: req.Type, Content: req.Content, ShareCode: req.ShareCode, Sender: UserInfo{UserID: senderID, Nickname: senderID}, SentAt: now.Format(time.RFC3339Nano), IsRead: false}
	if req.FileID != "" {
		msg.Media = &MessageMedia{URL: req.FileID}
	}
	if req.ReplyToMessageID != "" {
		for _, item := range r.messages[conversationKey(senderID, targetUserID)] {
			if item.MessageID == req.ReplyToMessageID {
				reply := MessageReplyTo{MessageID: item.MessageID, Type: item.Type, Content: item.Content, ShareCode: item.ShareCode, Sender: item.Sender, SentAt: item.SentAt}
				msg.ReplyTo = &reply
				break
			}
		}
	}
	key := conversationKey(senderID, targetUserID)
	r.messages[key] = append(r.messages[key], msg)
	return MessageSent{MessageID: msg.MessageID, SentAt: msg.SentAt}, nil
}
func (r *InMemoryRepository) History(_ context.Context, userID string, req GetHistoryRequest) (PaginatedMessages, error) {
	targetUserID := req.TargetUserID
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
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		cursor = req.Pagination.Cursor
	}
	if cursor != "" {
		start := 0
		for start < len(items) && items[start].MessageID != cursor {
			start++
		}
		if start == len(items) {
			return PaginatedMessages{Items: []PrivateMessageItem{}}, nil
		}
		items = items[start+1:]
	}
	if len(items) <= limit {
		return PaginatedMessages{Items: items}, nil
	}
	return PaginatedMessages{Items: items[:limit], NextCursor: items[limit-1].MessageID}, nil
}
func (r *InMemoryRepository) Recent(ctx context.Context, userID, targetUserID string, limit int, deletedAfter *time.Time) (PaginatedMessages, error) {
	history, err := r.History(ctx, userID, GetHistoryRequest{TargetUserID: targetUserID})
	if err != nil {
		return PaginatedMessages{}, err
	}
	if deletedAfter != nil {
		filtered := history.Items[:0]
		for _, item := range history.Items {
			sentAt, err := time.Parse(time.RFC3339Nano, item.SentAt)
			if err == nil && !sentAt.After(*deletedAfter) {
				continue
			}
			filtered = append(filtered, item)
		}
		history.Items = filtered
	}
	if limit > 0 && len(history.Items) > limit {
		history.Items = history.Items[:limit]
	}
	return history, nil
}
func (r *InMemoryRepository) Last(ctx context.Context, userID, targetUserID string, deletedAfter *time.Time) (*PrivateMessageItem, error) {
	recent, err := r.Recent(ctx, userID, targetUserID, 1, deletedAfter)
	if err != nil {
		return nil, err
	}
	if len(recent.Items) == 0 {
		return nil, nil
	}
	return &recent.Items[0], nil
}

func (r *InMemoryRepository) UnreadCount(_ context.Context, userID, targetUserID string, deletedAfter *time.Time) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := conversationKey(userID, targetUserID)
	lastReadMessageID := ""
	if r.readCursor[key] != nil {
		lastReadMessageID = r.readCursor[key][userID]
	}
	lastReadSentAt := ""
	if lastReadMessageID != "" {
		for _, item := range r.messages[key] {
			if item.MessageID == lastReadMessageID {
				lastReadSentAt = item.SentAt
				break
			}
		}
	}
	count := 0
	for _, item := range r.messages[key] {
		if item.Sender.UserID == userID {
			continue
		}
		if deletedAfter != nil {
			sentAt, err := time.Parse(time.RFC3339Nano, item.SentAt)
			if err == nil && !sentAt.After(*deletedAfter) {
				continue
			}
		}
		if lastReadSentAt != "" {
			if item.SentAt < lastReadSentAt {
				continue
			}
			if item.SentAt == lastReadSentAt && item.MessageID <= lastReadMessageID {
				continue
			}
		}
		count++
	}
	return count, nil
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
				return ErrMessagePermissionDenied
			}
			if msgs[i].Type != "text" || msgs[i].RecalledAt != "" {
				return ErrMessageNotEditable
			}
			msgs[i].Content = req.Content
			msgs[i].EditedAt = time.Now().UTC().Format(time.RFC3339Nano)
			r.messages[key] = msgs
			return nil
		}
	}
	return ErrMessageNotFound
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
				return ErrMessagePermissionDenied
			}
			if msgs[i].RecalledAt != "" {
				return ErrMessageNotRecallable
			}
			sentAt, err := time.Parse(time.RFC3339Nano, msgs[i].SentAt)
			if err == nil && time.Since(sentAt) > 2*time.Minute {
				return ErrMessageNotRecallable
			}
			msgs[i].RecalledAt = time.Now().UTC().Format(time.RFC3339Nano)
			r.messages[key] = msgs
			return nil
		}
	}
	return ErrMessageNotFound
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, userID string, req MarkPrivateMessagesReadRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := conversationKey(userID, req.TargetUserID)
	msgs := r.messages[key]
	if req.UpToMessageID != "" {
		found := false
		for i := range msgs {
			if msgs[i].MessageID == req.UpToMessageID {
				found = true
				break
			}
		}
		if !found {
			return ErrMessageNotFound
		}
	}
	if r.readCursor[key] == nil {
		r.readCursor[key] = map[string]string{}
	}
	if req.UpToMessageID != "" {
		r.readCursor[key][userID] = req.UpToMessageID
		return nil
	}
	latestMessageID := ""
	latestSentAt := ""
	for _, item := range msgs {
		if item.SentAt > latestSentAt || (item.SentAt == latestSentAt && item.MessageID > latestMessageID) {
			latestSentAt = item.SentAt
			latestMessageID = item.MessageID
		}
	}
	if latestMessageID != "" {
		r.readCursor[key][userID] = latestMessageID
	}
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
		latestMessageID := ""
		latestSentAt := ""
		for _, item := range msgs {
			if item.SentAt > latestSentAt || (item.SentAt == latestSentAt && item.MessageID > latestMessageID) {
				latestSentAt = item.SentAt
				latestMessageID = item.MessageID
			}
		}
		if latestMessageID == "" {
			continue
		}
		if r.readCursor[key] == nil {
			r.readCursor[key] = map[string]string{}
		}
		r.readCursor[key][userID] = latestMessageID
	}
	return nil
}
