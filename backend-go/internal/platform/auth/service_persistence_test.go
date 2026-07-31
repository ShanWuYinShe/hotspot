package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

type failingSessionStore struct{ err error }

func (s failingSessionStore) Save(context.Context, SessionRecord) error { return s.err }
func (s failingSessionStore) Touch(context.Context, string) error       { return s.err }
func (s failingSessionStore) ListActiveByUser(context.Context, string) ([]SessionRecord, error) {
	return nil, s.err
}
func (s failingSessionStore) Revoke(context.Context, string) (bool, error) { return false, s.err }
func (s failingSessionStore) RevokeOthers(context.Context, string, string) (int, error) {
	return 0, s.err
}
func (s failingSessionStore) Get(context.Context, string) (SessionRecord, bool, error) {
	return SessionRecord{}, false, s.err
}

type rotateFailRefreshStore struct {
	current RefreshSession
	err     error
}

func (s *rotateFailRefreshStore) Save(_ context.Context, session RefreshSession) error {
	s.current = session
	return nil
}
func (s *rotateFailRefreshStore) Get(_ context.Context, token string) (RefreshSession, bool, error) {
	if s.current.RefreshToken == token {
		return s.current, true, nil
	}
	return RefreshSession{}, false, nil
}
func (s *rotateFailRefreshStore) Rotate(context.Context, string, RefreshSession, RefreshSession) error {
	return s.err
}

func TestIssueSessionReturnsStoreWriteError(t *testing.T) {
	service := NewServiceWithStores(config.Config{AuthTokenSecret: "test-secret"}, failingSessionStore{err: errors.New("session save failed")}, NewInMemoryRefreshStore())
	if _, err := service.IssueSession(context.Background(), "user_alice"); err == nil {
		t.Fatal("expected issue session error")
	}
}

func TestIssueSessionWithMetaPersistsSessionMetadata(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	issued, err := service.IssueSessionWithMeta(context.Background(), "user_alice", SessionMeta{DeviceType: "app", DeviceName: "HotSpot App", IPAddress: "203.0.113.7", IPLocation: "US"})
	if err != nil {
		t.Fatalf("issue session with meta: %v", err)
	}
	stored, ok, err := service.sessions.Get(context.Background(), issued.Principal.SessionID)
	if err != nil || !ok {
		t.Fatalf("get session: ok=%v err=%v", ok, err)
	}
	if stored.DeviceType != "app" || stored.DeviceName != "HotSpot App" || stored.IPAddress != "203.0.113.7" || stored.IPLocation != "US" {
		t.Fatalf("unexpected stored session: %#v", stored)
	}
}

func TestRefreshSessionReturnsRotateError(t *testing.T) {
	refreshes := &rotateFailRefreshStore{current: RefreshSession{UserID: "user_alice", SessionID: "sess_1", RefreshToken: "refresh_1", ExpiresAt: time.Now().UTC().Add(time.Hour)}, err: errors.New("refresh rotate failed")}
	service := NewServiceWithStores(config.Config{AuthTokenSecret: "test-secret"}, NewInMemorySessionStore(), refreshes)
	now := time.Now().UTC()
	if err := service.sessions.Save(context.Background(), SessionRecord{SessionID: "sess_1", UserID: "user_alice", CreatedAt: now, LastActiveAt: now}); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if _, err := service.RefreshSession(context.Background(), "refresh_1"); err == nil {
		t.Fatal("expected refresh error")
	}
}

func TestRefreshSessionAllowsReplayedTokenWithinGraceWindow(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	issued, err := service.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	first, err := service.RefreshSession(context.Background(), issued.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	second, err := service.RefreshSession(context.Background(), issued.RefreshToken)
	if err != nil {
		t.Fatalf("second refresh in grace window: %v", err)
	}
	if second.RefreshToken != first.RefreshToken {
		t.Fatalf("expected rotated refresh token reuse within grace window, got %q want %q", second.RefreshToken, first.RefreshToken)
	}
	if second.Principal.SessionID != first.Principal.SessionID {
		t.Fatalf("expected same session id, got %q want %q", second.Principal.SessionID, first.Principal.SessionID)
	}
}

func TestRefreshSessionRejectsReplayedTokenAfterGraceWindow(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	issued, err := service.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	if _, err := service.RefreshSession(context.Background(), issued.RefreshToken); err != nil {
		t.Fatalf("refresh session: %v", err)
	}
	stored, ok, err := service.refreshes.Get(context.Background(), issued.RefreshToken)
	if err != nil || !ok {
		t.Fatalf("get old refresh session: ok=%v err=%v", ok, err)
	}
	stored.GraceUntil = time.Now().UTC().Add(-time.Second)
	memoryStore := service.refreshes.(*InMemoryRefreshStore)
	memoryStore.mu.Lock()
	memoryStore.items[issued.RefreshToken] = stored
	memoryStore.mu.Unlock()

	_, err = service.RefreshSession(context.Background(), issued.RefreshToken)
	if err == nil || err.Error() != "expired refresh rotation window" {
		t.Fatalf("expected expired rotation window, got %v", err)
	}
}

func TestRefreshSessionRejectsRevokedSession(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	issued, err := service.IssueSession(context.Background(), "user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	claims, err := service.tokens.ParseAccessToken(issued.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if _, err := service.sessions.Revoke(context.Background(), claims.SessionID); err != nil {
		t.Fatalf("revoke session: %v", err)
	}

	_, err = service.RefreshSession(context.Background(), issued.RefreshToken)
	if err == nil || err.Error() != "revoked session" {
		t.Fatalf("expected revoked session error, got %v", err)
	}
}

func TestListSessionsTouchesCurrentSessionWhenPresent(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	now := time.Now().UTC().Add(-time.Minute)
	if err := service.sessions.Save(context.Background(), SessionRecord{SessionID: "sess_old", UserID: "user_alice", CreatedAt: now, LastActiveAt: now}); err != nil {
		t.Fatalf("save old session: %v", err)
	}
	if err := service.sessions.Save(context.Background(), SessionRecord{SessionID: "sess_current", UserID: "user_alice", CreatedAt: now, LastActiveAt: now}); err != nil {
		t.Fatalf("save current session: %v", err)
	}

	before, ok, err := service.sessions.Get(context.Background(), "sess_current")
	if err != nil || !ok {
		t.Fatalf("get current session: ok=%v err=%v", ok, err)
	}
	listed := service.ListSessions(context.Background(), "user_alice", "sess_current")
	if len(listed) != 2 {
		t.Fatalf("expected 2 sessions, got %#v", listed)
	}
	after, ok, err := service.sessions.Get(context.Background(), "sess_current")
	if err != nil || !ok {
		t.Fatalf("get current session after list: ok=%v err=%v", ok, err)
	}
	if !after.LastActiveAt.After(before.LastActiveAt) {
		t.Fatalf("expected current session to be touched, before=%v after=%v", before.LastActiveAt, after.LastActiveAt)
	}
}
