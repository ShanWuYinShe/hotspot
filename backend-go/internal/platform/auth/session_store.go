package auth

import (
	"sort"
	"sync"
	"time"
)

type SessionRecord struct {
	SessionID    string
	UserID       string
	CreatedAt    time.Time
	LastActiveAt time.Time
	RevokedAt    *time.Time
	DeviceType   string
	DeviceName   string
	IPAddress    string
	IPLocation   string
}

type InMemorySessionStore struct {
	mu    sync.RWMutex
	items map[string]SessionRecord
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{items: make(map[string]SessionRecord)}
}

func (s *InMemorySessionStore) Save(session SessionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.SessionID] = session
	return nil
}

func (s *InMemorySessionStore) Touch(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.items[sessionID]
	if !ok || session.RevokedAt != nil {
		return nil
	}
	session.LastActiveAt = time.Now().UTC()
	s.items[sessionID] = session
	return nil
}

func (s *InMemorySessionStore) ListActiveByUser(userID string) ([]SessionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []SessionRecord
	for _, item := range s.items {
		if item.UserID == userID && item.RevokedAt == nil {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastActiveAt.Equal(out[j].LastActiveAt) {
			return out[i].SessionID < out[j].SessionID
		}
		return out[i].LastActiveAt.After(out[j].LastActiveAt)
	})
	return out, nil
}

func (s *InMemorySessionStore) Revoke(sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.items[sessionID]
	if !ok || session.RevokedAt != nil {
		return false, nil
	}
	now := time.Now().UTC()
	session.RevokedAt = &now
	s.items[sessionID] = session
	return true, nil
}

func (s *InMemorySessionStore) RevokeOthers(userID, currentSessionID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	now := time.Now().UTC()
	for id, session := range s.items {
		if session.UserID != userID || id == currentSessionID || session.RevokedAt != nil {
			continue
		}
		session.RevokedAt = &now
		s.items[id] = session
		count++
	}
	return count, nil
}

func (s *InMemorySessionStore) Get(sessionID string) (SessionRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[sessionID]
	return session, ok, nil
}
