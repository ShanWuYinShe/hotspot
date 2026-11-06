package groupmessages

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/types"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[string][]GroupMessageItem
	// readState 记录每个成员在某群的已读位点（groupID -> userID -> 位点），
	// 已读计数按位点动态计算（PG 同口径：last_read_sent_at >= 消息 sent_at
	// 且 joined_at <= 消息 sent_at），而非每次 mark 累加计数器。
	readState map[string]map[string]readPosition
	// memberJoined 记录成员加入时间（count 口径需要 joined_at <= sent_at）。
	memberJoined map[string]map[string]time.Time
	groupNames   map[string]string // groupID -> name（测试可注入）
	// members 记录已发送过消息的成员（发送即视为成员，测试可预置）。
	members map[string]map[string]bool
	// mutedUntil 记录群成员的禁言截止时间（groupID::userID → RFC3339），
	// 测试经 SetMemberMuted 注入。
	mutedUntil map[string]string
	// seq 保证同纳秒内连续发送的消息 ID 不碰撞（UnixNano 时间戳在快速
	// 循环下会重复，导致消息覆盖与历史分页缺失）。
	seq uint64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items:        make(map[string][]GroupMessageItem),
		readState:    make(map[string]map[string]readPosition),
		memberJoined: make(map[string]map[string]time.Time),
		groupNames:   make(map[string]string),
		members:      make(map[string]map[string]bool),
		mutedUntil:   make(map[string]string),
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
func (r *InMemoryRepository) Send(_ context.Context, req SendGroupMessageRequest) (MessageSentResult, error) {
	now := time.Now().UTC()
	r.mu.Lock()
	// seq 自增必须在锁内：锁外并发 Send 对同一字段竞态写（-race 必报）。
	r.seq++
	// 固定 9 位纳秒（不用 RFC3339Nano：尾零截断会让等值时间输出不同长度的
	// 字符串，History 的字符串排序与游标比较随之失序）。
	mentions := req.Mentions
	if mentions == nil {
		mentions = []string{}
	}
	msg := GroupMessageItem{
		MessageID:    fmt.Sprintf("gm_%d_%d", now.UnixNano(), r.seq),
		SenderUserID: req.SenderUserID,
		Type:         req.Type,
		Content:      types.StrPtr(req.Content),
		SentAt:       now.Format("2006-01-02T15:04:05.000000000Z"),
		Mentions:     mentions,
	}
	if req.ShareCode != "" {
		msg.ShareCode = types.StrPtr(req.ShareCode)
	}
	r.items[req.GroupID] = append(r.items[req.GroupID], msg)
	if r.members[req.GroupID] == nil {
		r.members[req.GroupID] = make(map[string]bool)
	}
	r.members[req.GroupID][req.SenderUserID] = true
	if r.memberJoined[req.GroupID] == nil {
		r.memberJoined[req.GroupID] = make(map[string]time.Time)
	}
	if _, ok := r.memberJoined[req.GroupID][req.SenderUserID]; !ok {
		r.memberJoined[req.GroupID][req.SenderUserID] = now
	}
	r.mu.Unlock()
	return MessageSentResult{MessageID: msg.MessageID, SentAt: msg.SentAt}, nil
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
		if items[i].RecalledAt != nil {
			// 已撤回消息 read_count 恒 null（spec：null = 不适用）。
			continue
		}
		count := r.countReadersLocked(groupID, items[i])
		items[i].ReadCount = &count
	}
	if len(items) <= limit {
		return PaginatedGroupMessages{Items: items}, nil
	}
	next := items[limit-1]
	c := encodeGroupMessageCursor(next.SentAt, next.MessageID)
	return PaginatedGroupMessages{Items: items[:limit], NextCursor: &c}, nil
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

