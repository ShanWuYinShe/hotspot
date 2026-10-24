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
	key := conversationKey(senderID, targetUserID)
	if req.ReplyToMessageID != "" {
		found := false
		for _, item := range r.messages[key] {
			if item.MessageID == req.ReplyToMessageID {
				reply := MessageBrief{MessageID: item.MessageID, Type: item.Type, Content: item.Content, ShareCode: item.ShareCode, Sender: item.Sender, SentAt: item.SentAt}
				msg.ReplyTo = &reply
				found = true
				break
			}
		}
		// 与 PG 实现同口径：回复目标不存在或不属于本会话 → ErrMessageNotFound
		//（契约测试发现内存版此前静默忽略非法 reply_to）。
		if !found {
			return MessageSentResult{}, ErrMessageNotFound
		}
	}
	r.messages[key] = append(r.messages[key], msg)
	return MessageSentResult{MessageID: msg.MessageID, SentAt: msg.SentAt}, nil
}

// listConversationLocked 返回 (sent_at, message_id) 降序的会话消息副本，并按查看者
// 计算 is_read（对齐 PG queryPrivateMessagesHistory）：
//   - 对方发来的消息：查看即视为已读；
//   - 自己发出的消息：取决于对方（targetUserID）的已读游标。
//
// 调用方必须持有读锁。
func (r *InMemoryRepository) listConversationLocked(userID, targetUserID string) []PrivateMessageItem {
	key := conversationKey(userID, targetUserID)
	items := append([]PrivateMessageItem(nil), r.messages[key]...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].SentAt == items[j].SentAt {
			return items[i].MessageID > items[j].MessageID
		}
		return items[i].SentAt > items[j].SentAt
	})
	readCursorID := ""
	if r.readCursor[key] != nil {
		readCursorID = r.readCursor[key][targetUserID]
	}
	readCursorSentAt := ""
	if readCursorID != "" {
		for _, item := range r.messages[key] {
			if item.MessageID == readCursorID {
				readCursorSentAt = item.SentAt
				break
			}
		}
	}
	for i := range items {
		if items[i].Sender.UserID != userID {
			items[i].IsRead = true
			continue
		}
		items[i].IsRead = readCursorSentAt != "" && (items[i].SentAt < readCursorSentAt ||
			(items[i].SentAt == readCursorSentAt && items[i].MessageID <= readCursorID))
	}
	return items
}

