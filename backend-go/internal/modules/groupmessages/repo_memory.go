package groupmessages

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu        sync.RWMutex
	items     map[string][]GroupMessageItem
	readCount map[string]map[string]int // groupID -> messageID -> count
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items:     make(map[string][]GroupMessageItem),
		readCount: make(map[string]map[string]int),
	}
}
func (r *InMemoryRepository) Send(_ context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	now := time.Now().UTC()
	msg := GroupMessageItem{
		MessageID: fmt.Sprintf("gm_%d", now.UnixNano()),
		Type:      req.Type,
		Content:   req.Content,
		SentAt:    now.Format(time.RFC3339),
		Mentions:  req.Mentions,
	}
	r.mu.Lock()
	r.items[req.GroupID] = append(r.items[req.GroupID], msg)
	r.mu.Unlock()
	return MessageSent{MessageID: msg.MessageID, SentAt: msg.SentAt}, nil
}
func (r *InMemoryRepository) History(_ context.Context, groupID string) (PaginatedGroupMessages, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]GroupMessageItem(nil), r.items[groupID]...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].SentAt != items[j].SentAt {
			return items[i].SentAt > items[j].SentAt // DESC
		}
		return items[i].MessageID > items[j].MessageID // DESC
	})

	for i := range items {
		if count, ok := r.readCount[groupID][items[i].MessageID]; ok {
			items[i].ReadCount = count
		}
	}
	return PaginatedGroupMessages{Items: items}, nil
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

func (r *InMemoryRepository) Update(_ context.Context, req UpdateGroupMessageRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupID, index, msg, found := r.findMessage(req.MessageID)
	if !found {
		return ErrMessageNotFound
	}
	if msg.RecalledAt != "" {
		return ErrMessageRecalled
	}
	if msg.Type != "text" {
		return ErrMessageNotEditable
	}

	msg.Content = req.Content
	msg.EditedAt = time.Now().UTC().Format(time.RFC3339)
	r.items[groupID][index] = *msg
	return nil
}

func (r *InMemoryRepository) Recall(_ context.Context, req RecallGroupMessageRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupID, index, msg, found := r.findMessage(req.MessageID)
	if !found {
		return ErrMessageNotFound
	}
	if msg.RecalledAt != "" {
		return ErrMessageRecalled
	}

	// 检查撤回时间窗口（2分钟）
	sentTime, err := time.Parse(time.RFC3339, msg.SentAt)
	if err == nil {
		if time.Since(sentTime) > 2*time.Minute {
			return ErrMessageRecallWindow
		}
	}

	msg.RecalledAt = time.Now().UTC().Format(time.RFC3339)
	r.items[groupID][index] = *msg
	return nil
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

func (r *InMemoryRepository) markMessageRead(groupID, messageID string) {
	if r.readCount[groupID] == nil {
		r.readCount[groupID] = make(map[string]int)
	}
	r.readCount[groupID][messageID]++
}
