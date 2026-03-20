package auth

import (
	"sync"
	"time"
)

type InMemoryRefreshStore struct {
	mu    sync.RWMutex
	items map[string]RefreshSession
}

func NewInMemoryRefreshStore() *InMemoryRefreshStore {
	return &InMemoryRefreshStore{items: make(map[string]RefreshSession)}
}

func (s *InMemoryRefreshStore) Save(session RefreshSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.RefreshToken] = session
}
func (s *InMemoryRefreshStore) Get(token string) (RefreshSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[token]
	return session, ok
}
func (s *InMemoryRefreshStore) Rotate(oldToken string, current RefreshSession, next RefreshSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current.RotatedTo = next.RefreshToken
	current.GraceUntil = time.Now().UTC().Add(30 * time.Second)
	s.items[oldToken] = current
	s.items[next.RefreshToken] = next
}
