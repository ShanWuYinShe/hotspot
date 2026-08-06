package groupmessages

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu         sync.RWMutex
	items      map[string][]GroupMessageItem
	readCount  map[string]map[string]int // groupID -> messageID -> count
	groupNames map[string]string         // groupID -> name（测试可注入）
	// seq 保证同纳秒内连续发送的消息 ID 不碰撞（UnixNano 时间戳在快速
	// 循环下会重复，导致消息覆盖与历史分页缺失）。
	seq uint64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items:      make(map[string][]GroupMessageItem),
		readCount:  make(map[string]map[string]int),
		groupNames: make(map[string]string),
	}
}

// SetGroupName 供测试注入群名（群名权威在 groups 模块，内存实现仅测试用）。
func (r *InMemoryRepository) SetGroupName(groupID, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groupNames[groupID] = name
}

func (r *InMemoryRepository) GetGroupName(_ context.Context, groupID string) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.groupNames[groupID]
	return name, ok, nil
}
func (r *InMemoryRepository) Send(_ context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	now := time.Now().UTC()
	r.seq++
	msg := GroupMessageItem{
		MessageID:    fmt.Sprintf("gm_%d_%d", now.UnixNano(), r.seq),
		SenderUserID: req.SenderUserID,
		Type:         req.Type,
		Content:      req.Content,
		SentAt:       now.Format(time.RFC3339),
		Mentions:     req.Mentions,
	}
	r.mu.Lock()
	r.items[req.GroupID] = append(r.items[req.GroupID], msg)
	r.mu.Unlock()
	return MessageSent{MessageID: msg.MessageID, SentAt: msg.SentAt}, nil
}
func (r *InMemoryRepository) History(_ context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]GroupMessageItem(nil), r.items[groupID]...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].SentAt != items[j].SentAt {
			return items[i].SentAt > items[j].SentAt // DESC
		}
		return items[i].MessageID > items[j].MessageID // DESC
	})
	limit := 20
	cursorValue := ""
	if pagination != nil {
		if pagination.Limit > 0 {
			limit = pagination.Limit
		}
		cursorValue = pagination.Cursor
	}
	cursorSentAt, cursorMessageID, err := decodeGroupMessageCursor(cursorValue)
	if err != nil {
		return PaginatedGroupMessages{}, err
	}
	if !cursorSentAt.IsZero() {
		filtered := make([]GroupMessageItem, 0, len(items))
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

	for i := range items {
		if count, ok := r.readCount[groupID][items[i].MessageID]; ok {
			items[i].ReadCount = count
		}
	}
	if len(items) <= limit {
		return PaginatedGroupMessages{Items: items}, nil
	}
	next := items[limit-1]
	return PaginatedGroupMessages{Items: items[:limit], NextCursor: encodeGroupMessageCursor(next.SentAt, next.MessageID)}, nil
}

// findMessage 查找消息并返回其索引和所属群组ID
// 调用者必须持有锁
func (r *InMemoryRepository) findMessage(messageID string) (groupID string, index int, msg *GroupMessageItem, found bool) {
	for gid, messages := range r.items {
		for i := range messages {
			if messages[i].MessageID == messageID {
				return gid, i, &messages[i], true
			}
		}
	}
	return "", -1, nil, false
}

func (r *InMemoryRepository) Update(_ context.Context, req UpdateGroupMessageRequest) (MessageEventMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupID, index, msg, found := r.findMessage(req.MessageID)
	if !found {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if msg.RecalledAt != "" {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	if msg.Type != "text" {
		return MessageEventMeta{}, ErrMessageNotEditable
	}
	// Strict ownership: only the sender may edit. Messages with an empty
	// sender (legacy/anonymous) are not editable by anyone.
	if msg.SenderUserID == "" || msg.SenderUserID != req.UserID {
		return MessageEventMeta{}, ErrPermissionDenied
	}

	msg.Content = req.Content
	msg.EditedAt = time.Now().UTC().Format(time.RFC3339Nano)
	r.items[groupID][index] = *msg
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: msg.SentAt, Content: req.Content}, nil
}

func (r *InMemoryRepository) Recall(_ context.Context, req RecallGroupMessageRequest) (MessageEventMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupID, index, msg, found := r.findMessage(req.MessageID)
	if !found {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if msg.RecalledAt != "" {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	// Strict ownership: only the sender may recall. Messages with an empty
	// sender (legacy/anonymous) are not recallable by anyone.
	if msg.SenderUserID == "" || msg.SenderUserID != req.UserID {
		return MessageEventMeta{}, ErrPermissionDenied
	}

	// 检查撤回时间窗口（2分钟）
	sentTime, err := time.Parse(time.RFC3339Nano, msg.SentAt)
	if err == nil {
		if time.Since(sentTime) > 2*time.Minute {
			return MessageEventMeta{}, ErrMessageRecallWindow
		}
	}

	msg.RecalledAt = time.Now().UTC().Format(time.RFC3339Nano)
	r.items[groupID][index] = *msg
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: msg.SentAt}, nil
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, _ string, req MarkGroupMessagesReadRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	messages, ok := r.items[req.GroupID]
	if !ok {
		return ErrMessageNotFound
	}

	if req.UpToMessageID == "" {
		for _, msg := range messages {
			r.markMessageRead(req.GroupID, msg.MessageID)
		}
		return nil
	}

	var targetTime string
	found := false
	for _, msg := range messages {
		if msg.MessageID == req.UpToMessageID {
			targetTime = msg.SentAt
			found = true
			break
		}
	}
	if !found {
		return ErrMessageNotFound
	}

	for _, msg := range messages {
		if msg.SentAt <= targetTime {
			r.markMessageRead(req.GroupID, msg.MessageID)
		}
	}
	return nil
}

func (r *InMemoryRepository) MarkAllRead(_ context.Context, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for groupID, messages := range r.items {
		for _, msg := range messages {
			r.markMessageRead(groupID, msg.MessageID)
		}
	}
	return nil
}

func (r *InMemoryRepository) IsMember(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}

func (r *InMemoryRepository) MemberMuteStatus(_ context.Context, _ string, _ string) (bool, string, bool, error) {
	return false, "", true, nil
}

func (r *InMemoryRepository) markMessageRead(groupID, messageID string) {
	if r.readCount[groupID] == nil {
		r.readCount[groupID] = make(map[string]int)
	}
	r.readCount[groupID][messageID]++
}

func (r *InMemoryRepository) GetMessageForReply(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (r *InMemoryRepository) MessageExists(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

func (r *InMemoryRepository) LookupMessageMeta(_ context.Context, messageID string) (MessageOwnerMeta, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	groupID, _, msg, found := r.findMessage(messageID)
	if !found {
		return MessageOwnerMeta{}, false, nil
	}
	return MessageOwnerMeta{GroupID: groupID, SenderUserID: msg.SenderUserID}, true, nil
}

func (r *InMemoryRepository) GetMediaForMessage(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}