// GetMessageReportSnapshot 内存版群消息快照（file_id 内存 Send 不落 Media，
// 媒体引用缺省为空）。
func (r *InMemoryRepository) GetMessageReportSnapshot(_ context.Context, messageID string) (MessageReportSnapshot, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	groupID, _, msg, found := r.findMessage(messageID)
	if !found {
		return MessageReportSnapshot{}, false, nil
	}
	content := ""
	if msg.Content != nil {
		content = *msg.Content
	}
	fileID := ""
	if msg.Media != nil {
		if id, ok := msg.Media["file_id"].(string); ok {
			fileID = id
		}
	}
	return MessageReportSnapshot{GroupID: groupID, Type: msg.Type, Content: content, FileID: fileID, SentAt: msg.SentAt}, true, nil
}

func (r *InMemoryRepository) Update(_ context.Context, req UpdateGroupMessageRequest) (MessageEventMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupID, index, msg, found := r.findMessage(req.MessageID)
	if !found {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if msg.RecalledAt != nil {
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

	msg.Content = types.StrPtr(req.Content)
	editedAt := time.Now().UTC().Format(time.RFC3339Nano)
	msg.EditedAt = &editedAt
	r.items[groupID][index] = *msg
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: msg.SentAt, Content: req.Content}, nil
}

func (r *InMemoryRepository) Recall(_ context.Context, window time.Duration, req RecallGroupMessageRequest) (MessageEventMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupID, index, msg, found := r.findMessage(req.MessageID)
	if !found {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if msg.RecalledAt != nil {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	// Strict ownership: only the sender may recall. Messages with an empty
	// sender (legacy/anonymous) are not recallable by anyone.
	if msg.SenderUserID == "" || msg.SenderUserID != req.UserID {
		return MessageEventMeta{}, ErrPermissionDenied
	}
	// 撤回操作者落库（作者撤回即作者本人），History 的 recalled_by 与实时
	// 事件口径一致（spec api_spec_chat.md GroupMessageItem.recalled_by）。
	msg.RecalledBy = map[string]any{"user_id": req.UserID}

	// 检查撤回时间窗（service 从 AppConfig 取值传入）；window<=0 = 撤回禁用。
	sentTime, err := time.Parse(time.RFC3339Nano, msg.SentAt)
	if err == nil {
		if window <= 0 || time.Since(sentTime) > window {
			return MessageEventMeta{}, ErrMessageRecallWindow
		}
	}

	recalledAt := time.Now().UTC().Format(time.RFC3339Nano)
	msg.RecalledAt = &recalledAt
	r.items[groupID][index] = *msg
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: msg.SentAt}, nil
}

// RecallAsAdmin recalls a message regardless of sender, skipping the owner and
// recall-window checks (admin path, see spec api_spec_group.md). Only
// not-found and already-recalled conflicts are returned.
func (r *InMemoryRepository) RecallAsAdmin(_ context.Context, messageID, operatorID string) (MessageEventMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	groupID, index, msg, found := r.findMessage(messageID)
	if !found {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if msg.RecalledAt != nil {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	recalledAt := time.Now().UTC().Format(time.RFC3339Nano)
	msg.RecalledAt = &recalledAt
	msg.RecalledBy = map[string]any{"user_id": operatorID}
	r.items[groupID][index] = *msg
	return MessageEventMeta{GroupID: groupID, MessageID: messageID, SentAt: msg.SentAt}, nil
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, userID string, req MarkGroupMessagesReadRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.members[req.GroupID][userID] {
		// PG 的 up_to 查询 JOIN group_members：非成员 0 行 → 10300；无 up_to 的
		// 批量 upsert WHERE members.user_id = $1 → 0 行，属于成功 no-op。
		if req.UpToMessageID == "" {
			return nil
		}
		return ErrMessageNotFound
	}
	if req.UpToMessageID == "" {
		latest, ok := r.latestMessageLocked(req.GroupID)
		if !ok {
			return nil
		}
		sentAt, err := time.Parse(time.RFC3339Nano, latest.SentAt)
		if err != nil {
			return nil
		}
		r.upsertReadPositionLocked(req.GroupID, userID, latest.MessageID, sentAt)
		return nil
	}
	for _, msg := range r.items[req.GroupID] {
		if msg.MessageID != req.UpToMessageID {
			continue
		}
		sentAt, err := time.Parse(time.RFC3339Nano, msg.SentAt)
		if err != nil {
			return ErrMessageNotFound
		}
		r.upsertReadPositionLocked(req.GroupID, userID, msg.MessageID, sentAt)
		return nil
	}
	return ErrMessageNotFound
}
func (r *InMemoryRepository) MarkAllRead(_ context.Context, userID string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// PG 返回 DISTINCT group_members.group_id（用户所在全部群，含暂无消息的群）；
	// 此前返回「有消息的全部群」，把非成员群也算进去了（契约测试发现）。
	groupIDs := make([]string, 0, len(r.members))
	for groupID, members := range r.members {
		if !members[userID] {
			continue
		}
		groupIDs = append(groupIDs, groupID)
		latest, ok := r.latestMessageLocked(groupID)
		if !ok {
			continue
		}
		sentAt, err := time.Parse(time.RFC3339Nano, latest.SentAt)
		if err != nil {
			continue
		}
		r.upsertReadPositionLocked(groupID, userID, latest.MessageID, sentAt)
	}
	sort.Strings(groupIDs)
	return groupIDs, nil
}
func (r *InMemoryRepository) GetReadCount(_ context.Context, groupID string, messageID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, msg := range r.items[groupID] {
		if msg.MessageID == messageID {
			return r.countReadersLocked(groupID, msg), nil
		}
	}
	// PG 单条查询对不存在消息返回 COUNT=0（标量子查询 NULL → 条件不成立），
	// 不是 10300；此前内存实现返回 ErrMessageNotFound（契约测试发现）。
	return 0, nil
}
func (r *InMemoryRepository) GetReadCounts(_ context.Context, groupID string, messageIDs []string) (map[string]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	counts := make(map[string]int, len(messageIDs))
	byID := make(map[string]GroupMessageItem, len(r.items[groupID]))
	for _, msg := range r.items[groupID] {
		byID[msg.MessageID] = msg
	}
	for _, id := range messageIDs {
		msg, ok := byID[id]
		if !ok {
			continue
		}
		counts[id] = r.countReadersLocked(groupID, msg)
	}
	return counts, nil
}
func (r *InMemoryRepository) LatestMessageID(_ context.Context, groupID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	messages := r.items[groupID]
	if len(messages) == 0 {
		return "", ErrMessageNotFound
	}
	latest := messages[0]
	for _, msg := range messages[1:] {
		if msg.SentAt > latest.SentAt || (msg.SentAt == latest.SentAt && msg.MessageID > latest.MessageID) {
			latest = msg
		}
	}
	return latest.MessageID, nil
}

func (r *InMemoryRepository) FilterMemberIDs(_ context.Context, groupID string, userIDs []string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	memberSet := r.members[groupID]
	out := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if memberSet[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

func (r *InMemoryRepository) IsMember(_ context.Context, groupID, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.members[groupID][userID], nil
}

func (r *InMemoryRepository) MemberMuteStatus(_ context.Context, groupID, userID string) (bool, string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.members[groupID][userID] {
		return false, "", false, nil
	}
	until := r.mutedUntil[groupID+"::"+userID]
	if until != "" && until <= time.Now().UTC().Format(time.RFC3339) {
		return false, "", true, nil
	}
	return until != "", until, true, nil
}

// SetMemberMuted 供测试注入禁言状态（生产 mute 权威在 groups 模块）。
func (r *InMemoryRepository) SetMemberMuted(groupID, userID, mutedUntil string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if mutedUntil == "" {
		delete(r.mutedUntil, groupID+"::"+userID)
		return
	}
	r.mutedUntil[groupID+"::"+userID] = mutedUntil
}

// readPosition 是成员在某群的已读位点（PG group_message_read_state 的内存等价）。
type readPosition struct {
	messageID string
	sentAt    time.Time
}

func readPositionAfter(a, b readPosition) bool {
	if a.sentAt.Equal(b.sentAt) {
		return a.messageID > b.messageID
	}
	return a.sentAt.After(b.sentAt)
}

// upsertReadPositionLocked 只在位点前进时写入（PG upsert 的
// last_read_sent_at/ID 比较守卫），避免迟到的旧 up_to 把已读位点回退。
func (r *InMemoryRepository) upsertReadPositionLocked(groupID, userID, messageID string, sentAt time.Time) {
	if r.readState[groupID] == nil {
		r.readState[groupID] = make(map[string]readPosition)
	}
	next := readPosition{messageID: messageID, sentAt: sentAt}
	if cur, ok := r.readState[groupID][userID]; ok && !readPositionAfter(next, cur) {
		return
	}
	r.readState[groupID][userID] = next
}

func (r *InMemoryRepository) latestMessageLocked(groupID string) (GroupMessageItem, bool) {
	messages := r.items[groupID]
	if len(messages) == 0 {
		return GroupMessageItem{}, false
	}
	latest := messages[0]
	for _, msg := range messages[1:] {
		if readPositionAfter(readPosition{messageID: msg.MessageID, sentAt: parseSentAt(msg.SentAt)}, readPosition{messageID: latest.MessageID, sentAt: parseSentAt(latest.SentAt)}) {
			latest = msg
		}
	}
	return latest, true
}

func parseSentAt(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

// countReadersLocked 按 PG queryGroupMessageReadCounts 口径计算已读成员数：
// 成员 joined_at <= 消息 sent_at 且 last_read_sent_at >= 消息 sent_at。
func (r *InMemoryRepository) countReadersLocked(groupID string, msg GroupMessageItem) int {
	msgAt := parseSentAt(msg.SentAt)
	if msgAt.IsZero() {
		return 0
	}
	count := 0
	for userID := range r.members[groupID] {
		joinedAt := r.memberJoined[groupID][userID]
		if joinedAt.IsZero() || joinedAt.After(msgAt) {
			continue
		}
		pos, ok := r.readState[groupID][userID]
		if !ok || pos.sentAt.Before(msgAt) {
			continue
		}
		count++
	}
	return count
}

// SeedMember 供测试/降级模式预置群成员（生产成员权威在 groups 模块的
// group_members 表；内存实现无法跨模块读取，仅测试注入）。
func (r *InMemoryRepository) SeedMember(groupID, userID string, joinedAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.members[groupID] == nil {
		r.members[groupID] = make(map[string]bool)
	}
	if r.memberJoined[groupID] == nil {
		r.memberJoined[groupID] = make(map[string]time.Time)
	}
	r.members[groupID][userID] = true
	r.memberJoined[groupID][userID] = joinedAt.UTC()
}
func (r *InMemoryRepository) GetMessageForReply(_ context.Context, messageID string) (*MessageBrief, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, _, msg, found := r.findMessage(messageID)
	if !found {
		return nil, false, nil
	}
	return &MessageBrief{
		MessageID: messageID,
		Type:      msg.Type,
		Content:   msg.Content,
		ShareCode: msg.ShareCode,
		Sender:    types.UserInfo{UserID: msg.SenderUserID, Nickname: msg.SenderUserID},
		SentAt:    msg.SentAt,
	}, true, nil
}

func (r *InMemoryRepository) MessageExists(_ context.Context, messageID string) (bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, _, msg, found := r.findMessage(messageID)
	if !found {
		return false, "", nil
	}
	return true, msg.SenderUserID, nil
}

func (r *InMemoryRepository) MessageGroupID(_ context.Context, messageID string) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	groupID, _, msg, found := r.findMessage(messageID)
	if !found {
		return "", false, nil
	}
	// 撤回消息按不存在处理（与 PG 侧 is_recalled = FALSE 同口径，审计 F-11）：
	// 该方法仅被 IsReactionTargetVisible 消费，撤回后不应再接受 reaction。
	if msg.RecalledAt != nil {
		return "", false, nil
	}
	return groupID, true, nil
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

func (r *InMemoryRepository) GetMediaForMessage(_ context.Context, _ string, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}
