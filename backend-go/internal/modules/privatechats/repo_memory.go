package privatechats

import (
	"context"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/types"
)

type memoryChatState struct {
	createdAt time.Time
	users     map[string]ChatState
}

type InMemoryRepository struct {
	mu    sync.RWMutex
	chats map[string]*memoryChatState
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{chats: make(map[string]*memoryChatState)}
}
func key(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}
func (r *InMemoryRepository) GetOrCreate(_ context.Context, userID, targetUserID string) (time.Time, error) {
	k := key(userID, targetUserID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.chats[k]; ok {
		return state.createdAt, nil
	}
	now := time.Now().UTC()
	r.chats[k] = &memoryChatState{createdAt: now, users: map[string]ChatState{userID: {ChatKey: k, TargetID: targetUserID, CreatedAt: now}, targetUserID: {ChatKey: k, TargetID: userID, CreatedAt: now}}}
	return now, nil
}
func (r *InMemoryRepository) List(_ context.Context, userID string, opts ListOptions) ([]ChatState, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ChatState, 0)
	for _, chat := range r.chats {
		if state, ok := chat.users[userID]; ok {
			out = append(out, cloneChatState(state))
		}
	}
	return paginateChatStates(out, opts)
}

func (r *InMemoryRepository) GetState(_ context.Context, userID, targetUserID string) (ChatState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if chat, ok := r.chats[key(userID, targetUserID)]; ok {
		if state, ok := chat.users[userID]; ok {
			return cloneChatState(state), nil
		}
	}
	return ChatState{ChatKey: key(userID, targetUserID), TargetID: targetUserID}, nil
}

func (r *InMemoryRepository) RestoreForUser(_ context.Context, userID, targetUserID string, visibleSince time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	chat := r.ensureChatLocked(userID, targetUserID)
	state := chat.users[userID]
	ts := visibleSince.UTC()
	state.VisibleSince = &ts
	state.DeletedAt = nil
	chat.users[userID] = state
	return nil
}

func (r *InMemoryRepository) UpdateSettings(_ context.Context, userID, targetUserID string, update UpdateChatSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	chat := r.ensureChatLocked(userID, targetUserID)
	state := chat.users[userID]
	if update.IsMuted != nil {
		state.IsMuted = *update.IsMuted
	}
	if update.IsPinned != nil {
		state.IsPinned = *update.IsPinned
	}
	chat.users[userID] = state
	return nil
}

func (r *InMemoryRepository) DeleteForUser(_ context.Context, userID, targetUserID string, deletedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	chat := r.ensureChatLocked(userID, targetUserID)
	state := chat.users[userID]
	if deletedAt != nil {
		ts := deletedAt.UTC()
		state.DeletedAt = &ts
	}
	chat.users[userID] = state
	return nil
}

func (r *InMemoryRepository) ensureChatLocked(userID, targetUserID string) *memoryChatState {
	k := key(userID, targetUserID)
	if chat, ok := r.chats[k]; ok {
		if _, ok := chat.users[userID]; !ok {
			chat.users[userID] = ChatState{ChatKey: k, TargetID: targetUserID, CreatedAt: chat.createdAt}
		}
		if _, ok := chat.users[targetUserID]; !ok {
			chat.users[targetUserID] = ChatState{ChatKey: k, TargetID: userID, CreatedAt: chat.createdAt}
		}
		return chat
	}
	now := time.Now().UTC()
	chat := &memoryChatState{createdAt: now, users: map[string]ChatState{userID: {ChatKey: k, TargetID: targetUserID, CreatedAt: now}, targetUserID: {ChatKey: k, TargetID: userID, CreatedAt: now}}}
	r.chats[k] = chat
	return chat
}

func cloneChatState(state ChatState) ChatState {
	cloned := state
	if state.VisibleSince != nil {
		ts := state.VisibleSince.UTC()
		cloned.VisibleSince = &ts
	}
	if state.DeletedAt != nil {
		ts := state.DeletedAt.UTC()
		cloned.DeletedAt = &ts
	}
	return cloned
}

// paginateChatStates 会话列表 keyset 分页（审查 P2-7）：与 PG SQL 同序
// （is_pinned DESC, last_activity DESC, target_id ASC），供内存实现与测试桩复用，
// 避免两套分页逻辑漂移。内存侧的 last_activity 在无消息表时回退 created_at。
func paginateChatStates(states []ChatState, opts ListOptions) ([]ChatState, string, error) {
	limit := types.NormalizeLimit(opts.Limit)
	items := make([]ChatState, 0, len(states))
	for _, state := range states {
		if state.LastActivityAt.IsZero() {
			state.LastActivityAt = state.CreatedAt
		}
		items = append(items, state)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsPinned != items[j].IsPinned {
			return items[i].IsPinned
		}
		if !items[i].LastActivityAt.Equal(items[j].LastActivityAt) {
			return items[i].LastActivityAt.After(items[j].LastActivityAt)
		}
		return items[i].TargetID < items[j].TargetID
	})
	cursorPinned, cursorAt, cursorTargetID, err := decodeChatCursor(opts.Cursor)
	if err != nil {
		return nil, "", err
	}
	if !cursorAt.IsZero() {
		start := 0
		for i, state := range items {
			// 跳过游标及之前的项：pinned 更大，或同 pinned 且 activity 更新；
			// 同 (pinned, activity) 时 target 更小者在前。
			if state.IsPinned != cursorPinned {
				if !state.IsPinned && cursorPinned {
					break
				}
				start = i + 1
				continue
			}
			if state.LastActivityAt.Before(cursorAt) || (state.LastActivityAt.Equal(cursorAt) && state.TargetID > cursorTargetID) {
				break
			}
			start = i + 1
		}
		items = items[start:]
	}
	if len(items) > limit {
		last := items[limit-1]
		next := encodeChatCursor(last.IsPinned, last.LastActivityAt.UTC().Format(time.RFC3339Nano), last.TargetID)
		return items[:limit], next, nil
	}
	return items, "", nil
}
