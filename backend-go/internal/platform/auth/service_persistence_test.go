package auth

import (
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

type failingSessionStore struct{ err error }

func (s failingSessionStore) Save(SessionRecord) error                         { return s.err }
func (s failingSessionStore) Touch(string) error                               { return s.err }
func (s failingSessionStore) ListActiveByUser(string) ([]SessionRecord, error) { return nil, s.err }
func (s failingSessionStore) Revoke(string) (bool, error)                      { return false, s.err }
func (s failingSessionStore) RevokeOthers(string, string) (int, error)         { return 0, s.err }
func (s failingSessionStore) Get(string) (SessionRecord, bool, error) {
	return SessionRecord{}, false, s.err
}

type rotateFailRefreshStore struct {
	current RefreshSession
	err     error
}

func (s *rotateFailRefreshStore) Save(session RefreshSession) error {
	s.current = session
	return nil
}
func (s *rotateFailRefreshStore) Get(token string) (RefreshSession, bool, error) {
	if s.current.RefreshToken == token {
		return s.current, true, nil
	}
	return RefreshSession{}, false, nil
}
func (s *rotateFailRefreshStore) Rotate(string, RefreshSession, RefreshSession) error { return s.err }

func TestIssueSessionReturnsStoreWriteError(t *testing.T) {
	service := NewServiceWithStores(config.Config{AuthTokenSecret: "test-secret"}, failingSessionStore{err: errors.New("session save failed")}, NewInMemoryRefreshStore())
	if _, err := service.IssueSession("user_alice"); err == nil {
		t.Fatal("expected issue session error")
	}
}

func TestRefreshSessionReturnsRotateError(t *testing.T) {
	refreshes := &rotateFailRefreshStore{current: RefreshSession{UserID: "user_alice", SessionID: "sess_1", RefreshToken: "refresh_1", ExpiresAt: time.Now().UTC().Add(time.Hour)}, err: errors.New("refresh rotate failed")}
	service := NewServiceWithStores(config.Config{AuthTokenSecret: "test-secret"}, NewInMemorySessionStore(), refreshes)
	now := time.Now().UTC()
	if err := service.sessions.Save(SessionRecord{SessionID: "sess_1", UserID: "user_alice", CreatedAt: now, LastActiveAt: now}); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if _, err := service.RefreshSession("refresh_1"); err == nil {
		t.Fatal("expected refresh error")
	}
}
