package auth

import (
	"context"
	"sort"
	"sync"
	"time"
)

// sessionIdleTTL caps how long an untouched session lingers in the
// in-memory fallback store. Sessions older than this are lazily expired on
// read and dropped by the background sweeper. Without it the dev/fallback
// store grows for the process lifetime; in production we use the Postgres
// store so this value only matters for tests and local dev.
const sessionIdleTTL = 24 * time.Hour

// sessionRevokedRetention caps how long a revoked session is kept around
// before being eligible for eviction. The grace window exists so
// post-revocation accounting (e.g. list-revoked-sessions for audit) still
// works shortly after revoke.
const sessionRevokedRetention = 24 * time.Hour

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

func (s *InMemorySessionStore) Save(_ context.Context, session SessionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.SessionID] = session
	return nil
}

func (s *InMemorySessionStore) Touch(_ context.Context, sessionID string) error {
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

func (s *InMemorySessionStore) ListActiveByUser(_ context.Context, userID string) ([]SessionRecord, error) {
	now := time.Now().UTC()
	idleCutoff := now.Add(-sessionIdleTTL)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []SessionRecord
	for _, item := range s.items {
		if item.UserID != userID || item.RevokedAt != nil {
			continue
		}
		// Lazy expiry: drop untouched sessions on read so the map can drain
		// even without the background sweeper.
		if item.LastActiveAt.Before(idleCutoff) {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastActiveAt.Equal(out[j].LastActiveAt) {
			return out[i].SessionID < out[j].SessionID
		}
		return out[i].LastActiveAt.After(out[j].LastActiveAt)
	})
	return out, nil
}

func (s *InMemorySessionStore) Revoke(_ context.Context, sessionID string) (bool, error) {
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

func (s *InMemorySessionStore) RevokeOthers(_ context.Context, userID, currentSessionID string) (int, error) {
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

func (s *InMemorySessionStore) Get(_ context.Context, sessionID string) (SessionRecord, bool, error) {
	now := time.Now().UTC()
	idleCutoff := now.Add(-sessionIdleTTL)
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[sessionID]
	if !ok {
		return SessionRecord{}, false, nil
	}
	// Lazy expiry: a session untouched past sessionIdleTTL is reported as
	// gone even before the sweeper physically removes it. Revoked sessions
	// are still returned so callers can observe the revocation reason.
	if session.RevokedAt == nil && session.LastActiveAt.Before(idleCutoff) {
		return SessionRecord{}, false, nil
	}
	return session, true, nil
}

// Sweep drops entries that are either long-idle or long-revoked. It is
// intended to be called by a background goroutine; see StartSweeper.
func (s *InMemorySessionStore) Sweep(now time.Time) {
	idleCutoff := now.Add(-sessionIdleTTL)
	revokedCutoff := now.Add(-sessionRevokedRetention)
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.items {
		if session.RevokedAt != nil {
			if session.RevokedAt.Before(revokedCutoff) {
				delete(s.items, id)
			}
			continue
		}
		if session.LastActiveAt.Before(idleCutoff) {
			delete(s.items, id)
		}
	}
}
