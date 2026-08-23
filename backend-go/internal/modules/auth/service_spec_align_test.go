package auth

import (
	"context"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// sessionEventSink records delivered events and close state for hub-based
// SessionRevoked assertions.
type sessionEventSink struct {
	events []realtime.Event
	closed bool
}

func (s *sessionEventSink) Send(e realtime.Event) error {
	s.events = append(s.events, e)
	return nil
}

func (s *sessionEventSink) Close() error { s.closed = true; return nil }

func sessionRevokedReason(e realtime.Event) string {
	p, ok := e.Payload.(realtime.SessionRevokedPayload)
	if !ok {
		return ""
	}
	return p.Reason
}

func assertSessionRevoked(t *testing.T, sink *sessionEventSink, wantReason string) {
	t.Helper()
	if len(sink.events) != 1 {
		t.Fatalf("expected exactly 1 event, got %+v", sink.events)
	}
	if sink.events[0].Type != realtime.EventTypeSessionRevoked {
		t.Fatalf("expected SessionRevoked, got %s", sink.events[0].Type)
	}
	if got := sessionRevokedReason(sink.events[0]); got != wantReason {
		t.Fatalf("expected reason %q, got %q", wantReason, got)
	}
	if !sink.closed {
		t.Fatal("expected connection to be closed after SessionRevoked")
	}
}

// --- Guest/Login 同设备驱逐旧 session(device_evicted) ---

func TestGuestEvictsOldSessionOnSameDevice(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()

	repo := &stubAuthRepo{guestAccount: Account{UserID: "g1"}}
	svc, _ := newTestService(repo)

	first, err := svc.Guest(context.Background(), GuestRequest{Nickname: types.StrPtr("g")}, defaultMeta())
	if err != nil {
		t.Fatalf("first guest: %v", err)
	}
	firstSink := &sessionEventSink{}
	hub.RegisterUser("g1", first.Principal.SessionID, firstSink)

	// 同一 (user_id, device_id) 再次 Guest → 旧会话被驱逐并推送 SessionRevoked。
	if _, err := svc.Guest(context.Background(), GuestRequest{Nickname: types.StrPtr("g2")}, defaultMeta()); err != nil {
		t.Fatalf("second guest: %v", err)
	}
	assertSessionRevoked(t, firstSink, "device_evicted")
}

func TestLoginEvictsOldSessionOnSameDevice(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()

	repo := &stubAuthRepo{authAccount: Account{UserID: "u1"}}
	svc, _ := newTestService(repo)

	first, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "p"}, defaultMeta())
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	firstSink := &sessionEventSink{}
	hub.RegisterUser("u1", first.Principal.SessionID, firstSink)

	if _, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "p"}, defaultMeta()); err != nil {
		t.Fatalf("second login: %v", err)
	}
	assertSessionRevoked(t, firstSink, "device_evicted")
}

func TestLoginSkipsEvictionForDifferentDevice(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()

	repo := &stubAuthRepo{authAccount: Account{UserID: "u1"}}
	svc, _ := newTestService(repo)

	first, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "p"}, defaultMeta())
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	firstSink := &sessionEventSink{}
	hub.RegisterUser("u1", first.Principal.SessionID, firstSink)

	// 不同 device_id 登录:旧会话保留,不驱逐。
	otherMeta := defaultMeta()
	otherMeta.DeviceID = "other-device"
	if _, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "p"}, otherMeta); err != nil {
		t.Fatalf("login with different device: %v", err)
	}
	if len(firstSink.events) != 0 {
		t.Fatalf("different-device login must not evict, got %+v", firstSink.events)
	}
	if firstSink.closed {
		t.Fatal("different-device login must not close the old connection")
	}
}

// --- ChangePassword / RevokeSession / ResetPassword 事件 ---

