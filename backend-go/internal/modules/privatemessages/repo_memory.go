package privatemessages

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/types"
)

type InMemoryRepository struct {
	mu         sync.RWMutex
	messages   map[string][]PrivateMessageItem
	readCursor map[string]map[string]string
	seq        int64
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
func (r *InMemoryRepository) Send(_ context.Context, senderID, targetUserID string, req SendPrivateMessageRequest) (MessageSentResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	r.seq++
	sentAt := now.Format("2006-01-02T15:04:05.000000000Z")
	msg := PrivateMessageItem{MessageID: fmt.Sprintf("pm_%d_%s", r.seq, now.Format("20060102150405.000000000")), Type: req.Type, Content: types.StrPtr(req.Content), ShareCode: types.StrPtr(req.ShareCode), Sender: UserInfo{UserID: senderID, Nickname: senderID}, SentAt: sentAt, IsRead: false}
	if req.FileID != "" {
		msg.Media = &MessageMedia{URL: req.FileID}
	}
	if req.ReplyToMessageID != "" {
		for _, item := range r.messages[conversationKey(senderID, targetUserID)] {
			if item.MessageID == req.ReplyToMessageID {
				reply := MessageBrief{MessageID: item.MessageID, Type: item.Type, Content: item.Content, ShareCode: item.ShareCode, Sender: item.Sender, SentAt: item.SentAt}
				msg.ReplyTo = &reply
				break
			}
		}
	}
	key := conversationKey(senderID, targetUserID)
	r.messages[key] = append(r.messages[key], msg)
	return MessageSentResult{MessageID: msg.MessageID, SentAt: msg.SentAt}, nil
}
func (r *InMemoryRepository) History(_ context.Context, userID string, req ListPrivateMessagesRequest) (PaginatedMessages, error) {
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
	cursorValue := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		cursorValue = req.Pagination.Cursor
	}
	cursorSentAt, cursorMessageID, err := decodeMessageCursor(cursorValue)
	if err != nil {
		return PaginatedMessages{}, err
	}
	if !cursorSentAt.IsZero() {
		filtered := make([]PrivateMessageItem, 0, len(items))
		for _, item := range items {
			sentAt, parseErr := time.Parse("2006-01-02T15:04:05.000000000Z", item.SentAt)
			if parseErr != nil {
				continue
			}
			if sentAt.Before(cursorSentAt) || (sentAt.Equal(cursorSentAt) && item.MessageID < cursorMessageID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if len(items) <= limit {
		return PaginatedMessages{Items: items}, nil
	}
	next := items[limit-1]
	c := encodeMessageCursor(next.SentAt, next.MessageID)
	return PaginatedMessages{Items: items[:limit], NextCursor: &c}, nil
}
func (r *InMemoryRepository) Recent(ctx context.Context, userID, targetUserID string, limit int, deletedAfter *time.Time) (PaginatedMessages, error) {
	history, err := r.History(ctx, userID, ListPrivateMessagesRequest{TargetUserID: targetUserID})
	if err != nil {
		return PaginatedMessages{}, err
	}
	if deletedAfter != nil {
		filtered := history.Items[:0]
		for _, item := range history.Items {
			sentAt, err := time.Parse("2006-01-02T15:04:05.000000000Z", item.SentAt)
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
			sentAt, err := time.Parse("2006-01-02T15:04:05.000000000Z", item.SentAt)
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

func (r *InMemoryRepository) Update(_ context.Context, userID string, req UpdatePrivateMessageRequest) (MessageEventMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, msgs := range r.messages {
		for i := range msgs {
			if msgs[i].MessageID != req.MessageID {
				continue
			}
			if msgs[i].Sender.UserID != userID {
				return MessageEventMeta{}, ErrMessagePermissionDenied
			}
			if msgs[i].Type != "text" || msgs[i].RecalledAt != nil {
				return MessageEventMeta{}, ErrMessageNotEditable
			}
			msgs[i].Content = types.StrPtr(req.Content)
			editedAt := time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")
			msgs[i].EditedAt = &editedAt
			r.messages[key] = msgs
			return MessageEventMeta{TargetUserID: otherUserID(key, userID), MessageID: req.MessageID, SentAt: msgs[i].SentAt, Content: req.Content}, nil
		}
	}
	return MessageEventMeta{}, ErrMessageNotFound
}

func (r *InMemoryRepository) Recall(_ context.Context, userID string, req RecallPrivateMessageRequest) (MessageEventMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, msgs := range r.messages {
		for i := range msgs {
			if msgs[i].MessageID != req.MessageID {
				continue
			}
			if msgs[i].Sender.UserID != userID {
				return MessageEventMeta{}, ErrMessagePermissionDenied
			}
			if msgs[i].RecalledAt != nil {
				return MessageEventMeta{}, ErrMessageNotRecallable
			}
			sentAt, err := time.Parse("2006-01-02T15:04:05.000000000Z", msgs[i].SentAt)
			if err == nil && time.Since(sentAt) > 2*time.Minute {
				return MessageEventMeta{}, ErrMessageNotRecallable
			}
			recalledAt := time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")
			msgs[i].RecalledAt = &recalledAt
			r.messages[key] = msgs
			return MessageEventMeta{TargetUserID: otherUserID(key, userID), MessageID: req.MessageID, SentAt: msgs[i].SentAt}, nil
		}
	}
	return MessageEventMeta{}, ErrMessageNotFound
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, userID string, req MarkPrivateMessagesReadRequest) (MessageEventMeta, error) {
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
			return MessageEventMeta{}, ErrMessageNotFound
		}
	}
	if r.readCursor[key] == nil {
		r.readCursor[key] = map[string]string{}
	}
	if req.UpToMessageID != "" {
		r.readCursor[key][userID] = req.UpToMessageID
		return MessageEventMeta{TargetUserID: req.TargetUserID, MessageID: req.UpToMessageID}, nil
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
	return MessageEventMeta{TargetUserID: req.TargetUserID, MessageID: latestMessageID, SentAt: latestSentAt}, nil
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

func otherUserID(chatKey, userID string) string {
	parts := strings.Split(chatKey, ":")
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == userID {
		return parts[1]
	}
	return parts[0]
}

func (r *InMemoryRepository) GetMessageForReply(ctx context.Context, messageID, viewerUserID string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (r *InMemoryRepository) MessageParticipants(_ context.Context, messageID string) (string, string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, items := range r.messages {
		for _, item := range items {
			if item.MessageID == messageID {
				// 已撤回消息按不存在处理（与 PG 侧 is_recalled = FALSE 同口径，
				// 审计 F-11）：该方法仅被 IsReactionTargetVisible 消费。
				if item.RecalledAt != nil {
					return "", "", false, nil
				}
				// 会话键为 userA:userB；接收方为键中除发送者外的另一方
				return item.Sender.UserID, otherUserID(key, item.Sender.UserID), true, nil
			}
		}
	}
	return "", "", false, nil
}

func (r *InMemoryRepository) MessageExists(_ context.Context, messageID string) (bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, items := range r.messages {
		for _, item := range items {
			if item.MessageID == messageID {
				return true, item.Sender.UserID, nil
			}
		}
	}
	return false, "", nil
}

// GetMessageReportSnapshot 内存版私聊消息快照：target_chat_id 为相对发送者
// 的会话对方；file_id 取自 Media（内存 Send 以 URL 字段承载 file_id）。
func (r *InMemoryRepository) GetMessageReportSnapshot(_ context.Context, messageID string) (MessageReportSnapshot, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, items := range r.messages {
		for _, item := range items {
			if item.MessageID != messageID {
				continue
			}
			content := ""
			if item.Content != nil {
				content = *item.Content
			}
			fileID := ""
			if item.Media != nil {
				fileID = item.Media.URL
			}
			return MessageReportSnapshot{
				SenderUserID: item.Sender.UserID,
				TargetUserID: otherUserID(key, item.Sender.UserID),
				Type:         item.Type,
				Content:      content,
				FileID:       fileID,
				SentAt:       item.SentAt,
			}, true, nil
		}
	}
	return MessageReportSnapshot{}, false, nil
}

func (r *InMemoryRepository) CountMessagesByTarget(_ context.Context, senderID string, since time.Time) (map[string]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	counts := make(map[string]int)
	for chatKey, items := range r.messages {
		targetUserID := otherUserID(chatKey, senderID)
		if targetUserID == "" {
			continue
		}
		for _, item := range items {
			if item.Sender.UserID != senderID {
				continue
			}
			sentAt, err := time.Parse(time.RFC3339Nano, item.SentAt)
			if err != nil {
				continue
			}
			if !sentAt.Before(since) {
				counts[targetUserID]++
			}
		}
	}
	return counts, nil
}

func (r *InMemoryRepository) GetMediaForMessage(ctx context.Context, fileID, ownerUserID string) (map[string]any, bool, error) {
	return nil, false, nil
}
