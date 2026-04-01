package privatechats

import (
	"context"
	"sync"
	"time"
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
func (r *InMemoryRepository) List(_ context.Context, userID string, _ ListOptions) ([]ChatState, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ChatState, 0)
	for _, chat := range r.chats {
		if state, ok := chat.users[userID]; ok {
			out = append(out, cloneChatState(state))
		}
	}
	return out, "", nil
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
