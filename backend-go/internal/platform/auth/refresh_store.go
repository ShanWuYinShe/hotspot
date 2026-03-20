package auth

import (
	"sync"
	"time"
)

type RefreshSession struct {
	UserID       string
	SessionID    string
	RefreshToken string
	RotatedTo    string
	GraceUntil   time.Time
	ExpiresAt    time.Time
}

type RefreshStore struct {
	mu    sync.RWMutex
	items map[string]RefreshSession
}

func NewRefreshStore() *RefreshStore {
	return &RefreshStore{items: make(map[string]RefreshSession)}
}

func (s *RefreshStore) Save(session RefreshSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.RefreshToken] = session
}

func (s *RefreshStore) Get(token string) (RefreshSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[token]
	return session, ok
}

func (s *RefreshStore) Rotate(oldToken string, current RefreshSession, next RefreshSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current.RotatedTo = next.RefreshToken
	current.GraceUntil = time.Now().UTC().Add(30 * time.Second)
	s.items[oldToken] = current
	s.items[next.RefreshToken] = next
}