func TestChangePasswordRevokesOtherSessionsWithEvent(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()

	repo := &stubAuthRepo{registerAccount: Account{UserID: "u1"}}
	svc, authSvc := newTestService(repo)

	current, err := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com", VerificationCode: types.StrPtr("123456")}, defaultMeta())
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	other, _ := authSvc.IssueSession(context.Background(), current.Principal.UserID)
	otherSink := &sessionEventSink{}
	hub.RegisterUser(current.Principal.UserID, other.Principal.SessionID, otherSink)

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: current.Principal.UserID, SessionID: current.Principal.SessionID})
	if err := svc.ChangePassword(ctx, ChangePasswordRequest{OldPassword: "old", NewPassword: "new"}); err != nil {
		t.Fatalf("change password: %v", err)
	}
	assertSessionRevoked(t, otherSink, "password_changed")
}

func TestRevokeSessionPushesRevokedByUserEvent(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()

	repo := &stubAuthRepo{}
	svc, authSvc := newTestService(repo)

	current, _ := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com", VerificationCode: types.StrPtr("123456")}, defaultMeta())
	other, _ := authSvc.IssueSession(context.Background(), current.Principal.UserID)
	sink := &sessionEventSink{}
	hub.RegisterUser(current.Principal.UserID, other.Principal.SessionID, sink)

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: current.Principal.UserID, SessionID: current.Principal.SessionID})
	if err := svc.RevokeSession(ctx, RevokeSessionRequest{SessionID: other.Principal.SessionID}); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	assertSessionRevoked(t, sink, "revoked_by_user")
}

func TestResetPasswordRevokesAllSessions(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()

	repo := &stubAuthRepo{resetAccount: Account{UserID: "u1"}}
	svc, authSvc := newTestService(repo)

	s1, _ := authSvc.IssueSession(context.Background(), "u1")
	s2, _ := authSvc.IssueSession(context.Background(), "u1")
	sink1 := &sessionEventSink{}
	sink2 := &sessionEventSink{}
	hub.RegisterUser("u1", s1.Principal.SessionID, sink1)
	hub.RegisterUser("u1", s2.Principal.SessionID, sink2)

	if _, err := svc.ResetPassword(context.Background(), ResetPasswordRequest{LoginID: "a@b.com", VerificationCode: "123456", NewPassword: "new"}, defaultMeta()); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	assertSessionRevoked(t, sink1, "password_changed")
	assertSessionRevoked(t, sink2, "password_changed")
}

// --- inMemoryLoginTracker 剩余分支(过期清理 / 已锁不再计数) ---

func TestLoginTrackerCheckCleansExpiredLock(t *testing.T) {
	tracker := newInMemoryLoginTracker(context.Background())
	defer tracker.stop()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		tracker.RecordFailed(ctx, "u@test")
	}
	if err := tracker.Check(ctx, "u@test"); err == nil {
		t.Fatal("expected locked after 5 failures")
	}
	// 让锁过期,Check 必须清理条目并放行。
	tracker.mu.Lock()
	tracker.entries["u@test"].lockUntil = time.Now().Add(-time.Minute)
	tracker.mu.Unlock()

	if err := tracker.Check(ctx, "u@test"); err != nil {
		t.Fatalf("expired lock must pass, got %v", err)
	}
	tracker.mu.Lock()
	_, ok := tracker.entries["u@test"]
	tracker.mu.Unlock()
	if ok {
		t.Fatal("expired entry must be removed by Check")
	}
}

func TestLoginTrackerRecordFailedWhileLockedDoesNotCount(t *testing.T) {
	tracker := newInMemoryLoginTracker(context.Background())
	defer tracker.stop()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		tracker.RecordFailed(ctx, "u@test")
	}
	tracker.RecordFailed(ctx, "u@test") // 已锁定:提前返回,不再计数

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	entry := tracker.entries["u@test"]
	if entry.failures != 5 {
		t.Fatalf("failures must stay 5 while locked, got %d", entry.failures)
	}
}