func (r *InMemoryRepository) History(_ context.Context, userID string, req ListPrivateMessagesRequest) (PaginatedMessages, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.listConversationLocked(userID, req.TargetUserID)
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	cursorValue := ""
	if req.Pagination != nil {
		cursorValue = req.Pagination.Cursor
	}
	cursorSentAt, cursorMessageID, err := decodeMessageCursor(cursorValue)
	if err != nil {
		return PaginatedMessages{}, err
	}
	if !cursorSentAt.IsZero() {
		filtered := make([]PrivateMessageItem, 0, len(items))
		for _, item := range items {
			sentAt, parseErr := time.Parse(time.RFC3339Nano, item.SentAt)
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

func (r *InMemoryRepository) Recent(_ context.Context, userID, targetUserID string, limit int, deletedAfter *time.Time) (PaginatedMessages, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.listConversationLocked(userID, targetUserID)
	if deletedAfter != nil {
		filtered := items[:0]
		for _, item := range items {
			sentAt, err := time.Parse(time.RFC3339Nano, item.SentAt)
			if err == nil && !sentAt.After(*deletedAfter) {
				continue
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return PaginatedMessages{Items: items}, nil
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

// LastByChatKeys 批量语义与逐 key Last 一致（内存实现无往返成本，循环复用）。
func (r *InMemoryRepository) LastByChatKeys(ctx context.Context, userID string, chatKeys []string, boundaries map[string]*time.Time) (map[string]PrivateMessageItem, error) {
	result := make(map[string]PrivateMessageItem, len(chatKeys))
	for _, chatKey := range chatKeys {
		target, ok := targetOfChatKey(chatKey, userID)
		if !ok {
			continue
		}
		var boundary *time.Time
		if boundaries != nil {
			boundary = boundaries[chatKey]
		}
		item, err := r.Last(ctx, userID, target, boundary)
		if err != nil {
			return nil, err
		}
		if item != nil {
			result[chatKey] = *item
		}
	}
	return result, nil
}

// UnreadCountsByChatKeys 批量语义与逐 key UnreadCount 一致。
func (r *InMemoryRepository) UnreadCountsByChatKeys(ctx context.Context, userID string, chatKeys []string, boundaries map[string]*time.Time) (map[string]int, error) {
	result := make(map[string]int, len(chatKeys))
	for _, chatKey := range chatKeys {
		target, ok := targetOfChatKey(chatKey, userID)
		if !ok {
			continue
		}
		var boundary *time.Time
		if boundaries != nil {
			boundary = boundaries[chatKey]
		}
		count, err := r.UnreadCount(ctx, userID, target, boundary)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			result[chatKey] = count
		}
	}
	return result, nil
}

// targetOfChatKey 从确定性 chat_key 反解对方 user_id（chat_key = 较小 id + ":" +
// 较大 id，与 conversationKey 同规则；user_id 由 idgen 生成不含分隔符）。
// 批量读（LastByChatKeys/UnreadCountsByChatKeys）与 PG 实现一样由调用方保证传入
// 的是本人会话；此处只做 chat_key 反解。
func targetOfChatKey(chatKey, userID string) (string, bool) {
	for _, target := range strings.Split(chatKey, ":") {
		if target != "" && target != userID {
			return target, true
		}
	}
	return "", false
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
		// 撤回消息不计未读（对齐 PG queryUnreadPrivateMessageCount 的 is_recalled = FALSE）。
		if item.RecalledAt != nil {
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

func (r *InMemoryRepository) Recall(_ context.Context, userID string, window time.Duration, req RecallPrivateMessageRequest) (MessageEventMeta, error) {
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
			// window<=0：撤回禁用（对齐 PG 实现，spec chat.md 0=禁用）。
			sentAt, err := time.Parse("2006-01-02T15:04:05.000000000Z", msgs[i].SentAt)
			if err == nil && (window <= 0 || time.Since(sentAt) > window) {
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
	if r.readCursor[key] == nil {
		r.readCursor[key] = map[string]string{}
	}
	currentID := r.readCursor[key][userID]
	currentSentAt := ""
	if currentID != "" {
		for _, item := range msgs {
			if item.MessageID == currentID {
				currentSentAt = item.SentAt
				break
			}
		}
	}
	if req.UpToMessageID != "" {
		var target *PrivateMessageItem
		for i := range msgs {
			if msgs[i].MessageID == req.UpToMessageID {
				target = &msgs[i]
				break
			}
		}
		if target == nil {
			return MessageEventMeta{}, ErrMessageNotFound
		}
		// 已读游标只前进（对齐 PG queryUpsertPrivateMessageReadState 的 WHERE 守卫）：
		// 标记更早消息不得回退游标、不得增加未读数。
		advance := currentSentAt == "" || target.SentAt > currentSentAt ||
			(target.SentAt == currentSentAt && target.MessageID > currentID)
		if advance {
			r.readCursor[key][userID] = target.MessageID
		}
		return MessageEventMeta{TargetUserID: req.TargetUserID, MessageID: target.MessageID, SentAt: target.SentAt}, nil
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
		advance := currentSentAt == "" || latestSentAt > currentSentAt ||
			(latestSentAt == currentSentAt && latestMessageID > currentID)
		if advance {
			r.readCursor[key][userID] = latestMessageID
		}
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

// LockStrangerQuota 内存实现的 no-op：单进程内 CountMessagesByTarget 与 Send
// 均在 r.mu 互斥保护下，无跨请求的 check-then-act 竞态窗口（咨询锁只对 PG
// 并发事务有意义）。
func (r *InMemoryRepository) LockStrangerQuota(_ context.Context, _, _ string) error {
	return nil
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
