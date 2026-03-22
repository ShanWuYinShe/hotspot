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

func (s *InMemoryRefreshStore) Save(session RefreshSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.RefreshToken] = session
	return nil
}
func (s *InMemoryRefreshStore) Get(token string) (RefreshSession, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[token]
	return session, ok, nil
}
func (s *InMemoryRefreshStore) Rotate(oldToken string, current RefreshSession, next RefreshSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current.RotatedTo = next.RefreshToken
	current.GraceUntil = time.Now().UTC().Add(30 * time.Second)
	s.items[oldToken] = current
	s.items[next.RefreshToken] = next
	return nil
}
