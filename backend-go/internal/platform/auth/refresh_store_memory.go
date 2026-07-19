package auth

import (
	"context"
	"sync"
	"time"
)

// refreshGraceRetention caps how long a rotated token stays in the store
// after its grace window closes. Tokens in their grace window must remain so
// the rotation can be detected; once grace is over they are eligible for
// eviction.
const refreshGraceRetention = 5 * time.Minute

type InMemoryRefreshStore struct {
	mu    sync.RWMutex
	items map[string]RefreshSession
}

func NewInMemoryRefreshStore() *InMemoryRefreshStore {
	return &InMemoryRefreshStore{items: make(map[string]RefreshSession)}
}

func (s *InMemoryRefreshStore) Save(_ context.Context, session RefreshSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.RefreshToken] = session
	return nil
}

func (s *InMemoryRefreshStore) Get(_ context.Context, token string) (RefreshSession, bool, error) {
	now := time.Now().UTC()
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[token]
	if !ok {
		return RefreshSession{}, false, nil
	}
	// Lazy expiry: a token past its ExpiresAt (and outside its rotation
	// grace window) is treated as missing so the map drains naturally even
	// without the background sweeper running.
	if now.After(session.ExpiresAt) {
		if session.GraceUntil.IsZero() || now.After(session.GraceUntil.Add(refreshGraceRetention)) {
			return RefreshSession{}, false, nil
		}
	}
	return session, true, nil
}

func (s *InMemoryRefreshStore) Rotate(_ context.Context, oldToken string, current RefreshSession, next RefreshSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current.RotatedTo = next.RefreshToken
	current.GraceUntil = time.Now().UTC().Add(30 * time.Second)
	s.items[oldToken] = current
	s.items[next.RefreshToken] = next
	return nil
}

// Sweep removes tokens whose ExpiresAt is in the past and whose rotation
// grace window has elapsed. Intended to be called from a background sweeper.
func (s *InMemoryRefreshStore) Sweep(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, session := range s.items {
		if now.After(session.ExpiresAt) {
			if session.GraceUntil.IsZero() || now.After(session.GraceUntil.Add(refreshGraceRetention)) {
				delete(s.items, token)
			}
		}
	}
}
